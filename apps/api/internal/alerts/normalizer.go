package alerts

import (
	"encoding/json"
	"strings"
	"time"
)

// Normalize converts a raw TMS payload into OpsOwl's canonical NormalisedAlert.
// source is one of: actimize, sas_aml, oracle_fccm, featurespace, generic
func Normalize(source string, raw []byte) (NormalisedAlert, error) {
	switch strings.ToLower(source) {
	case "actimize":
		return normalizeActimize(raw)
	case "sas_aml", "sas":
		return normalizeSAS(raw)
	case "oracle_fccm", "fccm":
		return normalizeOracleFCCM(raw)
	case "featurespace":
		return normalizeFeaturespace(raw)
	default:
		return normalizeGeneric(raw)
	}
}

// ── NICE Actimize ─────────────────────────────────────────────────────────────

type actimizePayload struct {
	AlertID          string  `json:"alertId"`
	AlertType        string  `json:"alertType"`
	RiskScore        int     `json:"riskScore"`
	CustomerName     string  `json:"customerName"`
	CustomerID       string  `json:"customerId"`
	AccountNumber    string  `json:"accountNumber"`
	TransactionAmt   float64 `json:"transactionAmount"`
	Currency         string  `json:"currency"`
	AlertDate        string  `json:"alertDate"`
	Description      string  `json:"description"`
	ScenarioCode     string  `json:"scenarioCode"`
	CustomerIDType   string  `json:"customerIdType"`
}

func normalizeActimize(raw []byte) (NormalisedAlert, error) {
	var p actimizePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return NormalisedAlert{}, err
	}
	alertType := mapActimizeScenario(p.ScenarioCode, p.AlertType)
	date, _ := time.Parse("2006-01-02T15:04:05Z", p.AlertDate)
	idType := p.CustomerIDType
	if idType == "" {
		idType = "NATIONAL_ID"
	}
	return NormalisedAlert{
		ExternalAlertID: p.AlertID,
		AlertType:       alertType,
		RiskScore:       p.RiskScore,
		RiskLevel:       riskLevel(p.RiskScore),
		SubjectName:     p.CustomerName,
		SubjectIDType:   idType,
		SubjectIDNumber: p.CustomerID,
		AccountNumber:   p.AccountNumber,
		Amount:          p.TransactionAmt,
		Currency:        p.Currency,
		TransactionDate: date,
		Description:     p.Description,
		SourceSystem:    "actimize",
	}, nil
}

func mapActimizeScenario(code, fallback string) string {
	m := map[string]string{
		"CASH_STRUCT":   "STRUCTURING",
		"CASH_STRUCT_001": "STRUCTURING",
		"RAPID_MOVE":    "RAPID_MOVEMENT",
		"VEL_BREACH":    "VELOCITY_BREACH",
		"HI_RISK_GEO":   "HIGH_RISK_GEOGRAPHY",
		"SANC_HIT":      "SANCTIONS_HIT",
		"PEP_HIT":       "PEP_MATCH",
		"LAY_001":       "LAYERING",
		"CTR_THRESH":    "CTR_THRESHOLD",
		"DORM_REACT":    "DORMANT_REACTIVATION",
		"ML_ANOM":       "UNUSUAL_PATTERN",
	}
	if t, ok := m[strings.ToUpper(code)]; ok {
		return t
	}
	if fallback != "" {
		return strings.ToUpper(strings.ReplaceAll(fallback, " ", "_"))
	}
	return "UNUSUAL_PATTERN"
}

// ── SAS AML ───────────────────────────────────────────────────────────────────

type sasPayload struct {
	AlertID       string  `json:"alert_id"`
	AlertCategory string  `json:"alert_category"`
	RiskRating    string  `json:"risk_rating"` // HIGH, MEDIUM, LOW
	PartyName     string  `json:"party_name"`
	PartyID       string  `json:"party_id"`
	Account       string  `json:"account"`
	Amount        float64 `json:"amount"`
	CCY           string  `json:"ccy"`
	DetectionDt   string  `json:"detection_dt"`
	Narrative     string  `json:"narrative"`
	RuleName      string  `json:"rule_name"`
	PartyIDType   string  `json:"party_id_type"`
}

