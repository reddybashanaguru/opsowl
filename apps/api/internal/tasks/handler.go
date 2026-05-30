package tasks

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// POST /api/v1/cases/{id}/tasks
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())

	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	t, err := h.svc.Create(r.Context(), claims, caseID, req)
	if err != nil {
		if errors.Is(err, ErrTitleRequired) {
			response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, t)
}

// GET /api/v1/cases/{id}/tasks
func (h *Handler) ListByCase(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	q := r.URL.Query()
	f := ListFilters{Limit: 50}
	if s := q.Get("status"); s != "" {
		st := db.TaskStatus(s)
		f.Status = &st
	}
	if p := q.Get("priority"); p != "" {
		pri := db.TaskPriority(p)
		f.Priority = &pri
	}
	if a := q.Get("assigned_to"); a != "" {
		if id, err := uuid.Parse(a); err == nil {
			f.AssignedTo = &id
		}
	}
	if q.Get("overdue") == "true" {
		f.OverdueOnly = true
	}
	if l := q.Get("limit"); l != "" {
		if n, _ := strconv.Atoi(l); n > 0 && n <= 100 {
			f.Limit = int32(n)
		}
	}
	if c := q.Get("cursor"); c != "" {
		if so, ca, id, err := DecodeCursor(c); err == nil {
			f.CursorSortOrder = &so
			f.CursorTime = &ca
			f.CursorID = &id
		}
	}

	tasks, meta, err := h.svc.ListByCase(r.Context(), caseID, f)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.JSONWithMeta(w, http.StatusOK, tasks, meta)
}

// GET /api/v1/tasks (my tasks, or all tasks when scope=all for manager+)
func (h *Handler) ListMine(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	q := r.URL.Query()
	f := ListFilters{Limit: 50}
	if s := q.Get("status"); s != "" {
		st := db.TaskStatus(s)
		f.Status = &st
	}
	if q.Get("overdue") == "true" {
		f.OverdueOnly = true
	}
	if q.Get("unassigned") == "true" {
		f.UnassignedOnly = true
	}
	if a := q.Get("assigned_to"); a != "" {
		if id, err := uuid.Parse(a); err == nil {
			f.AssignedTo = &id
		}
	}
	if l := q.Get("limit"); l != "" {
		if n, _ := strconv.Atoi(l); n > 0 && n <= 200 {
			f.Limit = int32(n)
		}
	}

	// scope=all lets managers and above see every task across all cases.
	isPrivileged := claims.Role == db.UserRoleManager ||
		claims.Role == db.UserRoleAdmin ||
		claims.Role == db.UserRoleSuperAdmin
	if q.Get("scope") == "all" && isPrivileged {
		tasks, meta, err := h.svc.ListAllTasks(r.Context(), f)
		if err != nil {
			response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}
		response.JSONWithMeta(w, http.StatusOK, tasks, meta)
		return
	}

	tasks, meta, err := h.svc.ListMyTasks(r.Context(), claims.UserID, f)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.JSONWithMeta(w, http.StatusOK, tasks, meta)
}

// GET /api/v1/tasks/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid task id")
		return
	}
	t, err := h.svc.Get(r.Context(), id)
	if err != nil {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, t)
}

// PUT /api/v1/tasks/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid task id")
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())

	var req UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	t, err := h.svc.Update(r.Context(), claims, id, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			response.Err(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		case errors.Is(err, ErrForbidden):
			response.Err(w, http.StatusForbidden, "FORBIDDEN", err.Error())
		case errors.Is(err, ErrInvalidTransition):
			response.Err(w, http.StatusUnprocessableEntity, "INVALID_TRANSITION", err.Error())
		case errors.Is(err, ErrBlocked):
			response.Err(w, http.StatusConflict, "TASK_BLOCKED", err.Error())
		default:
			response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return
	}
	response.JSON(w, http.StatusOK, t)
}

// POST /api/v1/tasks/{id}/complete
func (h *Handler) Complete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid task id")
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())

	t, err := h.svc.Complete(r.Context(), claims, id)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			response.Err(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		case errors.Is(err, ErrInvalidTransition):
			response.Err(w, http.StatusUnprocessableEntity, "INVALID_TRANSITION", err.Error())
		case errors.Is(err, ErrBlocked):
			response.Err(w, http.StatusConflict, "TASK_BLOCKED", err.Error())
		default:
			response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return
	}
	response.JSON(w, http.StatusOK, t)
}

// DELETE /api/v1/tasks/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid task id")
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())

	if err := h.svc.Delete(r.Context(), claims, id); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			response.Err(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		case errors.Is(err, ErrForbidden):
			response.Err(w, http.StatusForbidden, "FORBIDDEN", err.Error())
		default:
			response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/v1/tasks/{id}/comments
func (h *Handler) AddComment(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid task id")
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())

	var req CreateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	c, err := h.svc.AddComment(r.Context(), claims, id, req)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, c)
}

// GET /api/v1/tasks/{id}/comments
func (h *Handler) ListComments(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid task id")
		return
	}

	q := r.URL.Query()
	var limit int32 = 50
	if l := q.Get("limit"); l != "" {
		if n, _ := strconv.Atoi(l); n > 0 && n <= 100 {
			limit = int32(n)
		}
	}
	var cursorID *uuid.UUID
	if c := q.Get("cursor"); c != "" {
		if id, err := uuid.Parse(c); err == nil {
			cursorID = &id
		}
	}

	comments, hasMore, nextCursor, err := h.svc.ListComments(r.Context(), id, cursorID, limit)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	type meta struct {
		HasMore    bool       `json:"has_more"`
		NextCursor *uuid.UUID `json:"next_cursor"`
	}
	response.JSONWithMeta(w, http.StatusOK, comments, meta{HasMore: hasMore, NextCursor: nextCursor})
}
