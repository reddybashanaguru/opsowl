package config

import (
	"testing"
)

// Sweden is EU jurisdiction with country=se.
// Key difference from other EU countries: SEK currency (not EUR — Sweden is not eurozone).

func TestEUDefaults_Sweden_FIPORegulator(t *testing.T) {
	cfg := euDefaults("se")
	if cfg.StrRegulatorLabel != "Finanspolisen (FIPO)" {
		t.Errorf("expected Finanspolisen (FIPO), got %s", cfg.StrRegulatorLabel)
	}
}

func TestEUDefaults_Sweden_PersonnummerIDLabel(t *testing.T) {
	cfg := euDefaults("se")
	if cfg.StrIDLabel != "Personnummer / National ID" {
		t.Errorf("expected Personnummer / National ID, got %s", cfg.StrIDLabel)
	}
}

func TestEUDefaults_Sweden_EUAMLRLawReference(t *testing.T) {
	cfg := euDefaults("se")
	// EU AMLR 2024 applies — same as all EU countries
	if cfg.StrLawReference != "EU AMLR 2024 / AMLD6" {
		t.Errorf("expected EU AMLR 2024 / AMLD6, got %s", cfg.StrLawReference)
	}
}

func TestEUDefaults_Sweden_JurisdictionIsEU(t *testing.T) {
	cfg := euDefaults("se")
	if cfg.Jurisdiction != "eu" {
		t.Errorf("expected Jurisdiction=eu, got %s", cfg.Jurisdiction)
	}
	if cfg.Country != "se" {
		t.Errorf("expected Country=se, got %s", cfg.Country)
	}
}

func TestEUDefaults_Sweden_GoAMLEnabled(t *testing.T) {
	cfg := euDefaults("se")
	// Finanspolisen uses UNODC goAML portal
	if !cfg.GoAMLEnabled {
		t.Error("expected GoAMLEnabled=true — Finanspolisen (FIPO) is a goAML user")
	}
}

func TestEUDefaults_Sweden_UBOAndGDPREnabled(t *testing.T) {
	cfg := euDefaults("se")
	if !cfg.UBOEnabled {
		t.Error("expected UBOEnabled=true — EU AMLA 2024 + Swedish Companies Act")
	}
	if cfg.UBOThresholdPct != 25 {
		t.Errorf("expected UBOThresholdPct=25, got %d", cfg.UBOThresholdPct)
	}
	if !cfg.GDPREnabled {
		t.Error("expected GDPREnabled=true — Swedish DPA (IMY) enforces EU GDPR")
	}
}

func TestEUDefaults_Sweden_DefaultCurrencyIsEUR_OverriddenByPreset(t *testing.T) {
	// Without preset env var, euDefaults returns EUR as base.
	// The eu-sweden.env preset overrides LOCAL_CURRENCY=SEK.
	// This test documents the expected behaviour: preset must set LOCAL_CURRENCY.
	cfg := euDefaults("se")
	// Without env override, base EU currency is EUR
	if cfg.LocalCurrency != "EUR" {
		t.Errorf("expected base LocalCurrency=EUR (preset overrides to SEK), got %s", cfg.LocalCurrency)
	}
}

func TestEUDefaults_Sweden_WithSEKEnvOverride(t *testing.T) {
	// Simulate preset: LOCAL_CURRENCY=SEK CURRENCY_SYMBOL=kr
	t.Setenv("LOCAL_CURRENCY", "SEK")
	t.Setenv("CURRENCY_SYMBOL", "kr")
	t.Setenv("CTR_THRESHOLD", "15000")
	t.Setenv("STR_DEADLINE_DAYS", "1")

	cfg := euDefaults("se")
	if cfg.LocalCurrency != "SEK" {
		t.Errorf("expected LocalCurrency=SEK with env override, got %s", cfg.LocalCurrency)
	}
	if cfg.CurrencySymbol != "kr" {
		t.Errorf("expected CurrencySymbol=kr, got %s", cfg.CurrencySymbol)
	}
	if cfg.CTRThreshold != 15000 {
		t.Errorf("expected CTRThreshold=15000, got %f", cfg.CTRThreshold)
	}
	if cfg.StrDeadlineDays != 1 {
		t.Errorf("expected StrDeadlineDays=1 (utan dröjsmål), got %d", cfg.StrDeadlineDays)
	}
}

func TestDeploymentDefaults_SwedenViaEUJurisdiction(t *testing.T) {
	t.Setenv("DEPLOYMENT_JURISDICTION", "eu")
	t.Setenv("DEPLOYMENT_COUNTRY", "se")
	t.Setenv("LOCAL_CURRENCY", "SEK")

	cfg := deploymentDefaults()
	if cfg.Jurisdiction != "eu" {
		t.Errorf("expected jurisdiction=eu, got %s", cfg.Jurisdiction)
	}
	if cfg.StrRegulatorLabel != "Finanspolisen (FIPO)" {
		t.Errorf("expected Finanspolisen (FIPO), got %s", cfg.StrRegulatorLabel)
	}
	if cfg.LocalCurrency != "SEK" {
		t.Errorf("expected SEK from env override, got %s", cfg.LocalCurrency)
	}
}
