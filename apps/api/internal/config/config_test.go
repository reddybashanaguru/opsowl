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

// ── EU jurisdiction defaults ───────────────────────────────────────────────────

func TestEUDefaults_Lithuania(t *testing.T) {
	cfg := euDefaults("lt")

	if cfg.Jurisdiction != "eu" {
		t.Errorf("Jurisdiction: want eu, got %s", cfg.Jurisdiction)
	}
	if cfg.Country != "lt" {
		t.Errorf("Country: want lt, got %s", cfg.Country)
	}
	if cfg.StrRegulatorLabel != "FNTT" {
		t.Errorf("StrRegulatorLabel: want FNTT, got %s", cfg.StrRegulatorLabel)
	}
	if cfg.StrDeadlineDays != 5 {
		t.Errorf("StrDeadlineDays: want 5 (EU default), got %d", cfg.StrDeadlineDays)
	}
	if cfg.LocalCurrency != "EUR" {
		t.Errorf("LocalCurrency: want EUR, got %s", cfg.LocalCurrency)
	}
	if cfg.CurrencySymbol != "€" {
		t.Errorf("CurrencySymbol: want €, got %s", cfg.CurrencySymbol)
	}
	if cfg.CTRThreshold != 10000 {
		t.Errorf("CTRThreshold: want 10000, got %f", cfg.CTRThreshold)
	}
	if cfg.WireThreshold != 1000 {
		t.Errorf("WireThreshold: want 1000, got %f", cfg.WireThreshold)
	}
	if !cfg.GoAMLEnabled {
		t.Error("GoAMLEnabled must be true for EU jurisdiction")
	}
	if !cfg.UBOEnabled {
		t.Error("UBOEnabled must be true for EU jurisdiction (AMLR 2024 Art. 42)")
	}
	if !cfg.GDPREnabled {
		t.Error("GDPREnabled must be true for EU jurisdiction")
	}
	if cfg.UBOThresholdPct != 25 {
		t.Errorf("UBOThresholdPct: want 25 (AMLR 2024 Art. 42), got %d", cfg.UBOThresholdPct)
	}
	if cfg.RetentionYears != 5 {
		t.Errorf("RetentionYears: want 5 (AMLR 2024 Art. 77), got %d", cfg.RetentionYears)
	}
	if cfg.StrLawReference != "EU AMLR 2024 / AMLD6" {
		t.Errorf("StrLawReference: want 'EU AMLR 2024 / AMLD6', got %s", cfg.StrLawReference)
	}
}

func TestEUDefaults_Cyprus(t *testing.T) {
	cfg := euDefaults("cy")

	if cfg.StrRegulatorLabel != "MOKAS" {
		t.Errorf("StrRegulatorLabel: want MOKAS, got %s", cfg.StrRegulatorLabel)
	}
	if cfg.Country != "cy" {
		t.Errorf("Country: want cy, got %s", cfg.Country)
	}
	if cfg.LocalCurrency != "EUR" {
		t.Errorf("LocalCurrency: want EUR, got %s", cfg.LocalCurrency)
	}
	if !cfg.UBOEnabled {
		t.Error("UBOEnabled must be true for Cyprus (CySEC UBO requirements)")
	}
	if !cfg.GDPREnabled {
		t.Error("GDPREnabled must be true for Cyprus (EU member state)")
	}
	if !cfg.GoAMLEnabled {
		t.Error("GoAMLEnabled must be true for Cyprus (MOKAS mandates goAML)")
	}
}

func TestEUDefaults_Greece(t *testing.T) {
	cfg := euDefaults("gr")

	if cfg.StrRegulatorLabel != "Hellenic FIU" {
		t.Errorf("StrRegulatorLabel: want 'Hellenic FIU', got %s", cfg.StrRegulatorLabel)
	}
	if cfg.Country != "gr" {
		t.Errorf("Country: want gr, got %s", cfg.Country)
	}
	if cfg.Jurisdiction != "eu" {
		t.Errorf("Jurisdiction: want eu, got %s", cfg.Jurisdiction)
	}
}

