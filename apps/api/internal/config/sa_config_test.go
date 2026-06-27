package config

import (
	"testing"
)

func TestSADefaults_SARTerminology(t *testing.T) {
	cfg := saDefaults()
	if cfg.StrLabel != "SAR" {
		t.Errorf("expected StrLabel=SAR, got %s", cfg.StrLabel)
	}
	if cfg.StrRegulatorLabel != "SAMA / SAFIU" {
		t.Errorf("expected StrRegulatorLabel='SAMA / SAFIU', got %s", cfg.StrRegulatorLabel)
	}
	if cfg.StrReferenceLabel != "SAFIU Reference" {
		t.Errorf("expected StrReferenceLabel='SAFIU Reference', got %s", cfg.StrReferenceLabel)
	}
}

func TestSADefaults_SAMADeadline(t *testing.T) {
	cfg := saDefaults()
	// SAMA requires SAR filing within 3 business days of suspicion
	if cfg.StrDeadlineDays != 3 {
		t.Errorf("expected StrDeadlineDays=3, got %d", cfg.StrDeadlineDays)
	}
}

func TestSADefaults_CurrencyAndThresholds(t *testing.T) {
	cfg := saDefaults()
	if cfg.LocalCurrency != "SAR" {
		t.Errorf("expected LocalCurrency=SAR, got %s", cfg.LocalCurrency)
	}
	// SAMA cash reporting threshold: SAR 60,000
	if cfg.CTRThreshold != 60000 {
		t.Errorf("expected CTRThreshold=60000, got %f", cfg.CTRThreshold)
	}
}

func TestSADefaults_RetentionTenYears(t *testing.T) {
	cfg := saDefaults()
	// SAMA AML Regulation Article 16: 10-year retention
	if cfg.RetentionYears != 10 {
		t.Errorf("expected RetentionYears=10, got %d", cfg.RetentionYears)
	}
}

func TestSADefaults_UBOEnabled25Pct(t *testing.T) {
	cfg := saDefaults()
	if !cfg.UBOEnabled {
		t.Error("expected UBOEnabled=true for Saudi (SAMA requires UBO identification)")
	}
	if cfg.UBOThresholdPct != 25 {
		t.Errorf("expected UBOThresholdPct=25, got %d", cfg.UBOThresholdPct)
	}
}

func TestSADefaults_GoAMLEnabled(t *testing.T) {
	cfg := saDefaults()
	// SAFIU-Connect is built on UNODC goAML — same XML schema
	if !cfg.GoAMLEnabled {
		t.Error("expected GoAMLEnabled=true — SAFIU uses UNODC goAML format")
	}
}

func TestSADefaults_GDPRDisabled(t *testing.T) {
	cfg := saDefaults()
	if cfg.GDPREnabled {
		t.Error("expected GDPREnabled=false — Saudi uses PDPL not EU GDPR")
	}
}

func TestSADefaults_IDTypesContainSaudiDocs(t *testing.T) {
	cfg := saDefaults()
	types := make(map[string]bool)
	for _, t := range cfg.IDTypes {
		types[t] = true
	}
	required := []string{"SAUDI_NATIONAL_ID", "IQAMA", "PASSPORT", "COMMERCIAL_REGISTRATION"}
	for _, r := range required {
		if !types[r] {
			t.Errorf("expected ID type %s in Saudi config", r)
		}
	}
}

func TestSADefaults_JurisdictionAndCountry(t *testing.T) {
	cfg := saDefaults()
	if cfg.Jurisdiction != "sa" {
		t.Errorf("expected Jurisdiction=sa, got %s", cfg.Jurisdiction)
	}
	if cfg.Country != "sa" {
		t.Errorf("expected Country=sa, got %s", cfg.Country)
	}
}

func TestDeploymentDefaults_SAJurisdiction(t *testing.T) {
	t.Setenv("DEPLOYMENT_JURISDICTION", "sa")
	cfg := deploymentDefaults()
	if cfg.StrLabel != "SAR" {
		t.Errorf("expected SAR label for sa jurisdiction, got %s", cfg.StrLabel)
	}
	if cfg.Jurisdiction != "sa" {
		t.Errorf("expected jurisdiction=sa, got %s", cfg.Jurisdiction)
	}
}

func TestDeploymentDefaults_SaudiAlias(t *testing.T) {
	t.Setenv("DEPLOYMENT_JURISDICTION", "saudi")
	cfg := deploymentDefaults()
	if cfg.StrLabel != "SAR" {
		t.Errorf("expected SAR label for 'saudi' jurisdiction alias, got %s", cfg.StrLabel)
	}
}
