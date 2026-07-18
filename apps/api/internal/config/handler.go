package config

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/reddybashanaguru/opsowl/internal/auth"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

// AppConfig mirrors the JSONB blob stored in system_config.
// Jurisdiction and Country are read-only — set via DEPLOYMENT_JURISDICTION /
// DEPLOYMENT_COUNTRY env vars and injected by the GET handler. They are never
// persisted to the DB so a client cannot override them.
type AppConfig struct {
	AppName            string `json:"app_name"`
	Industry           string `json:"industry"`
	CasePrefix         string `json:"case_prefix"`          // prefix for auto-generated case numbers, e.g. "OWL", "DCN"
	CaseNumberFormat   string `json:"case_number_format"`   // "prefix_year_seq" | "prefix_seq" | "prefix_yearmonth_seq" | "prefix_year_seq_slash"
	CaseLabel          string `json:"case_label"`
	CasesLabel     string `json:"cases_label"`
	TaskLabel      string `json:"task_label"`
	TasksLabel     string `json:"tasks_label"`
	DocumentLabel  string `json:"document_label"`
	DocumentsLabel string `json:"documents_label"`
	RiskScoreLabel string `json:"risk_score_label"`
	RiskLevelLabel string `json:"risk_level_label"`
	SLALabel       string `json:"sla_label"`
	CommsLabel     string `json:"comms_label"`
	AnalystLabel   string `json:"analyst_label"`
	TimelineLabel  string `json:"timeline_label"`
	NoteLabel      string `json:"note_label"`
	NotesLabel     string `json:"notes_label"`

	// Regulatory filing labels — jurisdiction-specific
	StrLabel          string `json:"str_label"`           // "STR" (India/UAE), "SAR" (US/UK)
	StrReferenceLabel string `json:"str_reference_label"` // "FIU Reference", "UAE FIU Reference"
	StrRegulatorLabel string `json:"str_regulator_label"` // "FIU-IND", "UAE FIU", "FinCEN"
	StrLawReference   string `json:"str_law_reference"`   // "PMLA 2002", "Federal Decree-Law No. 20 of 2018"
	StrIDLabel        string `json:"str_id_label"`        // "PAN", "Emirates ID"
	StrDeadlineDays   int    `json:"str_deadline_days"`   // calendar days to file after suspicion arises

	// Currency & reporting thresholds
	LocalCurrency  string  `json:"local_currency"`  // ISO 4217: "INR", "AED", "EUR"
	CurrencySymbol string  `json:"currency_symbol"` // display: "₹", "AED", "€"
	CTRThreshold   float64 `json:"ctr_threshold"`   // cash transaction report threshold
	WireThreshold  float64 `json:"wire_threshold"`  // wire record-keeping threshold

	// Identity document types for this jurisdiction
	IDTypes []string `json:"id_types"`

	// ── Jurisdiction metadata — READ-ONLY, injected from env, never stored in DB ──
	Jurisdiction string `json:"jurisdiction"` // "india" | "uae" | "eu"
	Country      string `json:"country"`      // ISO 3166-1 alpha-2, e.g. "in", "ae", "gr"

	// Feature flags derived from jurisdiction — READ-ONLY
	GoAMLEnabled bool `json:"goaml_enabled"` // true for UAE + EU (goAML XML reporting)
	UBOEnabled   bool `json:"ubo_enabled"`   // true for EU (AMLA 2024 mandatory UBO checks)
	GDPREnabled  bool `json:"gdpr_enabled"`  // true for EU (GDPR right-to-erasure workflow)

	// Compliance parameters — READ-ONLY, env-configurable per deployment
	RetentionYears   int `json:"retention_years"`    // AML record retention (EU/UAE: 5–7 yrs)
	UBOThresholdPct  int `json:"ubo_threshold_pct"`  // beneficial ownership control % (EU: 25)
}

