package assignment

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

// Rule mirrors the assignment_rules DB row.
type Rule struct {
	ID            uuid.UUID  `json:"id"`
	Name          string     `json:"name"`
	Description   *string    `json:"description,omitempty"`
	PriorityOrder int        `json:"priority_order"`
	MatchPriority *string    `json:"match_priority,omitempty"`
	MatchCaseType *string    `json:"match_case_type,omitempty"`
	MatchSourceID *uuid.UUID `json:"match_source_id,omitempty"`
	AssignToUser  *uuid.UUID `json:"assign_to_user,omitempty"`
	AssignToRole  *string    `json:"assign_to_role,omitempty"`
	IsActive      bool       `json:"is_active"`
	CreatedAt     time.Time  `json:"created_at"`
}

// MatchInput describes an incoming case for rule evaluation.
type MatchInput struct {
	Priority  string
	CaseType  string
	SourceID  *uuid.UUID
}

// Engine evaluates active assignment rules against a new case and returns
// the first matching assignee user ID (or uuid.Nil if no rule matched).
type Engine struct {
	db *sql.DB
}

func NewEngine(db *sql.DB) *Engine {
	return &Engine{db: db}
}

// Evaluate returns the user that should be assigned, or uuid.Nil when no rule fires.
func (e *Engine) Evaluate(ctx context.Context, input MatchInput) (uuid.UUID, error) {
	rows, err := e.db.QueryContext(ctx, `
		SELECT assign_to_user, assign_to_role
		FROM assignment_rules
		WHERE is_active = TRUE
		  AND (match_priority IS NULL OR LOWER(match_priority) = LOWER($1))
		  AND (match_case_type IS NULL OR LOWER(match_case_type) = LOWER($2))
		  AND (match_source_id IS NULL OR match_source_id = $3)
		ORDER BY priority_order ASC
		LIMIT 20
	`, input.Priority, input.CaseType, input.SourceID)
	if err != nil {
		return uuid.Nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var assignUser uuid.NullUUID
		var assignRole sql.NullString
		if err := rows.Scan(&assignUser, &assignRole); err != nil {
			continue
		}
		// Direct user assignment takes precedence
		if assignUser.Valid {
			return assignUser.UUID, nil
		}
		// Role-based: pick the user with that role who has fewest open cases
		if assignRole.Valid {
			userID, err := e.leastLoadedUserWithRole(ctx, assignRole.String)
			if err == nil && userID != uuid.Nil {
				return userID, nil
			}
		}
	}
	return uuid.Nil, nil
}

func (e *Engine) leastLoadedUserWithRole(ctx context.Context, role string) (uuid.UUID, error) {
	var userID uuid.UUID
	err := e.db.QueryRowContext(ctx, `
		SELECT u.id
		FROM users u
		LEFT JOIN cases c ON c.assigned_to = u.id AND c.status NOT IN ('closed', 'escalated')
		WHERE u.role = $1 AND u.is_active = TRUE AND u.deleted_at IS NULL
		GROUP BY u.id
		ORDER BY COUNT(c.id) ASC
		LIMIT 1
	`, role).Scan(&userID)
	if err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}

// Handler exposes CRUD endpoints for assignment rules (admin only).
type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Route("/assignment-rules", func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))
		r.Use(auth.RequireRole("admin", "super_admin"))
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(), `
		SELECT id, name, description, priority_order, match_priority, match_case_type, match_source_id,
		       assign_to_user, assign_to_role, is_active, created_at
		FROM assignment_rules ORDER BY priority_order ASC
	`)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load rules")
		return
	}
	defer rows.Close()
	var rules []Rule
	for rows.Next() {
		var ru Rule
		var desc, matchPri, matchCaseType, assignRole sql.NullString
		var matchSrc, assignUser uuid.NullUUID
		if err := rows.Scan(&ru.ID, &ru.Name, &desc, &ru.PriorityOrder, &matchPri, &matchCaseType, &matchSrc,
			&assignUser, &assignRole, &ru.IsActive, &ru.CreatedAt); err != nil {
			continue
		}
		if desc.Valid {
			ru.Description = &desc.String
		}
		if matchPri.Valid {
			ru.MatchPriority = &matchPri.String
		}
		if matchCaseType.Valid {
			ru.MatchCaseType = &matchCaseType.String
		}
		if matchSrc.Valid {
			ru.MatchSourceID = &matchSrc.UUID
		}
		if assignUser.Valid {
			ru.AssignToUser = &assignUser.UUID
		}
		if assignRole.Valid {
			ru.AssignToRole = &assignRole.String
		}
		rules = append(rules, ru)
	}
	response.JSON(w, http.StatusOK, map[string]any{"rules": rules})
}

type createRuleRequest struct {
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	PriorityOrder int        `json:"priority_order"`
	MatchPriority string     `json:"match_priority"`
	MatchCaseType string     `json:"match_case_type"`
	MatchSourceID *uuid.UUID `json:"match_source_id"`
	AssignToUser  *uuid.UUID `json:"assign_to_user"`
	AssignToRole  string     `json:"assign_to_role"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var req createRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.Name == "" {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required")
		return
	}
	if req.PriorityOrder == 0 {
		req.PriorityOrder = 100
	}
	var id uuid.UUID
	err := h.db.QueryRowContext(r.Context(), `
		INSERT INTO assignment_rules
		  (name, description, priority_order, match_priority, match_case_type, match_source_id, assign_to_user, assign_to_role, created_by)
		VALUES ($1, NULLIF($2,''), $3, NULLIF(LOWER($4),''), NULLIF(LOWER($5),''), $6, $7, NULLIF(LOWER($8),''), $9)
		RETURNING id
	`, req.Name, req.Description, req.PriorityOrder, req.MatchPriority, req.MatchCaseType, req.MatchSourceID,
		req.AssignToUser, req.AssignToRole, claims.UserID).Scan(&id)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not create rule")
		return
	}
	response.JSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid rule id")
		return
	}
	var req createRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	result, err := h.db.ExecContext(r.Context(), `
		UPDATE assignment_rules
		SET name=$1, description=NULLIF($2,''), priority_order=$3,
		    match_priority=NULLIF(LOWER($4),''), match_case_type=NULLIF(LOWER($5),''),
		    match_source_id=$6, assign_to_user=$7, assign_to_role=NULLIF(LOWER($8),''), updated_at=NOW()
		WHERE id=$9
	`, req.Name, req.Description, req.PriorityOrder, req.MatchPriority, req.MatchCaseType, req.MatchSourceID,
		req.AssignToUser, req.AssignToRole, id)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not update rule")
		return
	}
	if n, _ := result.RowsAffected(); n == 0 {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "rule not found")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "updated"})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid rule id")
		return
	}
	result, err := h.db.ExecContext(r.Context(),
		`UPDATE assignment_rules SET is_active=FALSE, updated_at=NOW() WHERE id=$1`, id)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not delete rule")
		return
	}
	if n, _ := result.RowsAffected(); n == 0 {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "rule not found")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "deactivated"})
}
