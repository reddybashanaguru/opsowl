package documents

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/reddybashanaguru/opsowl/internal/auth"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

const maxMemory = 50 << 20 // 50 MB in-memory parse budget

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// POST /api/v1/cases/{caseId}/documents
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "caseId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	claims := auth.ClaimsFromCtx(r.Context())

	if err := r.ParseMultipartForm(maxMemory); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "failed to parse multipart form")
		return
	}

	file, fh, err := r.FormFile("file")
	if err != nil {
		response.Err(w, http.StatusBadRequest, "MISSING_FILE", "field 'file' is required")
		return
	}
	defer file.Close()

	doc, err := h.svc.Upload(r.Context(), claims, caseID, fh.Filename, file, fh.Size)
	if err != nil {
		switch {
		case errors.Is(err, ErrFileTooLarge):
			response.Err(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "file exceeds 50 MB limit")
		case errors.Is(err, ErrFileTypeBlocked):
			response.Err(w, http.StatusUnprocessableEntity, "FILE_TYPE_BLOCKED", err.Error())
		default:
			response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return
	}

	response.JSON(w, http.StatusCreated, doc)
}

// GET /api/v1/cases/{caseId}/documents
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "caseId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	f := ListFilters{Limit: 20}
	if v := r.URL.Query().Get("limit"); v != "" {
		var l int32
		if _, err := parseIntParam(v, &l); err == nil && l > 0 {
			f.Limit = l
		}
	}
	if v := r.URL.Query().Get("cursor"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			f.CursorID = &id
		}
	}
	if v := r.URL.Query().Get("status"); v != "" {
		s := db.DocumentProcessingStatus(v)
		f.Status = &s
	}

	docs, meta, err := h.svc.List(r.Context(), caseID, f)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.JSONWithMeta(w, http.StatusOK, docs, meta)
}

// GET /api/v1/cases/{caseId}/documents/{docId}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	docID, err := uuid.Parse(chi.URLParam(r, "docId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid document id")
		return
	}

	doc, err := h.svc.Get(r.Context(), docID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "document not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.JSON(w, http.StatusOK, doc)
}

// DELETE /api/v1/cases/{caseId}/documents/{docId}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	docID, err := uuid.Parse(chi.URLParam(r, "docId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid document id")
		return
	}

	claims := auth.ClaimsFromCtx(r.Context())

	if err := h.svc.Delete(r.Context(), claims, docID); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "document not found")
		case errors.Is(err, ErrForbidden):
			response.Err(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		default:
			response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return
	}

	response.JSON(w, http.StatusNoContent, nil)
}

// GET /api/v1/documents  — global document library across all cases
func (h *Handler) ListAll(w http.ResponseWriter, r *http.Request) {
	limit := int32(30)
	if v := r.URL.Query().Get("limit"); v != "" {
		var l int32
		if _, err := parseIntParam(v, &l); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	var cursorID *uuid.UUID
	if v := r.URL.Query().Get("cursor"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			cursorID = &id
		}
	}
	var status *db.DocumentProcessingStatus
	if v := r.URL.Query().Get("status"); v != "" {
		s := db.DocumentProcessingStatus(v)
		status = &s
	}

	docs, meta, err := h.svc.ListAll(r.Context(), status, cursorID, limit)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.JSONWithMeta(w, http.StatusOK, docs, meta)
}

func parseIntParam(s string, out *int32) (int32, error) {
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, err
	}
	*out = int32(v)
	return int32(v), nil
}