// envOr returns the env var value if set and non-empty, otherwise fallback.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// deploymentDefaults returns jurisdiction-aware defaults.
// Jurisdiction is read from DEPLOYMENT_JURISDICTION (india|uae|eu|<any>).
// Country fine-tunes within the jurisdiction (e.g. DEPLOYMENT_COUNTRY=gr for Greece).
// Unknown jurisdictions (singapore, uk, us, etc.) fall through to genericDefaults(),
// which reads all values from env vars — no code change needed for new markets.
func deploymentDefaults() AppConfig {
	jurisdiction := strings.ToLower(strings.TrimSpace(os.Getenv("DEPLOYMENT_JURISDICTION")))
	country := strings.ToLower(strings.TrimSpace(os.Getenv("DEPLOYMENT_COUNTRY")))
	switch jurisdiction {
	case "uae":
		return uaeDefaults()
	case "eu":
		return euDefaults(country)
	case "india", "":
		return indiaDefaults()
	default:
		return genericDefaults(jurisdiction, country)
	}
}

// genericDefaults handles any jurisdiction not explicitly modelled (Singapore, UK, US, etc.).
// Every value is driven by an env var, so deploying to a new market only requires
// setting the appropriate env vars — zero code changes required.
func genericDefaults(jurisdiction, country string) AppConfig {
	return AppConfig{
		AppName:           "OpsOwl AI",
		Industry:          envOr("INDUSTRY", "financial_services"),
		CasePrefix:        envOr("CASE_PREFIX", "OWL"),
		CaseNumberFormat:  envOr("CASE_NUMBER_FORMAT", "prefix_year_seq"),
		CaseLabel:         envOr("CASE_LABEL", "Case"),
		CasesLabel:        envOr("CASES_LABEL", "Cases"),
		TaskLabel:         envOr("TASK_LABEL", "Task"),
		TasksLabel:        envOr("TASKS_LABEL", "Tasks"),
		DocumentLabel:     envOr("DOCUMENT_LABEL", "Document"),
		DocumentsLabel:    envOr("DOCUMENTS_LABEL", "Documents"),
		RiskScoreLabel:    envOr("RISK_SCORE_LABEL", "Risk Score"),
		RiskLevelLabel:    envOr("RISK_LEVEL_LABEL", "Risk Level"),
		SLALabel:          envOr("SLA_LABEL", "SLA"),
		CommsLabel:        envOr("COMMS_LABEL", "Communications"),
		AnalystLabel:      envOr("ANALYST_LABEL", "Analyst"),
		TimelineLabel:     envOr("TIMELINE_LABEL", "Timeline"),
		NoteLabel:         envOr("NOTE_LABEL", "Note"),
		NotesLabel:        envOr("NOTES_LABEL", "Notes"),
		StrLabel:          envOr("STR_LABEL", "STR"),
		StrReferenceLabel: envOr("STR_REFERENCE_LABEL", "Reference No."),
		StrRegulatorLabel: envOr("STR_REGULATOR_LABEL", "FIU"),
		StrLawReference:   envOr("STR_LAW_REFERENCE", ""),
		StrIDLabel:        envOr("STR_ID_LABEL", "ID Number"),
		StrDeadlineDays:   getEnvInt("STR_DEADLINE_DAYS", 30),
		LocalCurrency:     envOr("LOCAL_CURRENCY", "USD"),
		CurrencySymbol:    envOr("CURRENCY_SYMBOL", "$"),
		CTRThreshold:      getEnvFloat("CTR_THRESHOLD", 0),
		WireThreshold:     getEnvFloat("WIRE_THRESHOLD", 0),
		IDTypes:           splitEnv("ID_TYPES", []string{"PASSPORT", "NATIONAL_ID", "DRIVING_LICENSE", "OTHER"}),
		Jurisdiction:      jurisdiction,
		Country:           envOr("DEPLOYMENT_COUNTRY", country),
		GoAMLEnabled:      getEnvBool("GOAML_ENABLED", false),
		UBOEnabled:        getEnvBool("UBO_ENABLED", false),
		GDPREnabled:       getEnvBool("GDPR_ENABLED", false),
		RetentionYears:    getEnvInt("RETENTION_YEARS", 5),
		UBOThresholdPct:   getEnvInt("UBO_THRESHOLD_PCT", 0),
	}
}

