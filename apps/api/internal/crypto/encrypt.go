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
	"sync"
)

// KeyID identifies the active encryption key so rotations are traceable.
const DefaultKeyID = "k1"

var (
	ErrKeyNotConfigured = errors.New("ENCRYPTION_KEY environment variable not set")
	ErrDecryptFailed    = errors.New("decryption failed — wrong key or corrupted ciphertext")
)

var (
	once   sync.Once
	aesKey []byte
	keyErr error
)

// loadKey reads ENCRYPTION_KEY from the environment once.
// Accepts either a 32-byte raw key or a base64-encoded 32-byte key.
func loadKey() ([]byte, error) {
	once.Do(func() {
		raw := os.Getenv("ENCRYPTION_KEY")
		if raw == "" {
			keyErr = ErrKeyNotConfigured
			return
		}
		// Try base64 decode first; fall back to raw bytes.
		decoded, err := base64.StdEncoding.DecodeString(raw)
		if err != nil || len(decoded) != 32 {
			if len(raw) == 32 {
				aesKey = []byte(raw)
			} else {
				keyErr = fmt.Errorf("ENCRYPTION_KEY must be 32 bytes (or base64-encoded 32 bytes), got %d", len(raw))
			}
			return
		}
		aesKey = decoded
	})
	return aesKey, keyErr
}

// Encrypt encrypts plaintext using AES-256-GCM and returns a base64 string.
// Format: base64(nonce || ciphertext || tag)
func Encrypt(plaintext string) (ciphertext string, keyID string, err error) {
	key, err := loadKey()
	if err != nil {
		return "", "", err
	}
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
	return base64.StdEncoding.EncodeToString(sealed), DefaultKeyID, nil
}

// Decrypt decrypts a base64-encoded AES-256-GCM ciphertext.
func Decrypt(ciphertext string) (string, error) {
	key, err := loadKey()
	if err != nil {
		return "", err
	}
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

// DecryptNullable decrypts a nullable encrypted field. Returns the original string.
func DecryptNullable(encrypted sql.NullString) (string, error) {
	if !encrypted.Valid || encrypted.String == "" {
		return "", nil
	}
	return Decrypt(encrypted.String)
}

// IsConfigured returns true if the encryption key is present and valid.
func IsConfigured() bool {
	_, err := loadKey()
	return err == nil
}
