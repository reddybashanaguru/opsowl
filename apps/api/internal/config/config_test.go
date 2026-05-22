package config

import (
	"encoding/base64"
	"testing"
)

// validKey returns a base64-encoded 32-byte key for use in tests.
func validKey() string {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	return base64.StdEncoding.EncodeToString(key)
}

// baseProdCfg returns a minimal valid production Config.
func baseProdCfg() *Config {
	return &Config{
		StorageAccessKey: "strong-access-key",
		StorageSecretKey: "strong-secret-key-123",
		EncryptionKey:    validKey(),
	}
}

func TestValidateProductionSecrets_Valid(t *testing.T) {
	if err := validateProductionSecrets(baseProdCfg()); err != nil {
		t.Fatalf("expected no error for valid config, got: %v", err)
	}
}

func TestValidateProductionSecrets_MissingEncryptionKey(t *testing.T) {
	cfg := baseProdCfg()
	cfg.EncryptionKey = ""
	err := validateProductionSecrets(cfg)
	if err == nil {
		t.Fatal("expected error for missing ENCRYPTION_KEY")
	}
}

func TestValidateProductionSecrets_InvalidBase64EncryptionKey(t *testing.T) {
	cfg := baseProdCfg()
	cfg.EncryptionKey = "not-valid-base64!!!"
	err := validateProductionSecrets(cfg)
	if err == nil {
		t.Fatal("expected error for non-base64 ENCRYPTION_KEY")
	}
}

func TestValidateProductionSecrets_ShortEncryptionKey(t *testing.T) {
	cfg := baseProdCfg()
	// 16 bytes → AES-128, not AES-256
	short := make([]byte, 16)
	cfg.EncryptionKey = base64.StdEncoding.EncodeToString(short)
	err := validateProductionSecrets(cfg)
	if err == nil {
		t.Fatal("expected error for 16-byte encryption key (need 32)")
	}
}

func TestValidateProductionSecrets_LongEncryptionKey(t *testing.T) {
	cfg := baseProdCfg()
	// 64 bytes → too long
	long := make([]byte, 64)
	cfg.EncryptionKey = base64.StdEncoding.EncodeToString(long)
	err := validateProductionSecrets(cfg)
	if err == nil {
		t.Fatal("expected error for 64-byte encryption key (need exactly 32)")
	}
}

func TestValidateProductionSecrets_WeakMinioAccessKey(t *testing.T) {
	cfg := baseProdCfg()
	cfg.StorageAccessKey = "minioadmin"
	err := validateProductionSecrets(cfg)
	if err == nil {
		t.Fatal("expected error for known-weak STORAGE_ACCESS_KEY 'minioadmin'")
	}
}

func TestValidateProductionSecrets_WeakMinioSecretKey(t *testing.T) {
	cfg := baseProdCfg()
	cfg.StorageSecretKey = "minioadmin123"
	err := validateProductionSecrets(cfg)
	if err == nil {
		t.Fatal("expected error for known-weak STORAGE_SECRET_KEY 'minioadmin123'")
	}
}

func TestValidateProductionSecrets_WeakSecretCaseInsensitive(t *testing.T) {
	cfg := baseProdCfg()
	cfg.StorageAccessKey = "MINIOADMIN" // uppercase variant
	err := validateProductionSecrets(cfg)
	if err == nil {
		t.Fatal("expected error for case-insensitive weak secret match")
	}
}

func TestValidateProductionSecrets_URLSafeBase64Key(t *testing.T) {
	cfg := baseProdCfg()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(255 - i) // values that produce + and / in standard base64
	}
	// Use URL-safe encoding (- and _ instead of + and /)
	cfg.EncryptionKey = base64.URLEncoding.EncodeToString(key)
	if err := validateProductionSecrets(cfg); err != nil {
		t.Fatalf("expected URL-safe base64 key to be accepted, got: %v", err)
	}
}
