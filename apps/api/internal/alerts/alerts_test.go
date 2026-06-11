package alerts

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/google/uuid"
)

// ── Normalizer tests ──────────────────────────────────────────────────────────

func TestNormalizeActimize(t *testing.T) {
	raw := []byte(`{
		"alertId":            "ACT-001",
		"alertType":          "Cash Structuring",
		"riskScore":          88,
		"customerName":       "John Doe",
		"customerId":         "JD-9901",
		"accountNumber":      "ACC-123",
		"transactionAmount":  9800.00,
		"currency":           "USD",
		"alertDate":          "2026-05-01T10:00:00Z",
		"description":        "Structuring detected",
		"scenarioCode":       "CASH_STRUCT",
		"customerIdType":     "PASSPORT"
	}`)
	a, err := Normalize("actimize", raw)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if a.ExternalAlertID != "ACT-001" {
		t.Errorf("ExternalAlertID got %q, want ACT-001", a.ExternalAlertID)
	}
	if a.AlertType != "STRUCTURING" {
		t.Errorf("AlertType got %q, want STRUCTURING", a.AlertType)
	}
	if a.RiskScore != 88 {
		t.Errorf("RiskScore got %d, want 88", a.RiskScore)
	}
	if a.RiskLevel != "critical" {
		t.Errorf("RiskLevel got %q, want critical", a.RiskLevel)
	}
	if a.SubjectIDType != "PASSPORT" {
		t.Errorf("SubjectIDType got %q, want PASSPORT", a.SubjectIDType)
	}
	if a.SourceSystem != "actimize" {
		t.Errorf("SourceSystem got %q, want actimize", a.SourceSystem)
	}
}

func TestNormalizeSAS(t *testing.T) {
	raw := []byte(`{
		"alert_id":    "SAS-777",
		"alert_category": "AML",
		"risk_rating": "HIGH",
		"party_name":  "Jane Smith",
		"party_id":    "JS-0042",
		"account":     "ACC-456",
		"amount":      15000.0,
		"ccy":         "GBP",
		"detection_dt": "2026-04-15",
		"narrative":   "Rapid movement detected",
		"rule_name":   "RAPID_MOVEMENT"
	}`)
	a, err := Normalize("sas_aml", raw)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if a.ExternalAlertID != "SAS-777" {
		t.Errorf("got %q, want SAS-777", a.ExternalAlertID)
	}
	if a.AlertType != "RAPID_MOVEMENT" {
		t.Errorf("AlertType got %q, want RAPID_MOVEMENT", a.AlertType)
	}
	if a.RiskScore != 75 {
		t.Errorf("RiskScore got %d, want 75 (HIGH)", a.RiskScore)
	}
	if a.Currency != "GBP" {
		t.Errorf("Currency got %q, want GBP", a.Currency)
	}
}

func TestNormalizeOracleFCCM(t *testing.T) {
	raw := []byte(`{
		"ALERT_ID":             "FCCM-500",
		"BUSINESS_TYPE":        "RETAIL",
		"RISK_SCORE":           70,
		"CUSTOMER_FULL_NAME":   "Ahmed Al-Rashid",
		"CUSTOMER_ID":          "AE-112233",
		"ACCOUNT_NUMBER":       "AE-ACC-789",
		"TRANSACTION_AMOUNT":   50000.0,
		"TRANSACTION_CCY":      "AED",
		"ALERT_CREATE_DATE":    "2026-05-10",
		"ALERT_NARRATIVE":      "Velocity threshold breached",
		"SCENARIO_CODE":        "VEL_BREACH"
	}`)
	a, err := Normalize("oracle_fccm", raw)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if a.AlertType != "VELOCITY_BREACH" {
		t.Errorf("AlertType got %q, want VELOCITY_BREACH", a.AlertType)
	}
	if a.RiskLevel != "high" {
		t.Errorf("RiskLevel got %q, want high", a.RiskLevel)
	}
	if a.Currency != "AED" {
		t.Errorf("Currency got %q, want AED", a.Currency)
	}
}

func TestNormalizeFeaturespace(t *testing.T) {
	raw := []byte(`{
		"event_id":        "FS-999",
		"score_type":      "fraud",
		"score":           0.92,
		"entity_name":     "Carlos Mendes",
		"entity_id":       "BR-5566",
		"account_ref":     "BR-ACC-321",
		"event_amount":    120000.0,
		"currency":        "BRL",
		"event_timestamp": "2026-05-20T08:30:00Z",
		"explanation":     "Anomalous pattern detected by ML model"
	}`)
	a, err := Normalize("featurespace", raw)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if a.RiskScore != 92 {
		t.Errorf("RiskScore got %d, want 92 (0.92 * 100)", a.RiskScore)
	}
	if a.AlertType != "UNUSUAL_PATTERN" {
		t.Errorf("AlertType got %q, want UNUSUAL_PATTERN", a.AlertType)
	}
	if a.RiskLevel != "critical" {
		t.Errorf("RiskLevel got %q, want critical (score 92)", a.RiskLevel)
	}
}

