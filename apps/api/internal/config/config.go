package config

import (
	"fmt"
	"os"
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
}

func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return &Config{
		Port:              getEnv("PORT", "8080"),
		Environment:       getEnv("ENVIRONMENT", "development"),
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
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
