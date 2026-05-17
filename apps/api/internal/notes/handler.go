package notes

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

type Handler struct {
	svc    *Service
	broker *Broker
	jwtMgr *auth.JWTManager
}

func NewHandler(svc *Service, broker *Broker, jwtMgr *auth.JWTManager) *Handler {
	return &Handler{svc: svc, broker: broker, jwtMgr: jwtMgr}
}

// POST /api/v1/cases/:id/notes
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())

	var req CreateNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	note, err := h.svc.Create(r.Context(), claims, caseID, req)
	if err != nil {
		if errors.Is(err, ErrBadRequest) {
			response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "content is required")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, note)
}

// GET /api/v1/cases/:id/notes
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	q := r.URL.Query()
	noteType := q.Get("note_type")
	limit := int32(20)
	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = int32(n)
		}
	}
	cursor := uuid.Nil
	if c := q.Get("cursor"); c != "" {
		if id, err := uuid.Parse(c); err == nil {
			cursor = id
		}
	}

	notes, meta, err := h.svc.List(r.Context(), caseID, noteType, cursor, limit)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.JSONWithMeta(w, http.StatusOK, notes, meta)
}

// GET /api/v1/cases/:id/notes/:note_id
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	noteID, err := uuid.Parse(chi.URLParam(r, "note_id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid note id")
		return
	}
	note, err := h.svc.Get(r.Context(), noteID)
	if err != nil {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "note not found")
		return
	}
	response.JSON(w, http.StatusOK, note)
}

// PUT /api/v1/cases/:id/notes/:note_id
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	noteID, err := uuid.Parse(chi.URLParam(r, "note_id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid note id")
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())

	var req UpdateNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	note, err := h.svc.Update(r.Context(), claims, noteID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "note not found")
		case errors.Is(err, ErrForbidden):
			response.Err(w, http.StatusForbidden, "FORBIDDEN", "not authorized to edit this note")
		default:
			response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return
	}
	response.JSON(w, http.StatusOK, note)
}

// DELETE /api/v1/cases/:id/notes/:note_id
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	noteID, err := uuid.Parse(chi.URLParam(r, "note_id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid note id")
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())

	if err := h.svc.Delete(r.Context(), claims, noteID); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "note not found")
		case errors.Is(err, ErrForbidden):
			response.Err(w, http.StatusForbidden, "FORBIDDEN", "not authorized to delete this note")
		default:
			response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// POST /api/v1/cases/:id/comments
func (h *Handler) CreateComment(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())

	var req CreateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	comment, err := h.svc.CreateComment(r.Context(), claims, caseID, req)
	if err != nil {
		if errors.Is(err, ErrBadRequest) {
			response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "content is required")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, comment)
}

// GET /api/v1/cases/:id/notes/:note_id/comments
func (h *Handler) ListComments(w http.ResponseWriter, r *http.Request) {
	noteID, err := uuid.Parse(chi.URLParam(r, "note_id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid note id")
		return
	}
	comments, err := h.svc.ListCommentsByNote(r.Context(), noteID)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, comments)
}

// GET /api/v1/cases/:id/activity
func (h *Handler) ListActivity(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}
	q := r.URL.Query()
	limit := int32(25)
	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = int32(n)
		}
	}
	cursor := uuid.Nil
	if c := q.Get("cursor"); c != "" {
		if id, err := uuid.Parse(c); err == nil {
			cursor = id
		}
	}

	items, meta, err := h.svc.ListActivity(r.Context(), caseID, cursor, limit)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.JSONWithMeta(w, http.StatusOK, items, meta)
}

// PUT /api/v1/mentions/:id/read
func (h *Handler) MarkMentionRead(w http.ResponseWriter, r *http.Request) {
	mentionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid mention id")
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	if err := h.svc.MarkMentionRead(r.Context(), claims, mentionID); err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "read"})
}

// GET /api/v1/mentions/unread-count
func (h *Handler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	count, err := h.svc.UnreadCount(r.Context(), claims)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]int64{"unread_count": count})
}

// GET /api/v1/cases/:id/notes/stream  — SSE for real-time note/comment updates
func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
	caseIDStr := chi.URLParam(r, "id")
	if _, err := uuid.Parse(caseIDStr); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	// Auth via ?token= query param (EventSource cannot set headers)
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if _, err := h.jwtMgr.Verify(token); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch, unsub := h.broker.Subscribe(caseIDStr)
	defer unsub()

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Send keep-alive ping every 25 seconds
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	_, _ = fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case data := <-ch:
			_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-ticker.C:
			_, _ = fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}