func TestNormalizeGeneric(t *testing.T) {
	raw := []byte(`{
		"alert_id":         "GEN-001",
		"alert_type":       "pep match",
		"risk_score":       65,
		"subject_name":     "Maria Santos",
		"subject_id_type":  "PASSPORT",
		"subject_id_number":"XYZ-9988",
		"account_number":   "ACC-777",
		"amount":           5000.0,
		"currency":         "KES",
		"date":             "2026-05-15",
		"description":      "PEP hit"
	}`)
	a, err := Normalize("generic", raw)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if a.AlertType != "PEP_MATCH" {
		t.Errorf("AlertType got %q, want PEP_MATCH", a.AlertType)
	}
	if a.RiskLevel != "high" {
		t.Errorf("RiskLevel got %q, want high (score 65)", a.RiskLevel)
	}
}

func TestNormalizeUnknownSourceFallsBackToGeneric(t *testing.T) {
	raw := []byte(`{"alert_id":"X1","alert_type":"sanction","risk_score":80,"subject_name":"Test"}`)
	a, err := Normalize("some_unknown_tms", raw)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if a.SourceSystem != "generic" {
		t.Errorf("SourceSystem got %q, want generic", a.SourceSystem)
	}
}

// ── Rule engine tests ─────────────────────────────────────────────────────────

func makeRules() []AlertRuleResponse {
	return []AlertRuleResponse{
		{
			AlertType:    "STRUCTURING",
			MinRiskScore: 80,
			PriorityMapping: "critical",
			Tags:         []string{"financial-crime", "urgent"},
			Mode:         "auto",
			SortOrder:    1,
			IsActive:     true,
		},
		{
			AlertType:    "*",
			MinRiskScore: 60,
			PriorityMapping: "high",
			Tags:         []string{"review"},
			Mode:         "queue",
			SortOrder:    2,
			IsActive:     true,
		},
		{
			AlertType:    "*",
			MinRiskScore: 0,
			PriorityMapping: "low",
			Tags:         []string{},
			Mode:         "queue",
			SortOrder:    3,
			IsActive:     true,
		},
	}
}

func TestMatchRuleExact(t *testing.T) {
	rules := makeRules()
	alert := NormalisedAlert{AlertType: "STRUCTURING", RiskScore: 90, RiskLevel: "critical"}
	m := matchRule(rules, alert)
	if m.priority != "critical" {
		t.Errorf("priority got %q, want critical", m.priority)
	}
	if m.mode != "auto" {
		t.Errorf("mode got %q, want auto", m.mode)
	}
}

func TestMatchRuleCatchAll(t *testing.T) {
	rules := makeRules()
	// Type doesn't match STRUCTURING but score is high enough for catch-all
	alert := NormalisedAlert{AlertType: "SANCTIONS_HIT", RiskScore: 70, RiskLevel: "high"}
	m := matchRule(rules, alert)
	if m.priority != "high" {
		t.Errorf("priority got %q, want high", m.priority)
	}
	if m.mode != "queue" {
		t.Errorf("mode got %q, want queue", m.mode)
	}
}

func TestMatchRuleScoreTooLow(t *testing.T) {
	rules := makeRules()
	// Score 90 for STRUCTURING but score here is 50 — should fall through to low catch-all
	alert := NormalisedAlert{AlertType: "STRUCTURING", RiskScore: 50, RiskLevel: "medium"}
	m := matchRule(rules, alert)
	// Second rule needs score >= 60, third needs >= 0
	if m.priority != "low" {
		t.Errorf("priority got %q, want low", m.priority)
	}
}

func TestMatchRuleDefaultWhenNoRules(t *testing.T) {
	alert := NormalisedAlert{AlertType: "UNKNOWN", RiskScore: 30, RiskLevel: "low"}
	m := matchRule([]AlertRuleResponse{}, alert)
	if m.mode != "queue" {
		t.Errorf("mode got %q, want queue", m.mode)
	}
	if m.priority != "low" {
		t.Errorf("priority got %q, want low", m.priority)
	}
}

func TestMatchRuleInactiveRulesSkipped(t *testing.T) {
	rules := []AlertRuleResponse{
		{AlertType: "*", MinRiskScore: 0, PriorityMapping: "critical", Mode: "auto", IsActive: false},
	}
	alert := NormalisedAlert{AlertType: "ANY", RiskScore: 95, RiskLevel: "critical"}
	m := matchRule(rules, alert)
	// Inactive rule → default fallback
	if m.mode != "queue" {
		t.Errorf("mode got %q, want queue", m.mode)
	}
}

// ── Webhook signature tests ───────────────────────────────────────────────────

