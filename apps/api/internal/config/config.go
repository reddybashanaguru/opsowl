package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port        string
	Environment string

	DatabaseURL string

	JWTPublicKeyPath  string
	JWTPrivateKeyPath string
	JWTAccessExpiry   string
	JWTRefreshExpiry  string

	StorageEndpoint  string
	StorageAccessKey string
	StorageSecretKey string
	StorageBucket    string
	StorageUseSSL    bool

	AIServiceURL    string
	AnthropicAPIKey string
	OpenAIAPIKey    string

	// LDAP / Active Directory (all optional — empty URL disables LDAP)
	LDAPUrl          string
	LDAPBaseDN       string
	LDAPUserAttr     string
	LDAPBindDN       string
	LDAPBindPassword string

	// Security hardening
	EncryptionKey      string // AES-256 column encryption key (base64)
	SIEMWebhookURL     string // optional SIEM/SOAR webhook endpoint
	SIEMFormat         string // json | cef
	SMTPHost           string
	SMTPPort           int
	SMTPUser           string
	SMTPPassword       string
	SMTPFrom           string
	SMTPTLS            bool
	PasswordExpiryDays int
	PasswordHistoryN   int
	PasswordMinLength  int
	PasswordUppercase  bool
	PasswordNumber     bool
	MaxSessionsPerUser int
}

// knownWeakSecrets lists values that must never be used in production.
var knownWeakSecrets = []string{
	"minioadmin", "minioadmin123", "postgres", "password", "secret", "changeme",
	"opsowl123", "OpsOwl@2024",
}

func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	env := getEnv("ENVIRONMENT", "development")

	cfg := &Config{
		Port:              getEnv("PORT", "8080"),
		Environment:       env,
		DatabaseURL:       dbURL,
		JWTPublicKeyPath:  getEnv("JWT_PUBLIC_KEY_PATH", "keys/public.pem"),
		JWTPrivateKeyPath: getEnv("JWT_PRIVATE_KEY_PATH", "keys/private.pem"),
		JWTAccessExpiry:   getEnv("JWT_ACCESS_EXPIRY", "1h"),
		JWTRefreshExpiry:  getEnv("JWT_REFRESH_EXPIRY", "168h"),
		StorageEndpoint:   getEnv("STORAGE_ENDPOINT", "localhost:9000"),
		StorageAccessKey:  getEnv("STORAGE_ACCESS_KEY", "minioadmin"),
		StorageSecretKey:  getEnv("STORAGE_SECRET_KEY", "minioadmin123"),
		StorageBucket:     getEnv("STORAGE_BUCKET", "opsowl"),
		StorageUseSSL:     getEnv("STORAGE_USE_SSL", "false") == "true",
		AIServiceURL:      getEnv("AI_SERVICE_URL", "http://localhost:8000"),
		AnthropicAPIKey:   getEnv("ANTHROPIC_API_KEY", ""),
		OpenAIAPIKey:      getEnv("OPENAI_API_KEY", ""),
		LDAPUrl:           getEnv("LDAP_URL", ""),
		LDAPBaseDN:        getEnv("LDAP_BASE_DN", ""),
		LDAPUserAttr:      getEnv("LDAP_USER_ATTR", "sAMAccountName"),
		LDAPBindDN:        getEnv("LDAP_BIND_DN", ""),
		LDAPBindPassword:  getEnv("LDAP_BIND_PASSWORD", ""),
		EncryptionKey:     getEnv("ENCRYPTION_KEY", ""),
		SIEMWebhookURL:    getEnv("SIEM_WEBHOOK_URL", ""),
		SIEMFormat:        getEnv("SIEM_FORMAT", "json"),
		SMTPHost:          getEnv("SMTP_HOST", ""),
		SMTPPort:          getEnvInt("SMTP_PORT", 587),
		SMTPUser:          getEnv("SMTP_USER", ""),
		SMTPPassword:      getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:          getEnv("SMTP_FROM", "noreply@opsowl.local"),
		SMTPTLS:           getEnv("SMTP_TLS", "true") == "true",
		PasswordExpiryDays: getEnvInt("PASSWORD_EXPIRY_DAYS", 90),
		PasswordHistoryN:   getEnvInt("PASSWORD_HISTORY_COUNT", 5),
		PasswordMinLength:  getEnvInt("PASSWORD_MIN_LENGTH", 8),
		PasswordUppercase:  getEnv("PASSWORD_REQUIRE_UPPERCASE", "true") == "true",
		PasswordNumber:     getEnv("PASSWORD_REQUIRE_NUMBER", "true") == "true",
		MaxSessionsPerUser: getEnvInt("MAX_SESSIONS_PER_USER", 5),
	}

	if env == "production" {
		if err := validateProductionSecrets(cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// validateProductionSecrets fails startup if weak/default credentials are in use.
// This ensures secrets are rotated before deploying to a regulated environment.
func validateProductionSecrets(cfg *Config) error {
	checks := map[string]string{
		"STORAGE_ACCESS_KEY": cfg.StorageAccessKey,
		"STORAGE_SECRET_KEY": cfg.StorageSecretKey,
	}
	for envKey, val := range checks {
		for _, weak := range knownWeakSecrets {
			if strings.EqualFold(val, weak) {
				return fmt.Errorf("SECURITY: %s is set to a known default value ('%s') — rotate before running in production", envKey, val)
			}
		}
	}
	if cfg.EncryptionKey == "" {
		return fmt.Errorf("SECURITY: ENCRYPTION_KEY must be set in production (generate: openssl rand -base64 32)")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
