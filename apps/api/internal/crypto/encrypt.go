package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

var (
	ErrKeyNotConfigured = errors.New("ENCRYPTION_KEY environment variable not set")
	ErrDecryptFailed    = errors.New("decryption failed — wrong key or corrupted ciphertext")
	ErrKeyIDNotFound    = errors.New("encryption key not found for key_id")
)

// keyStore caches loaded keys by their key_id (e.g. "k1", "k2").
// Keys are loaded lazily from env vars named ENCRYPTION_KEY_<id> (e.g. ENCRYPTION_KEY_k1).
// The special env var ENCRYPTION_KEY_CURRENT names the active key used for new encryptions.
// ENCRYPTION_KEY (no suffix) is treated as key_id "k1" for backward compatibility.
var keyStore struct {
	sync.RWMutex
	keys      map[string][]byte
	currentID string
	loaded    bool
}

func loadKeys() (map[string][]byte, string, error) {
	keyStore.Lock()
	defer keyStore.Unlock()

	if keyStore.loaded {
		return keyStore.keys, keyStore.currentID, nil
	}

	keys := make(map[string][]byte)

	// Backward-compat: ENCRYPTION_KEY (no suffix) maps to key_id "k1".
	if v := os.Getenv("ENCRYPTION_KEY"); v != "" {
		k, err := decodeKey(v)
		if err != nil {
			return nil, "", fmt.Errorf("ENCRYPTION_KEY: %w", err)
		}
		keys["k1"] = k
	}

	// Load any ENCRYPTION_KEY_<id> vars (e.g. ENCRYPTION_KEY_k1, ENCRYPTION_KEY_k2).
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, "ENCRYPTION_KEY_") {
			continue
		}
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		name := parts[0]
		id := strings.TrimPrefix(name, "ENCRYPTION_KEY_")
		if id == "CURRENT" {
			continue // handled separately below
		}
		k, err := decodeKey(parts[1])
		if err != nil {
			return nil, "", fmt.Errorf("%s: %w", name, err)
		}
		keys[id] = k
	}

	if len(keys) == 0 {
		return nil, "", ErrKeyNotConfigured
	}

	// Determine the active key for new encryptions.
	currentID := os.Getenv("ENCRYPTION_KEY_CURRENT")
	if currentID == "" {
		currentID = "k1" // default to k1 for backward compatibility
	}
	if _, ok := keys[currentID]; !ok {
		return nil, "", fmt.Errorf("ENCRYPTION_KEY_CURRENT=%q but no key with that id is loaded", currentID)
	}

	keyStore.keys = keys
	keyStore.currentID = currentID
	keyStore.loaded = true
	return keys, currentID, nil
}

func decodeKey(raw string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || len(decoded) != 32 {
		if len(raw) == 32 {
			return []byte(raw), nil
		}
		return nil, fmt.Errorf("key must be 32 bytes (or base64-encoded 32 bytes), got %d bytes", len(raw))
	}
	return decoded, nil
}

func keyByID(keyID string) ([]byte, error) {
	keys, _, err := loadKeys()
	if err != nil {
		return nil, err
	}
	keyStore.RLock()
	k, ok := keyStore.keys[keyID]
	keyStore.RUnlock()
	if !ok {
		// Try re-reading keys in case they were hot-loaded (graceful rotation).
		_ = keys
		return nil, fmt.Errorf("%w: %q", ErrKeyIDNotFound, keyID)
	}
	return k, nil
}

// Encrypt encrypts plaintext using the current active key.
// Returns the ciphertext, the key_id used, and any error.
// Format: base64(nonce || ciphertext || GCM tag)
func Encrypt(plaintext string) (ciphertext string, keyID string, err error) {
	keys, currentID, err := loadKeys()
	if err != nil {
		return "", "", err
	}
	key := keys[currentID]

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), currentID, nil
}

// DecryptWithKeyID decrypts ciphertext using the key identified by keyID.
// This is the correct function to call when you have the stored key_id (most callers).
func DecryptWithKeyID(ciphertext, keyID string) (string, error) {
	if keyID == "" {
		keyID = "k1" // backward compat: rows written before key_id column existed
	}
	key, err := keyByID(keyID)
	if err != nil {
		return "", err
	}
	return decryptWithKey(ciphertext, key)
}

// Decrypt decrypts using the current active key. Use only for data known to have
// been encrypted with the current key (e.g., freshly written in the same process).
// For stored ciphertext, prefer DecryptWithKeyID.
func Decrypt(ciphertext string) (string, error) {
	_, currentID, err := loadKeys()
	if err != nil {
		return "", err
	}
	return DecryptWithKeyID(ciphertext, currentID)
}

func decryptWithKey(ciphertext string, key []byte) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", ErrDecryptFailed
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(data) < gcm.NonceSize() {
		return "", ErrDecryptFailed
	}
	nonce, ct := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", ErrDecryptFailed
	}
	return string(plain), nil
}

// EncryptNullable encrypts a nullable string field. Returns sql.NullString.
func EncryptNullable(s string) (encrypted sql.NullString, keyID sql.NullString, err error) {
	if s == "" {
		return sql.NullString{}, sql.NullString{}, nil
	}
	enc, kid, err := Encrypt(s)
	if err != nil {
		return sql.NullString{}, sql.NullString{}, err
	}
	return sql.NullString{String: enc, Valid: true}, sql.NullString{String: kid, Valid: true}, nil
}

// DecryptNullable decrypts a nullable encrypted field using the stored key_id.
func DecryptNullable(encrypted sql.NullString, keyID sql.NullString) (string, error) {
	if !encrypted.Valid || encrypted.String == "" {
		return "", nil
	}
	kid := ""
	if keyID.Valid {
		kid = keyID.String
	}
	return DecryptWithKeyID(encrypted.String, kid)
}

// IsConfigured returns true if at least one encryption key is present and valid.
func IsConfigured() bool {
	_, _, err := loadKeys()
	return err == nil
}
