package config

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"

	"github.com/reddybashanaguru/opsowl/internal/auth"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

// AppConfig mirrors the JSONB blob stored in system_config.
type AppConfig struct {
	AppName        string `json:"app_name"`
	Industry       string `json:"industry"`
	CaseLabel      string `json:"case_label"`
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
}

// envOr returns the env var value if set and non-empty, otherwise fallback.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// deploymentDefaults reads terminology labels from env vars so each on-premise
// client can configure their vocabulary in .env without touching the database.
// AppName is intentionally NOT configurable — "OpsOwl AI" is the product brand.
func deploymentDefaults() AppConfig {
	return AppConfig{
		AppName:        "OpsOwl AI",
		Industry:       envOr("INDUSTRY", "financial_services"),
		CaseLabel:      envOr("CASE_LABEL", "Case"),
		CasesLabel:     envOr("CASES_LABEL", "Cases"),
		TaskLabel:      envOr("TASK_LABEL", "Task"),
		TasksLabel:     envOr("TASKS_LABEL", "Tasks"),
		DocumentLabel:  envOr("DOCUMENT_LABEL", "Document"),
		DocumentsLabel: envOr("DOCUMENTS_LABEL", "Documents"),
		RiskScoreLabel: envOr("RISK_SCORE_LABEL", "Risk Score"),
		RiskLevelLabel: envOr("RISK_LEVEL_LABEL", "Risk Level"),
		SLALabel:       envOr("SLA_LABEL", "SLA"),
		CommsLabel:     envOr("COMMS_LABEL", "Communications"),
		AnalystLabel:   envOr("ANALYST_LABEL", "Analyst"),
		TimelineLabel:  envOr("TIMELINE_LABEL", "Timeline"),
		NoteLabel:      envOr("NOTE_LABEL", "Note"),
		NotesLabel:     envOr("NOTES_LABEL", "Notes"),
	}
}

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

// GET /api/v1/config — public, no auth required so FE can load before login.
// Priority: DB row (admin-edited) → deployment defaults (from .env) → hardcoded.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	var raw []byte
	err := h.db.QueryRowContext(r.Context(),
		`SELECT config FROM system_config LIMIT 1`,
	).Scan(&raw)

	if err != nil {
		response.JSON(w, http.StatusOK, deploymentDefaults())
		return
	}

	var cfg AppConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		response.JSON(w, http.StatusOK, deploymentDefaults())
		return
	}
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

	raw, _ := json.Marshal(incoming)

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

	response.JSON(w, http.StatusOK, incoming)
}