func indiaDefaults() AppConfig {
	return AppConfig{
		AppName:          "OpsOwl AI",
		Industry:         envOr("INDUSTRY", "financial_services"),
		CasePrefix:       envOr("CASE_PREFIX", "OWL"),
		CaseNumberFormat: envOr("CASE_NUMBER_FORMAT", "prefix_year_seq"),
		CaseLabel:        envOr("CASE_LABEL", "Case"),
		CasesLabel:       envOr("CASES_LABEL", "Cases"),
		TaskLabel:        envOr("TASK_LABEL", "Task"),
		TasksLabel:       envOr("TASKS_LABEL", "Tasks"),
		DocumentLabel:    envOr("DOCUMENT_LABEL", "Document"),
		DocumentsLabel:   envOr("DOCUMENTS_LABEL", "Documents"),
		RiskScoreLabel:   envOr("RISK_SCORE_LABEL", "Risk Score"),
		RiskLevelLabel:   envOr("RISK_LEVEL_LABEL", "Risk Level"),
		SLALabel:         envOr("SLA_LABEL", "SLA"),
		CommsLabel:       envOr("COMMS_LABEL", "Communications"),
		AnalystLabel:     envOr("ANALYST_LABEL", "Analyst"),
		TimelineLabel:    envOr("TIMELINE_LABEL", "Timeline"),
		NoteLabel:        envOr("NOTE_LABEL", "Note"),
		NotesLabel:       envOr("NOTES_LABEL", "Notes"),
		StrLabel:          envOr("STR_LABEL", "STR"),
		StrReferenceLabel: envOr("STR_REFERENCE_LABEL", "FIU Reference"),
		StrRegulatorLabel: envOr("STR_REGULATOR_LABEL", "FIU-IND"),
		StrLawReference:   envOr("STR_LAW_REFERENCE", "PMLA 2002"),
		StrIDLabel:        envOr("STR_ID_LABEL", "PAN"),
		StrDeadlineDays:   getEnvInt("STR_DEADLINE_DAYS", 30),
		LocalCurrency:  envOr("LOCAL_CURRENCY", "INR"),
		CurrencySymbol: envOr("CURRENCY_SYMBOL", "₹"),
		CTRThreshold:   getEnvFloat("CTR_THRESHOLD", 1000000),
		WireThreshold:  getEnvFloat("WIRE_THRESHOLD", 0),
		IDTypes: splitEnv("ID_TYPES", []string{
			"PASSPORT", "NATIONAL_ID", "TAX_ID", "DRIVING_LICENSE",
			"PAN", "AADHAAR_REF", "VOTER_ID", "OTHER",
		}),
		Jurisdiction:    "india",
		Country:         envOr("DEPLOYMENT_COUNTRY", "in"),
		GoAMLEnabled:    getEnvBool("GOAML_ENABLED", false),
		UBOEnabled:      getEnvBool("UBO_ENABLED", false),
		GDPREnabled:     getEnvBool("GDPR_ENABLED", false),
		RetentionYears:  getEnvInt("RETENTION_YEARS", 5),
		UBOThresholdPct: getEnvInt("UBO_THRESHOLD_PCT", 0),
	}
}