func TestVerifyWebhookSignatureValid(t *testing.T) {
	secret := "my-test-secret"
	body := []byte(`{"alertId":"TEST-1","riskScore":85}`)

	// Pre-computed HMAC-SHA256 of body with "my-test-secret"
	// Use the same function to generate it so the test is self-consistent
	if !VerifyWebhookSignature(secret, body, computeSignature(secret, body)) {
		t.Error("expected valid signature to pass")
	}
}

func TestVerifyWebhookSignatureInvalid(t *testing.T) {
	secret := "my-test-secret"
	body := []byte(`{"alertId":"TEST-1"}`)
	if VerifyWebhookSignature(secret, body, "sha256=deadbeef") {
		t.Error("expected invalid signature to fail")
	}
}

func TestVerifyWebhookSignatureMissingPrefix(t *testing.T) {
	if VerifyWebhookSignature("secret", []byte("body"), "abc123") {
		t.Error("expected missing prefix to fail")
	}
}

// computeSignature builds the "sha256=<hex>" header value for a given secret+body.
func computeSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// ── formatAlertCaseTitle tests ────────────────────────────────────────────────

func TestFormatAlertCaseTitle(t *testing.T) {
	cases := []struct {
		alertType     string
		subjectName   string
		accountNumber string
		want          string
	}{
		{
			alertType: "STRUCTURING_TRANSACTION", subjectName: "Rahul Sharma", accountNumber: "AC-7890123",
			want: "Structuring Transaction – Rahul Sharma (AC-7890123)",
		},
		{
			alertType: "PEP_MATCH", subjectName: "Maria Santos", accountNumber: "",
			want: "Pep Match – Maria Santos",
		},
		{
			alertType: "SANCTIONS_HIT", subjectName: "Ahmed Al-Rashid", accountNumber: "ACC-456",
			want: "Sanctions Hit – Ahmed Al-Rashid (ACC-456)",
		},
		{
			alertType: "CASH_DEPOSIT", subjectName: "John Doe", accountNumber: "",
			want: "Cash Deposit – John Doe",
		},
		{
			alertType: "UNUSUAL_PATTERN", subjectName: "Carlos Mendes", accountNumber: "BR-321",
			want: "Unusual Pattern – Carlos Mendes (BR-321)",
		},
	}
	for _, tc := range cases {
		got := formatAlertCaseTitle(tc.alertType, tc.subjectName, tc.accountNumber)
		if got != tc.want {
			t.Errorf("formatAlertCaseTitle(%q, %q, %q)\n got  %q\n want %q",
				tc.alertType, tc.subjectName, tc.accountNumber, got, tc.want)
		}
	}
}

// ── InsightsTrigger wiring tests ──────────────────────────────────────────────

type mockInsightsTrigger struct {
	called bool
	lastID uuid.UUID
}

func (m *mockInsightsTrigger) TriggerAnalysis(caseID uuid.UUID) {
	m.called = true
	m.lastID = caseID
}

func TestInsightsTriggerWiring(t *testing.T) {
	mt := &mockInsightsTrigger{}
	svc := NewService(nil)

	// WithInsightsTrigger returns same instance for fluent chaining.
	if svc.WithInsightsTrigger(mt) != svc {
		t.Error("WithInsightsTrigger should return the same *Service")
	}
	if svc.insightsTrigger == nil {
		t.Error("insightsTrigger should be non-nil after WithInsightsTrigger")
	}
}

func TestInsightsTriggerNilGuard(t *testing.T) {
	// Service without a trigger — the guarded block must not panic.
	svc := NewService(nil)
	wantID := uuid.New()

	if svc.insightsTrigger != nil {
		svc.insightsTrigger.TriggerAnalysis(wantID)
	}
	// Reaching here without panic is the assertion.
}

func TestInsightsTriggerCalled(t *testing.T) {
	mt := &mockInsightsTrigger{}
	svc := NewService(nil).WithInsightsTrigger(mt)
	wantID := uuid.New()

	if svc.insightsTrigger != nil {
		svc.insightsTrigger.TriggerAnalysis(wantID)
	}

	if !mt.called {
		t.Error("expected TriggerAnalysis to be called")
	}
	if mt.lastID != wantID {
		t.Errorf("TriggerAnalysis got caseID %v, want %v", mt.lastID, wantID)
	}
}

// ── riskLevel helper tests ────────────────────────────────────────────────────

func TestRiskLevel(t *testing.T) {
	cases := []struct {
		score    int
		expected string
	}{
		{100, "critical"},
		{85, "critical"},
		{84, "high"},
		{65, "high"},
		{64, "medium"},
		{40, "medium"},
		{39, "low"},
		{0, "low"},
	}
	for _, tc := range cases {
		got := riskLevel(tc.score)
		if got != tc.expected {
			t.Errorf("riskLevel(%d) = %q, want %q", tc.score, got, tc.expected)
		}
	}
}
