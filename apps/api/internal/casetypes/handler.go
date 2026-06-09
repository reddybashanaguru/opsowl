package casetypes

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

type CaseType struct {
	ID          string    `json:"id"`
	Slug        string    `json:"slug"`
	Label       string    `json:"label"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Route("/case-types", func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))
		r.Get("/", h.List)
		r.With(auth.RequireRole("admin", "super_admin")).Post("/", h.Create)
		r.With(auth.RequireRole("admin", "super_admin")).Delete("/{slug}", h.Delete)
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(),
		`SELECT id, slug, label, description, created_at FROM case_types ORDER BY label ASC`)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	defer rows.Close()

	types := []CaseType{}
	for rows.Next() {
		var ct CaseType
		if err := rows.Scan(&ct.ID, &ct.Slug, &ct.Label, &ct.Description, &ct.CreatedAt); err != nil {
			response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}
		types = append(types, ct)
	}
	response.JSON(w, http.StatusOK, types)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Slug        string `json:"slug"`
		Label       string `json:"label"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}

	body.Slug = strings.ToLower(strings.TrimSpace(body.Slug))
	body.Label = strings.TrimSpace(body.Label)
	if body.Slug == "" || body.Label == "" {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "slug and label are required")
		return
	}

	var ct CaseType
	err := h.db.QueryRowContext(r.Context(),
		`INSERT INTO case_types (slug, label, description)
		 VALUES ($1, $2, $3)
		 RETURNING id, slug, label, description, created_at`,
		body.Slug, body.Label, body.Description,
	).Scan(&ct.ID, &ct.Slug, &ct.Label, &ct.Description, &ct.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			response.Err(w, http.StatusConflict, "DUPLICATE_SLUG", "a case type with this slug already exists")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, ct)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	// Prevent deletion if any cases use this type
	var count int
	h.db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM cases WHERE case_type = $1 AND deleted_at IS NULL`, slug,
	).Scan(&count)
	if count > 0 {
		response.Err(w, http.StatusConflict, "IN_USE",
			"cannot delete — this case type is used by existing cases")
		return
	}

	res, err := h.db.ExecContext(r.Context(),
		`DELETE FROM case_types WHERE slug = $1`, slug)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "case type not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
