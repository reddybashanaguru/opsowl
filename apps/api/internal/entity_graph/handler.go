package entity_graph

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// POST /subjects/{id}/relationships
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	subjectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid subject id")
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())

	var req CreateRelationshipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.ToSubjectID == uuid.Nil {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "to_subject_id is required")
		return
	}
	if req.RelationshipType == "" {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "relationship_type is required")
		return
	}

	rel, err := h.svc.Create(r.Context(), claims, subjectID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrSelfLink):
			response.Err(w, http.StatusBadRequest, "SELF_LINK", err.Error())
		case errors.Is(err, ErrInvalidType):
			response.Err(w, http.StatusBadRequest, "INVALID_TYPE", err.Error())
		case errors.Is(err, ErrDuplicate):
			response.Err(w, http.StatusConflict, "DUPLICATE_RELATIONSHIP", "relationship already exists between these subjects")
		default:
			response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create relationship")
		}
		return
	}
	response.JSON(w, http.StatusCreated, rel)
}

// GET /subjects/{id}/relationships
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	subjectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid subject id")
		return
	}
	rels, err := h.svc.List(r.Context(), subjectID)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list relationships")
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"relationships": rels,
		"total":         len(rels),
	})
}

// DELETE /subjects/{id}/relationships/{rel_id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	subjectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid subject id")
		return
	}
	relID, err := uuid.Parse(chi.URLParam(r, "rel_id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid relationship id")
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	if err := h.svc.Delete(r.Context(), claims, subjectID, relID); err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "relationship not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete relationship")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// GET /subjects/{id}/network?depth=2
func (h *Handler) GetNetwork(w http.ResponseWriter, r *http.Request) {
	subjectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid subject id")
		return
	}
	depth := 2
	if d := r.URL.Query().Get("depth"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n >= 1 && n <= 3 {
			depth = n
		}
	}
	net, err := h.svc.GetNetwork(r.Context(), subjectID, depth)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to traverse network")
		return
	}
	response.JSON(w, http.StatusOK, net)
}
