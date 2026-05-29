package cases

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

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

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var req CreateCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if len(req.Title) < 5 {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "title must be at least 5 characters")
		return
	}
	if req.Priority == "" {
		req.Priority = db.CasePriorityMedium
	}

	c, err := h.svc.Create(r.Context(), claims, req)
	if err != nil {
		if errors.Is(err, ErrTagLimit) {
			response.Err(w, http.StatusBadRequest, "TAG_LIMIT", err.Error())
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create case")
		return
	}
	response.JSON(w, http.StatusCreated, c)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}
	c, err := h.svc.Get(r.Context(), id, DefaultOrgID)
	if err != nil {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "case not found")
		return
	}
	response.JSON(w, http.StatusOK, c)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := ListFilters{Limit: 20}

	if s := q.Get("status"); s != "" {
		if s == "active" {
			f.ActiveOnly = true
		} else {
			st := db.CaseStatus(s)
			f.Status = &st
		}
	}
	if p := q.Get("priority"); p != "" {
		pr := db.CasePriority(p)
		f.Priority = &pr
	}
	if a := q.Get("assigned_to"); a != "" {
		if id, err := uuid.Parse(a); err == nil {
			f.AssignedTo = &id
		}
	}
	if sb := q.Get("sla_breached"); sb == "true" {
		t := true
		f.SLABreached = &t
	}
	if from := q.Get("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			f.From = &t
		}
	}
	if to := q.Get("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			f.To = &t
		}
	}
	if c := q.Get("cursor"); c != "" {
		if id, err := uuid.Parse(c); err == nil {
			f.Cursor = &id
		}
	}
	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			f.Limit = int32(n)
		}
	}
	if qStr := q.Get("q"); qStr != "" {
		f.Q = &qStr
	}

	// Restrict analysts and viewers to cases they own, are assigned to, or watch.
	claims := auth.ClaimsFromCtx(r.Context())
	if claims.Role == db.UserRoleAnalyst || claims.Role == db.UserRoleViewer {
		f.ViewerID = &claims.UserID
	}

	cases, meta, err := h.svc.List(r.Context(), f)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list cases")
		return
	}
	response.JSONWithMeta(w, http.StatusOK, cases, meta)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	var req UpdateCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	c, err := h.svc.Update(r.Context(), claims, id, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "case not found")
			return
		}
		if errors.Is(err, ErrForbidden) {
			response.Err(w, http.StatusForbidden, "FORBIDDEN", "you do not have permission to update this case")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update case")
		return
	}
	response.JSON(w, http.StatusOK, c)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	if err := h.svc.Delete(r.Context(), claims, id); err != nil {
		if errors.Is(err, ErrForbidden) {
			response.Err(w, http.StatusForbidden, "FORBIDDEN", "only admins can delete cases")
			return
		}
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "case not found")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"id": id.String(), "status": "deleted"})
}

func (h *Handler) ChangeStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	var req StatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	c, err := h.svc.ChangeStatus(r.Context(), claims, id, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "case not found")
			return
		}
		if errors.Is(err, ErrInvalidTransition) || errors.Is(err, ErrReasonRequired) {
			response.Err(w, http.StatusUnprocessableEntity, "INVALID_TRANSITION", err.Error())
			return
		}
		if errors.Is(err, ErrForbiddenTransition) {
			response.Err(w, http.StatusForbidden, "FORBIDDEN", err.Error())
			return
		}
		if errors.Is(err, ErrSamePersonApproval) {
			response.Err(w, http.StatusForbidden, "MAKER_CHECKER_VIOLATION", err.Error())
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update status")
		return
	}
	response.JSON(w, http.StatusOK, c)
}

func (h *Handler) Assign(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	var req AssignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	c, err := h.svc.Assign(r.Context(), claims, id, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "case not found")
			return
		}
		if errors.Is(err, ErrForbiddenTransition) {
			response.Err(w, http.StatusForbidden, "FORBIDDEN", "only managers and admins can assign cases")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to assign case")
		return
	}
	response.JSON(w, http.StatusOK, c)
}

func (h *Handler) UpdateTags(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	var req TagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	c, err := h.svc.UpdateTags(r.Context(), claims, id, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "case not found")
			return
		}
		if errors.Is(err, ErrTagLimit) {
			response.Err(w, http.StatusBadRequest, "TAG_LIMIT", err.Error())
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update tags")
		return
	}
	response.JSON(w, http.StatusOK, c)
}

func (h *Handler) ListActivity(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}
	q := r.URL.Query()
	var cursor *uuid.UUID
	if c := q.Get("cursor"); c != "" {
		if cid, err := uuid.Parse(c); err == nil {
			cursor = &cid
		}
	}
	limit := int32(50)
	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = int32(n)
		}
	}

	activities, hasMore, nextCursor, err := h.svc.ListActivities(r.Context(), id, cursor, limit)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list activities")
		return
	}
	response.JSONWithMeta(w, http.StatusOK, activities, map[string]interface{}{
		"next_cursor": nextCursor,
		"has_more":    hasMore,
	})
}

type bulkStatusRequest struct {
	IDs    []uuid.UUID `json:"ids"`
	Status string      `json:"status"`
}

func (h *Handler) BulkStatus(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var req bulkStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.IDs) == 0 {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "ids and status are required")
		return
	}
	if len(req.IDs) > 100 {
		response.Err(w, http.StatusBadRequest, "TOO_MANY", "maximum 100 cases per bulk operation")
		return
	}
	updated, failed := h.svc.BulkStatus(r.Context(), claims, req.IDs, db.CaseStatus(req.Status))
	response.JSON(w, http.StatusOK, map[string]any{"updated": updated, "failed": failed})
}

type bulkAssignRequest struct {
	IDs        []uuid.UUID `json:"ids"`
	AssigneeID uuid.UUID   `json:"assignee_id"`
}

func (h *Handler) BulkAssign(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var req bulkAssignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.IDs) == 0 {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "ids and assignee_id are required")
		return
	}
	if len(req.IDs) > 100 {
		response.Err(w, http.StatusBadRequest, "TOO_MANY", "maximum 100 cases per bulk operation")
		return
	}
	updated, failed := h.svc.BulkAssign(r.Context(), claims, req.IDs, req.AssigneeID)
	response.JSON(w, http.StatusOK, map[string]any{"updated": updated, "failed": failed})
}

type bulkDeleteRequest struct {
	IDs []uuid.UUID `json:"ids"`
}

func (h *Handler) BulkDelete(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var req bulkDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.IDs) == 0 {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "ids are required")
		return
	}
	if len(req.IDs) > 100 {
		response.Err(w, http.StatusBadRequest, "TOO_MANY", "maximum 100 cases per bulk operation")
		return
	}
	deleted, failed := h.svc.BulkDelete(r.Context(), claims, req.IDs)
	response.JSON(w, http.StatusOK, map[string]any{"deleted": deleted, "failed": failed})
}