func TestEUDefaults_UnknownCountry_FallsToEUBase(t *testing.T) {
	// Any unrecognised EU country code should still get EU base defaults
	cfg := euDefaults("bg") // Bulgaria — not in switch yet

	if cfg.Jurisdiction != "eu" {
		t.Errorf("Jurisdiction: want eu, got %s", cfg.Jurisdiction)
	}
	if cfg.LocalCurrency != "EUR" {
		t.Errorf("LocalCurrency: want EUR, got %s", cfg.LocalCurrency)
	}
	if !cfg.UBOEnabled {
		t.Error("UBOEnabled must be true even for unlisted EU country")
	}
	if !cfg.GDPREnabled {
		t.Error("GDPREnabled must be true even for unlisted EU country")
	}
	// Generic EU regulator label for unlisted countries
	if cfg.StrRegulatorLabel != "FIU" {
		t.Errorf("StrRegulatorLabel: want 'FIU' for unlisted country, got %s", cfg.StrRegulatorLabel)
	}
}

func TestEUDefaults_IDTypesContainEUResidencePermit(t *testing.T) {
	cfg := euDefaults("lt")
	found := false
	for _, id := range cfg.IDTypes {
		if id == "EU_RESIDENCE_PERMIT" {
			found = true
			break
		}
	}
	if !found {
		t.Error("EU_RESIDENCE_PERMIT must be in IDTypes for EU jurisdiction (third-country nationals)")
	}
}

func TestDeploymentDefaults_EUJurisdiction_ReturnsEUConfig(t *testing.T) {
	t.Setenv("DEPLOYMENT_JURISDICTION", "eu")
	t.Setenv("DEPLOYMENT_COUNTRY", "lt")

	cfg := deploymentDefaults()

	if cfg.Jurisdiction != "eu" {
		t.Errorf("Jurisdiction: want eu, got %s", cfg.Jurisdiction)
	}
	if cfg.StrRegulatorLabel != "FNTT" {
		t.Errorf("StrRegulatorLabel: want FNTT, got %s", cfg.StrRegulatorLabel)
	}
	if !cfg.UBOEnabled {
		t.Error("UBOEnabled must be true when DEPLOYMENT_JURISDICTION=eu")
	}
}

func TestDeploymentDefaults_IndiaJurisdiction_UBODisabledByDefault(t *testing.T) {
	t.Setenv("DEPLOYMENT_JURISDICTION", "india")
	t.Setenv("DEPLOYMENT_COUNTRY", "in")

	cfg := deploymentDefaults()

	if cfg.Jurisdiction != "india" {
		t.Errorf("Jurisdiction: want india, got %s", cfg.Jurisdiction)
	}
	if cfg.UBOEnabled {
		t.Error("UBOEnabled must be false by default for India jurisdiction")
	}
	if cfg.GDPREnabled {
		t.Error("GDPREnabled must be false by default for India jurisdiction")
	}
	if cfg.LocalCurrency != "INR" {
		t.Errorf("LocalCurrency: want INR, got %s", cfg.LocalCurrency)
	}
}

// ── Australia jurisdiction defaults ───────────────────────────────────────────

func TestAUDefaults_SMRTerminology(t *testing.T) {
	cfg := auDefaults()

	if cfg.StrLabel != "SMR" {
		t.Errorf("StrLabel: want SMR (Suspicious Matter Report), got %s", cfg.StrLabel)
	}
	if cfg.StrRegulatorLabel != "AUSTRAC" {
		t.Errorf("StrRegulatorLabel: want AUSTRAC, got %s", cfg.StrRegulatorLabel)
	}
	if cfg.StrLawReference != "AML/CTF Act 2006 (Cth) s.41" {
		t.Errorf("StrLawReference: want 'AML/CTF Act 2006 (Cth) s.41', got %s", cfg.StrLawReference)
	}
}

