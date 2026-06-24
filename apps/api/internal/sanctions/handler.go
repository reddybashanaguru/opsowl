package sanctions

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ScreenSubject POST /subjects/{id}/sanctions/screen
func (h *Handler) ScreenSubject(w http.ResponseWriter, r *http.Request) {
	subjectID := chi.URLParam(r, "id")
	var body struct {
		SubjectName string `json:"subject_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}
	if body.SubjectName == "" {
		response.Err(w, http.StatusBadRequest, "MISSING_NAME", "subject_name is required")
		return
	}

	claims := auth.ClaimsFromCtx(r.Context())
	userID := ""
	if claims != nil {
		userID = claims.UserID.String()
	}

	result, err := h.svc.Screen(r.Context(), subjectID, body.SubjectName, userID)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "SCREEN_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, result)
}

// GetHits GET /subjects/{id}/sanctions/hits
func (h *Handler) GetHits(w http.ResponseWriter, r *http.Request) {
	subjectID := chi.URLParam(r, "id")
	hits, err := h.svc.GetHits(r.Context(), subjectID)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "FETCH_FAILED", err.Error())
		return
	}
	if hits == nil {
		hits = []Hit{}
	}
	response.JSON(w, http.StatusOK, hits)
}

// AcknowledgeHit PATCH /sanctions/hits/{hitId}
func (h *Handler) AcknowledgeHit(w http.ResponseWriter, r *http.Request) {
	hitID := chi.URLParam(r, "hitId")
	var body struct {
		Status string `json:"status"`
		Notes  string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}

	claims := auth.ClaimsFromCtx(r.Context())
	userID := ""
	if claims != nil {
		userID = claims.UserID.String()
	}

	if err := h.svc.AcknowledgeHit(r.Context(), hitID, body.Status, body.Notes, userID); err != nil {
		response.Err(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": body.Status})
}

// GetLists GET /sanctions/lists
func (h *Handler) GetLists(w http.ResponseWriter, r *http.Request) {
	lists, err := h.svc.GetLists(r.Context())
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "FETCH_FAILED", err.Error())
		return
	}
	if lists == nil {
		lists = []SanctionsList{}
	}
	response.JSON(w, http.StatusOK, lists)
}

// GetAllHits GET /sanctions/hits — all open/confirmed hits across all subjects
func (h *Handler) GetAllHits(w http.ResponseWriter, r *http.Request) {
	hits, err := h.svc.GetAllOpenHits(r.Context())
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "FETCH_FAILED", err.Error())
		return
	}
	if hits == nil {
		hits = []Hit{}
	}
	response.JSON(w, http.StatusOK, hits)
}
