package config

import (
	"testing"
)

func TestUKDefaults_SARTerminology(t *testing.T) {
	cfg := ukDefaults()
	if cfg.StrLabel != "SAR" {
		t.Errorf("expected StrLabel=SAR, got %s", cfg.StrLabel)
	}
	if cfg.StrRegulatorLabel != "NCA / UKFIU" {
		t.Errorf("expected StrRegulatorLabel='NCA / UKFIU', got %s", cfg.StrRegulatorLabel)
	}
	if cfg.StrReferenceLabel != "NCA Reference" {
		t.Errorf("expected StrReferenceLabel='NCA Reference', got %s", cfg.StrReferenceLabel)
	}
}

func TestUKDefaults_POCALawReference(t *testing.T) {
	cfg := ukDefaults()
	if cfg.StrLawReference != "Proceeds of Crime Act 2002 (POCA) s.330" {
		t.Errorf("expected POCA law reference, got %s", cfg.StrLawReference)
	}
}

func TestUKDefaults_DAMLDeadline7Days(t *testing.T) {
	cfg := ukDefaults()
	// POCA s.335: 7 working days for DAML consent SAR decisions
	if cfg.StrDeadlineDays != 7 {
		t.Errorf("expected StrDeadlineDays=7, got %d", cfg.StrDeadlineDays)
	}
}

func TestUKDefaults_GBPCurrencyAndThresholds(t *testing.T) {
	cfg := ukDefaults()
	if cfg.LocalCurrency != "GBP" {
		t.Errorf("expected LocalCurrency=GBP, got %s", cfg.LocalCurrency)
	}
	if cfg.CurrencySymbol != "£" {
		t.Errorf("expected CurrencySymbol=£, got %s", cfg.CurrencySymbol)
	}
	if cfg.CTRThreshold != 10000 {
		t.Errorf("expected CTRThreshold=10000, got %f", cfg.CTRThreshold)
	}
	if cfg.WireThreshold != 1000 {
		t.Errorf("expected WireThreshold=1000, got %f", cfg.WireThreshold)
	}
}

func TestUKDefaults_UBOAndGDPREnabled(t *testing.T) {
	cfg := ukDefaults()
	// UK GDPR retained post-Brexit
	if !cfg.GDPREnabled {
		t.Error("expected GDPREnabled=true — UK GDPR applies post-Brexit")
	}
	// Companies Act 2006 PSC register requires 25% UBO disclosure
	if !cfg.UBOEnabled {
		t.Error("expected UBOEnabled=true — UK PSC register requires UBO identification")
	}
	if cfg.UBOThresholdPct != 25 {
		t.Errorf("expected UBOThresholdPct=25, got %d", cfg.UBOThresholdPct)
	}
}

func TestUKDefaults_GoAMLDisabled(t *testing.T) {
	cfg := ukDefaults()
	// UK SARs filed via NCA SARs Online portal (not UNODC goAML)
	if cfg.GoAMLEnabled {
		t.Error("expected GoAMLEnabled=false — UK uses NCA SARs Online, not goAML portal")
	}
}

func TestUKDefaults_RetentionFiveYears(t *testing.T) {
	cfg := ukDefaults()
	if cfg.RetentionYears != 5 {
		t.Errorf("expected RetentionYears=5, got %d", cfg.RetentionYears)
	}
}

func TestUKDefaults_IDTypesContainUKDocs(t *testing.T) {
	cfg := ukDefaults()
	types := make(map[string]bool)
	for _, t := range cfg.IDTypes {
		types[t] = true
	}
	required := []string{"PASSPORT", "UK_DRIVING_LICENCE", "NI_NUMBER", "COMPANIES_HOUSE_NUMBER"}
	for _, r := range required {
		if !types[r] {
			t.Errorf("expected UK ID type %s in config", r)
		}
	}
}

func TestUKDefaults_JurisdictionAndCountry(t *testing.T) {
	cfg := ukDefaults()
	if cfg.Jurisdiction != "uk" {
		t.Errorf("expected Jurisdiction=uk, got %s", cfg.Jurisdiction)
	}
	if cfg.Country != "gb" {
		t.Errorf("expected Country=gb (ISO 3166-1 alpha-2 for UK), got %s", cfg.Country)
	}
}

func TestDeploymentDefaults_UKJurisdiction(t *testing.T) {
	t.Setenv("DEPLOYMENT_JURISDICTION", "uk")
	cfg := deploymentDefaults()
	if cfg.StrLabel != "SAR" {
		t.Errorf("expected SAR label for uk jurisdiction, got %s", cfg.StrLabel)
	}
	if cfg.Jurisdiction != "uk" {
		t.Errorf("expected jurisdiction=uk, got %s", cfg.Jurisdiction)
	}
	if cfg.LocalCurrency != "GBP" {
		t.Errorf("expected GBP currency for uk jurisdiction, got %s", cfg.LocalCurrency)
	}
}