func normalizeSAS(raw []byte) (NormalisedAlert, error) {
	var p sasPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return NormalisedAlert{}, err
	}
	score := sasRatingToScore(p.RiskRating)
	date, _ := time.Parse("2006-01-02", p.DetectionDt)
	idType := p.PartyIDType
	if idType == "" {
		idType = "NATIONAL_ID"
	}
	return NormalisedAlert{
		ExternalAlertID: p.AlertID,
		AlertType:       mapSASRule(p.RuleName),
		RiskScore:       score,
		RiskLevel:       strings.ToLower(p.RiskRating),
		SubjectName:     p.PartyName,
		SubjectIDType:   idType,
		SubjectIDNumber: p.PartyID,
		AccountNumber:   p.Account,
		Amount:          p.Amount,
		Currency:        p.CCY,
		TransactionDate: date,
		Description:     p.Narrative,
		SourceSystem:    "sas_aml",
	}, nil
}

func sasRatingToScore(r string) int {
	switch strings.ToUpper(r) {
	case "CRITICAL":
		return 90
	case "HIGH":
		return 75
	case "MEDIUM":
		return 50
	case "LOW":
		return 25
	}
	return 50
}

func mapSASRule(rule string) string {
	m := map[string]string{
		"RAPID_MOVEMENT":  "RAPID_MOVEMENT",
		"STRUCTURING":     "STRUCTURING",
		"HIGH_RISK_CNTRY": "HIGH_RISK_GEOGRAPHY",
		"SANCTIONS_MATCH": "SANCTIONS_HIT",
		"PEP_EXPOSURE":    "PEP_MATCH",
		"LAYERING_DETECT": "LAYERING",
		"VELOCITY_RULE":   "VELOCITY_BREACH",
		"DORMANT_ACCT":    "DORMANT_REACTIVATION",
		"ANOMALY_ML":      "UNUSUAL_PATTERN",
	}
	if t, ok := m[strings.ToUpper(rule)]; ok {
		return t
	}
	return "UNUSUAL_PATTERN"
}

// ── Oracle FCCM ───────────────────────────────────────────────────────────────

type oracleFCCMPayload struct {
	AlertID          string  `json:"ALERT_ID"`
	BusinessType     string  `json:"BUSINESS_TYPE"`
	RiskScore        int     `json:"RISK_SCORE"`
	CustomerFullName string  `json:"CUSTOMER_FULL_NAME"`
	CustomerID       string  `json:"CUSTOMER_ID"`
	AccountNumber    string  `json:"ACCOUNT_NUMBER"`
	TxnAmount        float64 `json:"TRANSACTION_AMOUNT"`
	TxnCCY           string  `json:"TRANSACTION_CCY"`
	AlertCreateDate  string  `json:"ALERT_CREATE_DATE"`
	AlertNarrative   string  `json:"ALERT_NARRATIVE"`
	ScenarioCode     string  `json:"SCENARIO_CODE"`
}

func normalizeOracleFCCM(raw []byte) (NormalisedAlert, error) {
	var p oracleFCCMPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return NormalisedAlert{}, err
	}
	date, _ := time.Parse("2006-01-02", p.AlertCreateDate)
	return NormalisedAlert{
		ExternalAlertID: p.AlertID,
		AlertType:       mapFCCMScenario(p.ScenarioCode),
		RiskScore:       p.RiskScore,
		RiskLevel:       riskLevel(p.RiskScore),
		SubjectName:     p.CustomerFullName,
		SubjectIDType:   "NATIONAL_ID",
		SubjectIDNumber: p.CustomerID,
		AccountNumber:   p.AccountNumber,
		Amount:          p.TxnAmount,
		Currency:        p.TxnCCY,
		TransactionDate: date,
		Description:     p.AlertNarrative,
		SourceSystem:    "oracle_fccm",
	}, nil
}