func TestAUDefaults_CurrencyAndThresholds(t *testing.T) {
	cfg := auDefaults()

	if cfg.LocalCurrency != "AUD" {
		t.Errorf("LocalCurrency: want AUD, got %s", cfg.LocalCurrency)
	}
	if cfg.CurrencySymbol != "A$" {
		t.Errorf("CurrencySymbol: want A$, got %s", cfg.CurrencySymbol)
	}
	if cfg.CTRThreshold != 10000 {
		t.Errorf("CTRThreshold: want 10000 (AML/CTF Act TTR threshold), got %f", cfg.CTRThreshold)
	}
	if cfg.WireThreshold != 10000 {
		t.Errorf("WireThreshold: want 10000 (AML/CTF Act IFTI threshold), got %f", cfg.WireThreshold)
	}
}

func TestAUDefaults_RetentionAndUBO(t *testing.T) {
	cfg := auDefaults()

	if cfg.RetentionYears != 7 {
		t.Errorf("RetentionYears: want 7 (AML/CTF Act s.105), got %d", cfg.RetentionYears)
	}
	if !cfg.UBOEnabled {
		t.Error("UBOEnabled must be true for Australia (AML/CTF Rules Chapter 4)")
	}
	if cfg.UBOThresholdPct != 25 {
		t.Errorf("UBOThresholdPct: want 25, got %d", cfg.UBOThresholdPct)
	}
}

func TestAUDefaults_GoAMLDisabledGDPRDisabled(t *testing.T) {
	cfg := auDefaults()

	// AUSTRAC uses its own online portal — goAML XML is not accepted
	if cfg.GoAMLEnabled {
		t.Error("GoAMLEnabled must be false for Australia (AUSTRAC uses online portal, not goAML)")
	}
	// Australian Privacy Act 1988 — not GDPR
	if cfg.GDPREnabled {
		t.Error("GDPREnabled must be false for Australia (Privacy Act 1988, not GDPR)")
	}
}

func TestAUDefaults_JurisdictionAndCountry(t *testing.T) {
	cfg := auDefaults()

	if cfg.Jurisdiction != "au" {
		t.Errorf("Jurisdiction: want au, got %s", cfg.Jurisdiction)
	}
	if cfg.Country != "au" {
		t.Errorf("Country: want au, got %s", cfg.Country)
	}
}

func TestAUDefaults_IDTypesContainAustralianDocuments(t *testing.T) {
	cfg := auDefaults()

	required := []string{"PASSPORT", "AUSTRALIAN_DRIVERS_LICENCE", "MEDICARE_CARD"}
	for _, want := range required {
		found := false
		for _, id := range cfg.IDTypes {
			if id == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("IDTypes must contain %s for Australian jurisdiction", want)
		}
	}
}

func TestDeploymentDefaults_AUJurisdiction_ReturnsSMRConfig(t *testing.T) {
	t.Setenv("DEPLOYMENT_JURISDICTION", "au")
	t.Setenv("DEPLOYMENT_COUNTRY", "au")

	cfg := deploymentDefaults()

	if cfg.Jurisdiction != "au" {
		t.Errorf("Jurisdiction: want au, got %s", cfg.Jurisdiction)
	}
	if cfg.StrLabel != "SMR" {
		t.Errorf("StrLabel: want SMR, got %s", cfg.StrLabel)
	}
	if cfg.LocalCurrency != "AUD" {
		t.Errorf("LocalCurrency: want AUD, got %s", cfg.LocalCurrency)
	}
	if cfg.GoAMLEnabled {
		t.Error("GoAMLEnabled must be false when DEPLOYMENT_JURISDICTION=au")
	}
	if !cfg.UBOEnabled {
		t.Error("UBOEnabled must be true when DEPLOYMENT_JURISDICTION=au")
	}
}
