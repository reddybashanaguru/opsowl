package notifications

import (
	"encoding/json"
	"fmt"
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
	svc    *Service
	broker *UserBroker
	jwtMgr *auth.JWTManager
}

func NewHandler(svc *Service, broker *UserBroker, jwtMgr *auth.JWTManager) *Handler {
	return &Handler{svc: svc, broker: broker, jwtMgr: jwtMgr}
}

// GET /api/v1/notifications
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
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

	items, meta, err := h.svc.List(r.Context(), claims.UserID, cursor, limit)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.JSONWithMeta(w, http.StatusOK, items, meta)
}

// GET /api/v1/notifications/unread-count
func (h *Handler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	count, err := h.svc.UnreadCount(r.Context(), claims.UserID)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]int64{"unread_count": count})
}

// PUT /api/v1/notifications/:id/read
func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	notifID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid notification id")
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	if err := h.svc.MarkRead(r.Context(), claims.UserID, notifID); err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "read"})
}

// PUT /api/v1/notifications/read-all
func (h *Handler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	if err := h.svc.MarkAllRead(r.Context(), claims.UserID); err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GET /api/v1/notifications/preferences
func (h *Handler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	prefs, err := h.svc.GetPreferences(r.Context(), claims.UserID)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, prefs)
}

// PUT /api/v1/notifications/preferences/:type
func (h *Handler) UpdatePreference(w http.ResponseWriter, r *http.Request) {
	notifType := db.NotificationType(chi.URLParam(r, "type"))
	claims := auth.ClaimsFromCtx(r.Context())

	var req struct {
		InAppEnabled bool `json:"in_app_enabled"`
		EmailEnabled bool `json:"email_enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if err := h.svc.UpsertPreference(r.Context(), claims.UserID, notifType, req.InAppEnabled, req.EmailEnabled); err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// GET /api/v1/notifications/stream — SSE per-user notification stream
func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	claims, err := h.jwtMgr.Verify(token)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch, unsub := h.broker.Subscribe(claims.UserID.String())
	defer unsub()

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

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