func uaeDefaults() AppConfig {
	return AppConfig{
		AppName:          "OpsOwl AI",
		Industry:         envOr("INDUSTRY", "financial_services"),
		CasePrefix:       envOr("CASE_PREFIX", "OWL"),
		CaseNumberFormat: envOr("CASE_NUMBER_FORMAT", "prefix_year_seq"),
		CaseLabel:        envOr("CASE_LABEL", "Case"),
		CasesLabel:       envOr("CASES_LABEL", "Cases"),
		TaskLabel:        envOr("TASK_LABEL", "Task"),
		TasksLabel:       envOr("TASKS_LABEL", "Tasks"),
		DocumentLabel:    envOr("DOCUMENT_LABEL", "Document"),
		DocumentsLabel:   envOr("DOCUMENTS_LABEL", "Documents"),
		RiskScoreLabel:   envOr("RISK_SCORE_LABEL", "Risk Score"),
		RiskLevelLabel:   envOr("RISK_LEVEL_LABEL", "Risk Level"),
		SLALabel:         envOr("SLA_LABEL", "SLA"),
		CommsLabel:       envOr("COMMS_LABEL", "Communications"),
		AnalystLabel:     envOr("ANALYST_LABEL", "Analyst"),
		TimelineLabel:    envOr("TIMELINE_LABEL", "Timeline"),
		NoteLabel:        envOr("NOTE_LABEL", "Note"),
		NotesLabel:       envOr("NOTES_LABEL", "Notes"),
		StrLabel:          envOr("STR_LABEL", "STR"),
		StrReferenceLabel: envOr("STR_REFERENCE_LABEL", "UAE FIU Reference"),
		StrRegulatorLabel: envOr("STR_REGULATOR_LABEL", "UAE FIU"),
		StrLawReference:   envOr("STR_LAW_REFERENCE", "Federal Decree-Law No. 20 of 2018"),
		StrIDLabel:        envOr("STR_ID_LABEL", "Emirates ID"),
		StrDeadlineDays:   getEnvInt("STR_DEADLINE_DAYS", 30),
		LocalCurrency:  envOr("LOCAL_CURRENCY", "AED"),
		CurrencySymbol: envOr("CURRENCY_SYMBOL", "AED"),
		CTRThreshold:   getEnvFloat("CTR_THRESHOLD", 55000),
		WireThreshold:  getEnvFloat("WIRE_THRESHOLD", 3675),
		IDTypes: splitEnv("ID_TYPES", []string{
			"PASSPORT", "EMIRATES_ID", "UAE_TRADE_LICENSE",
			"TRN", "GCC_NATIONAL_ID", "RESIDENCY_VISA",
			"DRIVING_LICENSE", "OTHER",
		}),
		Jurisdiction:    "uae",
		Country:         envOr("DEPLOYMENT_COUNTRY", "ae"),
		GoAMLEnabled:    getEnvBool("GOAML_ENABLED", true),
		UBOEnabled:      getEnvBool("UBO_ENABLED", false),
		GDPREnabled:     getEnvBool("GDPR_ENABLED", false),
		RetentionYears:  getEnvInt("RETENTION_YEARS", 5),
		UBOThresholdPct: getEnvInt("UBO_THRESHOLD_PCT", 0),
	}
}

