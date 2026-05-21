package users

import (
	"encoding/json"
	"errors"
	"net/http"

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
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.Email == "" || req.FullName == "" || req.Password == "" {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "email, full_name, and password are required")
		return
	}
	if req.Role == "" {
		req.Role = db.UserRoleAnalyst
	}

	user, err := h.svc.Create(r.Context(), claims, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailTaken):
			response.Err(w, http.StatusConflict, "EMAIL_TAKEN", "a user with this email already exists")
		case errors.Is(err, ErrWeakPassword):
			response.Err(w, http.StatusBadRequest, "WEAK_PASSWORD", "password must be at least 8 characters with one uppercase letter and one digit")
		default:
			response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not create user")
		}
		return
	}
	response.JSON(w, http.StatusCreated, user)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var roleFilter *db.UserRole
	if rs := q.Get("role"); rs != "" {
		role := db.UserRole(rs)
		roleFilter = &role
	}

	var cursor *uuid.UUID
	if cs := q.Get("cursor"); cs != "" {
		id, err := uuid.Parse(cs)
		if err == nil {
			cursor = &id
		}
	}

	limit := int32(20)
	result, err := h.svc.List(r.Context(), roleFilter, cursor, limit)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list users")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": result.Users,
		"meta": map[string]any{
			"next_cursor":  result.NextCursor,
			"has_more":     result.HasMore,
			"total_count":  result.TotalCount,
		},
		"error": nil,
	})
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid user id")
		return
	}
	user, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		response.Err(w, http.StatusNotFound, "USER_NOT_FOUND", "user not found")
		return
	}
	response.JSON(w, http.StatusOK, user)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid user id")
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	user, err := h.svc.Update(r.Context(), claims, id, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrForbidden):
			response.Err(w, http.StatusForbidden, "FORBIDDEN", "you cannot perform this update")
		case errors.Is(err, ErrUserNotFound):
			response.Err(w, http.StatusNotFound, "USER_NOT_FOUND", "user not found")
		default:
			response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not update user")
		}
		return
	}
	response.JSON(w, http.StatusOK, user)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid user id")
		return
	}

	if err := h.svc.Delete(r.Context(), claims, id); err != nil {
		switch err.Error() {
		case "cannot_delete_self":
			response.Err(w, http.StatusBadRequest, "CANNOT_DELETE_SELF", "you cannot delete your own account")
		case "user_not_found":
			response.Err(w, http.StatusNotFound, "USER_NOT_FOUND", "user not found")
		default:
			response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not delete user")
		}
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "user deactivated"})
}

// Unlock clears account lockout for a user — admin/super_admin only.
func (h *Handler) Unlock(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid user id")
		return
	}
	if err := h.svc.Unlock(r.Context(), id); err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not unlock account")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "account unlocked"})
}
