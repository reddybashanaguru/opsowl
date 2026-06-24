package typologies

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

// ── Types ──────────────────────────────────────────────────────────────────

type Typology struct {
	ID            string          `json:"id"`
	Code          string          `json:"code"`
	Name          string          `json:"name"`
	Category      string          `json:"category"`
	Description   string          `json:"description"`
	RedFlags      json.RawMessage `json:"red_flags"`
	RiskKeywords  json.RawMessage `json:"risk_keywords"`
	FATFReference string          `json:"fatf_reference"`
	CreatedAt     time.Time       `json:"created_at"`
}

type CaseTypology struct {
	Typology
	TaggedBy      *string   `json:"tagged_by,omitempty"`
	TaggedAt      time.Time `json:"tagged_at"`
	Notes         string    `json:"notes"`
	IsAISuggested bool      `json:"is_ai_suggested"`
}

type TagRequest struct {
	TypologyID    string `json:"typology_id"`
	Notes         string `json:"notes"`
	IsAISuggested bool   `json:"is_ai_suggested"`
}

// ── Handler ────────────────────────────────────────────────────────────────

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))

		// Library — all authenticated users can browse
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/typologies", h.List)
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/typologies/{id}", h.Get)

		// Case tagging — analysts and above
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Get("/cases/{id}/typologies", h.ListForCase)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/cases/{id}/typologies", h.Tag)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Delete("/cases/{id}/typologies/{typologyId}", h.Untag)
	})
}

// GET /api/v1/typologies?category=LAYERING&q=search
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	q        := strings.TrimSpace(r.URL.Query().Get("q"))

	query := `SELECT id, code, name, category, description, red_flags, risk_keywords, fatf_reference, created_at
	          FROM typologies`
	args  := []any{}
	where := []string{}

	if category != "" {
		args = append(args, category)
		where = append(where, "category = $"+itoa(len(args)))
	}
	if q != "" {
		args = append(args, "%"+strings.ToLower(q)+"%")
		n := itoa(len(args))
		where = append(where, "(LOWER(name) LIKE $"+n+" OR LOWER(description) LIKE $"+n+")")
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY category ASC, name ASC"

	rows, err := h.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	defer rows.Close()

	result := []Typology{}
	for rows.Next() {
		var t Typology
		if err := rows.Scan(&t.ID, &t.Code, &t.Name, &t.Category, &t.Description,
			&t.RedFlags, &t.RiskKeywords, &t.FATFReference, &t.CreatedAt); err != nil {
			response.Err(w, http.StatusInternalServerError, "SCAN_ERROR", err.Error())
			return
		}
		result = append(result, t)
	}
	response.JSON(w, http.StatusOK, result)
}

// GET /api/v1/typologies/:id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var t Typology
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, code, name, category, description, red_flags, risk_keywords, fatf_reference, created_at
		 FROM typologies WHERE id = $1`, id,
	).Scan(&t.ID, &t.Code, &t.Name, &t.Category, &t.Description,
		&t.RedFlags, &t.RiskKeywords, &t.FATFReference, &t.CreatedAt)
	if err == sql.ErrNoRows {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "typology not found")
		return
	}
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, t)
}

// GET /api/v1/cases/:id/typologies
func (h *Handler) ListForCase(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "id")

	rows, err := h.db.QueryContext(r.Context(), `
		SELECT t.id, t.code, t.name, t.category, t.description,
		       t.red_flags, t.risk_keywords, t.fatf_reference, t.created_at,
		       ct.tagged_by::text, ct.tagged_at, ct.notes, ct.is_ai_suggested
		FROM case_typologies ct
		JOIN typologies t ON t.id = ct.typology_id
		WHERE ct.case_id = $1
		ORDER BY ct.is_ai_suggested DESC, ct.tagged_at DESC`, caseID)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	defer rows.Close()

	result := []CaseTypology{}
	for rows.Next() {
		var ct CaseTypology
		var taggedBy sql.NullString
		if err := rows.Scan(
			&ct.ID, &ct.Code, &ct.Name, &ct.Category, &ct.Description,
			&ct.RedFlags, &ct.RiskKeywords, &ct.FATFReference, &ct.CreatedAt,
			&taggedBy, &ct.TaggedAt, &ct.Notes, &ct.IsAISuggested,
		); err != nil {
			response.Err(w, http.StatusInternalServerError, "SCAN_ERROR", err.Error())
			return
		}
		if taggedBy.Valid {
			ct.TaggedBy = &taggedBy.String
		}
		result = append(result, ct)
	}
	response.JSON(w, http.StatusOK, result)
}

// POST /api/v1/cases/:id/typologies
func (h *Handler) Tag(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "id")
	claims := auth.ClaimsFromCtx(r.Context())

	var req TagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}
	if req.TypologyID == "" {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "typology_id is required")
		return
	}

	_, err := h.db.ExecContext(r.Context(), `
		INSERT INTO case_typologies (case_id, typology_id, tagged_by, notes, is_ai_suggested)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (case_id, typology_id) DO UPDATE
		  SET notes = EXCLUDED.notes,
		      is_ai_suggested = EXCLUDED.is_ai_suggested`,
		caseID, req.TypologyID, claims.UserID, req.Notes, req.IsAISuggested,
	)
	if err != nil {
		if strings.Contains(err.Error(), "foreign key") {
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "typology or case not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// DELETE /api/v1/cases/:id/typologies/:typologyId
func (h *Handler) Untag(w http.ResponseWriter, r *http.Request) {
	caseID     := chi.URLParam(r, "id")
	typologyID := chi.URLParam(r, "typologyId")

	res, err := h.db.ExecContext(r.Context(),
		`DELETE FROM case_typologies WHERE case_id = $1 AND typology_id = $2`,
		caseID, typologyID)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "typology not tagged to this case")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func itoa(n int) string { return strconv.Itoa(n) }