// euDefaults returns EU AMLR 2024-compliant defaults, with thin country overlays.
// Most EU settings are identical across all 27 member states; only regulator name
// and some ID label details differ per country.
func euDefaults(country string) AppConfig {
	regulatorLabel := "FIU"
	idLabel := "National ID / Passport"
	switch country {
	case "gr":
		regulatorLabel = "Hellenic FIU"
		idLabel = "AFM / ID Card"
	case "de":
		regulatorLabel = "Financial Intelligence Unit (FIU)"
		idLabel = "Personalausweis / Reisepass"
	case "fr":
		regulatorLabel = "TRACFIN"
		idLabel = "Carte Nationale d'Identité"
	case "nl":
		regulatorLabel = "FIU-Netherlands"
		idLabel = "BSN / Paspoort"
	case "it":
		regulatorLabel = "Unità di Informazione Finanziaria (UIF)"
		idLabel = "Codice Fiscale / Carta d'Identità"
	case "es":
		regulatorLabel = "SEPBLAC"
		idLabel = "DNI / NIE / Pasaporte"
	case "pl":
		regulatorLabel = "GIIF"
		idLabel = "PESEL / Dowód Osobisty"
	case "lt":
		regulatorLabel = "FNTT"
		idLabel = "Asmens tapatybės kortelė / Pasas"
	case "cy":
		regulatorLabel = "MOKAS"
		idLabel = "ID Card / Passport"
	}

	return AppConfig{
		AppName:          "OpsOwl AI",
		Industry:         envOr("INDUSTRY", "financial_services"),
		CasePrefix:       envOr("CASE_PREFIX", "OWL"),
		CaseNumberFormat: envOr("CASE_NUMBER_FORMAT", "prefix_year_seq"),
		CaseLabel:        envOr("CASE_LABEL", "Case"),
		CasesLabel:       envOr("CASES_LABEL", "Cases"),
		TaskLabel:        envOr("TASK_LABEL", "Task"),
		TasksLabel:       envOr("TASKS_LABEL", "Tasks"),
		DocumentLabel:    envOr("DOCUMENT_LABEL", "Document"),
		DocumentsLabel:   envOr("DOCUMENTS_LABEL", "Documents"),
		RiskScoreLabel:   envOr("RISK_SCORE_LABEL", "Risk Score"),
		RiskLevelLabel:   envOr("RISK_LEVEL_LABEL", "Risk Level"),
		SLALabel:         envOr("SLA_LABEL", "SLA"),
		CommsLabel:       envOr("COMMS_LABEL", "Communications"),
		AnalystLabel:     envOr("ANALYST_LABEL", "Analyst"),
		TimelineLabel:    envOr("TIMELINE_LABEL", "Timeline"),
		NoteLabel:        envOr("NOTE_LABEL", "Note"),
		NotesLabel:       envOr("NOTES_LABEL", "Notes"),
		StrLabel:          envOr("STR_LABEL", "STR"),
		StrReferenceLabel: envOr("STR_REFERENCE_LABEL", "AMLA Reference"),
		StrRegulatorLabel: envOr("STR_REGULATOR_LABEL", regulatorLabel),
		StrLawReference:   envOr("STR_LAW_REFERENCE", "EU AMLR 2024 / AMLD6"),
		StrIDLabel:        envOr("STR_ID_LABEL", idLabel),
		StrDeadlineDays:   getEnvInt("STR_DEADLINE_DAYS", 5), // EU: immediate / ≤5 working days
		LocalCurrency:  envOr("LOCAL_CURRENCY", "EUR"),
		CurrencySymbol: envOr("CURRENCY_SYMBOL", "€"),
		CTRThreshold:   getEnvFloat("CTR_THRESHOLD", 10000),  // EU cash threshold
		WireThreshold:  getEnvFloat("WIRE_THRESHOLD", 1000),  // EU wire transfer regulation
		IDTypes: splitEnv("ID_TYPES", []string{
			"PASSPORT", "NATIONAL_ID", "TAX_ID", "DRIVING_LICENSE",
			"EU_RESIDENCE_PERMIT", "OTHER",
		}),
		Jurisdiction:    "eu",
		Country:         envOr("DEPLOYMENT_COUNTRY", country),
		GoAMLEnabled:    getEnvBool("GOAML_ENABLED", true),
		UBOEnabled:      getEnvBool("UBO_ENABLED", true),
		GDPREnabled:     getEnvBool("GDPR_ENABLED", true),
		RetentionYears:  getEnvInt("RETENTION_YEARS", 5),
		UBOThresholdPct: getEnvInt("UBO_THRESHOLD_PCT", 25),
	}
}

type Handler struct {
	db           *sql.DB
	jurisdiction string // from DEPLOYMENT_JURISDICTION env
	country      string // from DEPLOYMENT_COUNTRY env
}

func NewHandler(db *sql.DB) *Handler {
	jurisdiction := strings.ToLower(strings.TrimSpace(os.Getenv("DEPLOYMENT_JURISDICTION")))
	country := strings.ToLower(strings.TrimSpace(os.Getenv("DEPLOYMENT_COUNTRY")))
	if jurisdiction == "" {
		jurisdiction = "india"
	}
	return &Handler{db: db, jurisdiction: jurisdiction, country: country}
}

// injectReadOnlyFields overlays the deployment env values that must never come
// from the database — jurisdiction, country, and the feature flags derived from them.
func (h *Handler) injectReadOnlyFields(cfg *AppConfig) {
	defaults := deploymentDefaults()
	cfg.Jurisdiction = defaults.Jurisdiction
	cfg.Country = defaults.Country
	cfg.GoAMLEnabled = defaults.GoAMLEnabled
	cfg.UBOEnabled = defaults.UBOEnabled
	cfg.GDPREnabled = defaults.GDPREnabled
	cfg.RetentionYears = defaults.RetentionYears
	cfg.UBOThresholdPct = defaults.UBOThresholdPct
}