func mapFCCMScenario(code string) string {
	m := map[string]string{
		"VEL_BREACH":   "VELOCITY_BREACH",
		"CASH_STRUCT":  "STRUCTURING",
		"SANC_SCR":     "SANCTIONS_HIT",
		"PEP_SCR":      "PEP_MATCH",
		"LAY_DETECT":   "LAYERING",
		"DORM_ACCT":    "DORMANT_REACTIVATION",
		"TRADE_ML":     "TRADE_BASED_ML",
		"CTR_THRESH":   "CTR_THRESHOLD",
		"ML_ANOMALY":   "UNUSUAL_PATTERN",
	}
	if t, ok := m[strings.ToUpper(code)]; ok {
		return t
	}
	return "UNUSUAL_PATTERN"
}

// ── Featurespace ARIC ─────────────────────────────────────────────────────────

type featurespacePayload struct {
	EventID      string  `json:"event_id"`
	ScoreType    string  `json:"score_type"`
	Score        float64 `json:"score"` // 0.0–1.0
	EntityName   string  `json:"entity_name"`
	EntityID     string  `json:"entity_id"`
	AccountRef   string  `json:"account_ref"`
	EventAmt     float64 `json:"event_amount"`
	CCY          string  `json:"currency"`
	EventTs      string  `json:"event_timestamp"`
	Explanation  string  `json:"explanation"`
}

func normalizeFeaturespace(raw []byte) (NormalisedAlert, error) {
	var p featurespacePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return NormalisedAlert{}, err
	}
	score := int(p.Score * 100)
	date, _ := time.Parse(time.RFC3339, p.EventTs)
	return NormalisedAlert{
		ExternalAlertID: p.EventID,
		AlertType:       "UNUSUAL_PATTERN",
		RiskScore:       score,
		RiskLevel:       riskLevel(score),
		SubjectName:     p.EntityName,
		SubjectIDType:   "NATIONAL_ID",
		SubjectIDNumber: p.EntityID,
		AccountNumber:   p.AccountRef,
		Amount:          p.EventAmt,
		Currency:        p.CCY,
		TransactionDate: date,
		Description:     p.Explanation,
		SourceSystem:    "featurespace",
	}, nil
}

// ── Generic (catch-all for custom TMS / unknown systems) ──────────────────────

type genericPayload struct {
	AlertID         string  `json:"alert_id"`
	AlertType       string  `json:"alert_type"`
	RiskScore       int     `json:"risk_score"`
	RiskLevel       string  `json:"risk_level"`
	SubjectName     string  `json:"subject_name"`
	SubjectIDType   string  `json:"subject_id_type"`
	SubjectIDNumber string  `json:"subject_id_number"`
	AccountNumber   string  `json:"account_number"`
	Amount          float64 `json:"amount"`
	Currency        string  `json:"currency"`
	Date            string  `json:"date"`
	Description     string  `json:"description"`
}

func normalizeGeneric(raw []byte) (NormalisedAlert, error) {
	var p genericPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return NormalisedAlert{}, err
	}
	date, _ := time.Parse("2006-01-02", p.Date)
	if date.IsZero() {
		date, _ = time.Parse(time.RFC3339, p.Date)
	}
	score := p.RiskScore
	level := p.RiskLevel
	if level == "" {
		level = riskLevel(score)
	}
	idType := p.SubjectIDType
	if idType == "" {
		idType = "NATIONAL_ID"
	}
	alertType := strings.ToUpper(strings.ReplaceAll(p.AlertType, " ", "_"))
	if alertType == "" {
		alertType = "UNUSUAL_PATTERN"
	}
	return NormalisedAlert{
		ExternalAlertID: p.AlertID,
		AlertType:       alertType,
		RiskScore:       score,
		RiskLevel:       level,
		SubjectName:     p.SubjectName,
		SubjectIDType:   idType,
		SubjectIDNumber: p.SubjectIDNumber,
		AccountNumber:   p.AccountNumber,
		Amount:          p.Amount,
		Currency:        p.Currency,
		TransactionDate: date,
		Description:     p.Description,
		SourceSystem:    "generic",
	}, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func riskLevel(score int) string {
	switch {
	case score >= 85:
		return "critical"
	case score >= 65:
		return "high"
	case score >= 40:
		return "medium"
	default:
		return "low"
	}
}
