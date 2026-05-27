package csvimport

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/reddybashanaguru/opsowl/internal/auth"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

const maxUploadBytes = maxFileBytes + (1 << 10) // slightly above to detect oversized files

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// POST /api/v1/imports/preview
// Accepts multipart/form-data with field "file".
// Returns detected headers, sample rows, and auto-detected column mapping.
func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		response.Err(w, http.StatusBadRequest, "FILE_TOO_LARGE", ErrFileTooLarge.Error())
		return
	}
	f, _, err := r.FormFile("file")
	if err != nil {
		response.Err(w, http.StatusBadRequest, "MISSING_FILE", "form field 'file' is required")
		return
	}
	defer f.Close()

	preview, err := h.svc.Preview(io.LimitReader(f, maxFileBytes))
	switch {
	case errors.Is(err, ErrFileTooLarge):
		response.Err(w, http.StatusBadRequest, "FILE_TOO_LARGE", err.Error())
	case errors.Is(err, ErrTooManyRows):
		response.Err(w, http.StatusBadRequest, "TOO_MANY_ROWS", err.Error())
	case errors.Is(err, ErrEmptyFile):
		response.Err(w, http.StatusBadRequest, "EMPTY_FILE", err.Error())
	case err != nil:
		response.Err(w, http.StatusBadRequest, "PARSE_ERROR", err.Error())
	default:
		response.JSON(w, http.StatusOK, preview)
	}
}

// POST /api/v1/imports
// Accepts multipart/form-data with fields "file" (CSV) and "mapping" (JSON array of ColumnMapping).
func (h *Handler) StartImport(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		response.Err(w, http.StatusBadRequest, "FILE_TOO_LARGE", ErrFileTooLarge.Error())
		return
	}

	f, fh, err := r.FormFile("file")
	if err != nil {
		response.Err(w, http.StatusBadRequest, "MISSING_FILE", "form field 'file' is required")
		return
	}
	defer f.Close()

	mappingJSON := r.FormValue("mapping")
	if mappingJSON == "" {
		response.Err(w, http.StatusBadRequest, "MISSING_MAPPING", "form field 'mapping' is required")
		return
	}
	var mapping []ColumnMapping
	if err := json.Unmarshal([]byte(mappingJSON), &mapping); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_MAPPING", "mapping must be a JSON array of {csv_header, case_field}")
		return
	}

	claims := auth.ClaimsFromCtx(r.Context())
	job, err := h.svc.StartImport(r.Context(), fh.Filename, io.LimitReader(f, maxFileBytes), mapping, claims)
	switch {
	case errors.Is(err, ErrNoTitleColumn):
		response.Err(w, http.StatusBadRequest, "NO_TITLE_MAPPING", err.Error())
	case errors.Is(err, ErrFileTooLarge):
		response.Err(w, http.StatusBadRequest, "FILE_TOO_LARGE", err.Error())
	case errors.Is(err, ErrTooManyRows):
		response.Err(w, http.StatusBadRequest, "TOO_MANY_ROWS", err.Error())
	case errors.Is(err, ErrEmptyFile):
		response.Err(w, http.StatusBadRequest, "EMPTY_FILE", err.Error())
	case err != nil:
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "import failed to start")
	default:
		response.JSON(w, http.StatusAccepted, job)
	}
}

// GET /api/v1/imports
func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	limit := int32(20)
	offset := int32(0)
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = int32(n)
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = int32(n)
		}
	}
	jobs, err := h.svc.ListJobs(r.Context(), limit, offset)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list import jobs")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"imports": jobs})
}

// GET /api/v1/imports/{id}
func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid import ID")
		return
	}
	job, err := h.svc.GetJob(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "import job not found")
		return
	}
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not fetch import job")
		return
	}
	response.JSON(w, http.StatusOK, job)
}

// GET /api/v1/imports/{id}/rows?status=failed&limit=50&offset=0
func (h *Handler) ListRows(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid import ID")
		return
	}
	var statusFilter *string
	if v := r.URL.Query().Get("status"); v != "" {
		statusFilter = &v
	}
	limit := int32(50)
	offset := int32(0)
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = int32(n)
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = int32(n)
		}
	}
	page, err := h.svc.ListRows(r.Context(), id, statusFilter, limit, offset)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list rows")
		return
	}
	response.JSON(w, http.StatusOK, page)
}

// GET /api/v1/imports/columns
func (h *Handler) ListColumns(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]any{"columns": SupportedColumns})
}