// GET /api/v1/config — public, no auth required so FE can load before login.
// Priority: DB row (admin-edited, same jurisdiction) → deployment defaults.
// If the stored config was saved under a different jurisdiction (detected via
// the hidden "_saved_for_jurisdiction" marker), the DB row is treated as stale
// and fresh deployment defaults are returned so labels/currency/thresholds
// always match the active jurisdiction without requiring a manual reset.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	var raw []byte
	err := h.db.QueryRowContext(r.Context(),
		`SELECT config FROM system_config LIMIT 1`,
	).Scan(&raw)

	if err != nil {
		cfg := deploymentDefaults()
		response.JSON(w, http.StatusOK, cfg)
		return
	}

	// Check jurisdiction marker before deserializing.
	var rawMap map[string]interface{}
	if jsonErr := json.Unmarshal(raw, &rawMap); jsonErr == nil {
		savedFor, _ := rawMap["_saved_for_jurisdiction"].(string)
		if savedFor == "" {
			savedFor = "india" // backward compat: rows without marker assumed India
		}
		if savedFor != h.jurisdiction {
			cfg := deploymentDefaults()
			response.JSON(w, http.StatusOK, cfg)
			return
		}
	}

	var cfg AppConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		cfg := deploymentDefaults()
		response.JSON(w, http.StatusOK, cfg)
		return
	}
	// Backfill fields missing from older stored blobs.
	defaults := deploymentDefaults()
	if cfg.CasePrefix == "" {
		cfg.CasePrefix = defaults.CasePrefix
	}
	if cfg.CaseNumberFormat == "" {
		cfg.CaseNumberFormat = defaults.CaseNumberFormat
	}
	if cfg.LocalCurrency == "" {
		cfg.LocalCurrency = defaults.LocalCurrency
	}
	if cfg.CurrencySymbol == "" {
		cfg.CurrencySymbol = defaults.CurrencySymbol
	}
	if cfg.CTRThreshold == 0 {
		cfg.CTRThreshold = defaults.CTRThreshold
	}
	if cfg.StrDeadlineDays == 0 {
		cfg.StrDeadlineDays = defaults.StrDeadlineDays
	}
	if len(cfg.IDTypes) == 0 {
		cfg.IDTypes = defaults.IDTypes
	}
	// Jurisdiction fields are always from env — never from DB.
	h.injectReadOnlyFields(&cfg)
	response.JSON(w, http.StatusOK, cfg)
}

// GET /api/v1/config/defaults — public, returns the deployment-level defaults
// from env vars. Frontend uses this for the "Reset to Deployment Defaults" button.
func (h *Handler) Defaults(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, deploymentDefaults())
}

// PUT /api/v1/config — admin+ only, saves admin customisations to DB.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())

	var incoming AppConfig
	if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if incoming.CaseLabel == "" {
		response.Err(w, http.StatusBadRequest, "MISSING_FIELDS", "case_label is required")
		return
	}

	// Brand is non-negotiable — always store as the product name.
	incoming.AppName = "OpsOwl AI"
	// Strip read-only fields so they are never stored in the DB.
	incoming.Jurisdiction = ""
	incoming.Country = ""
	incoming.GoAMLEnabled = false
	incoming.UBOEnabled = false
	incoming.GDPREnabled = false
	incoming.RetentionYears = 0
	incoming.UBOThresholdPct = 0

	// Embed a hidden jurisdiction marker so Get() can detect stale configs
	// when the deployment jurisdiction changes between saves.
	configMap := make(map[string]interface{})
	if b, err := json.Marshal(incoming); err == nil {
		_ = json.Unmarshal(b, &configMap)
	}
	configMap["_saved_for_jurisdiction"] = h.jurisdiction
	raw, _ := json.Marshal(configMap)

	result, err := h.db.ExecContext(r.Context(),
		`UPDATE system_config SET config = $1, updated_at = NOW(), updated_by = $2`,
		raw, claims.UserID,
	)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to save config")
		return
	}

	if n, _ := result.RowsAffected(); n == 0 {
		_, err = h.db.ExecContext(r.Context(),
			`INSERT INTO system_config (config, updated_at, updated_by) VALUES ($1, NOW(), $2)`,
			raw, claims.UserID,
		)
		if err != nil {
			response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to save config")
			return
		}
	}

	h.injectReadOnlyFields(&incoming)
	response.JSON(w, http.StatusOK, incoming)
}
