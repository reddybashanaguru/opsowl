package timeline

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
	hub    *Hub
	jwtMgr *auth.JWTManager
}

func NewHandler(svc *Service, hub *Hub, jwtMgr *auth.JWTManager) *Handler {
	return &Handler{svc: svc, hub: hub, jwtMgr: jwtMgr}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	q := r.URL.Query()
	f := ListFilters{Limit: 50}

	if et := q.Get("event_type"); et != "" {
		t := db.TimelineEventType(et)
		f.EventType = &t
	}
	if src := q.Get("source"); src != "" {
		s := db.TimelineEventSource(src)
		f.Source = &s
	}
	if a := q.Get("actor_id"); a != "" {
		if id, err := uuid.Parse(a); err == nil {
			f.ActorID = &id
		}
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
	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			f.Limit = int32(n)
		}
	}
	if cursor := q.Get("cursor"); cursor != "" {
		if ct, cid, err := decodeCursor(cursor); err == nil {
			f.CursorTime = ct
			f.CursorID = cid
		}
	}

	events, meta, err := h.svc.List(r.Context(), caseID, f)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list timeline")
		return
	}
	response.JSONWithMeta(w, http.StatusOK, events, meta)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	var req CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.OccurredAt.IsZero() {
		req.OccurredAt = time.Now()
	}
	evt, err := h.svc.Create(r.Context(), claims, caseID, req)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_EVENT", err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, evt)
}

// Stream serves the SSE endpoint. JWT is read from ?token= query param
// because SSE (EventSource API) cannot set custom headers.
func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	// Validate JWT from query param
	token := r.URL.Query().Get("token")
	if token == "" {
		response.Err(w, http.StatusUnauthorized, "MISSING_TOKEN", "token required")
		return
	}
	if _, err := h.jwtMgr.Verify(token); err != nil {
		response.Err(w, http.StatusUnauthorized, "INVALID_TOKEN", "invalid token")
		return
	}

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}

	// Subscribe to live events before replaying so no events are missed between
	// the replay query and the subscription being registered.
	client, unsub := h.hub.Subscribe(caseID.String())
	defer unsub()

	// Replay missed events if the client sends Last-Event-ID on reconnect.
	if lastID := r.Header.Get("Last-Event-ID"); lastID != "" {
		if id, err := uuid.Parse(lastID); err == nil {
			if missed, err := h.svc.repo.GetEventsSince(r.Context(), caseID, id); err == nil {
				for _, evt := range missed {
					resp := toEventResponse(evt)
					b, _ := json.Marshal(resp)
					fmt.Fprintf(w, "id: %s\nevent: timeline_event\ndata: %s\n\n", evt.ID, b)
				}
				if len(missed) > 0 {
					flusher.Flush()
				}
			}
		}
	}

	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	// Send connected confirmation
	fmt.Fprintf(w, "event: connected\ndata: {\"case_id\":\"%s\"}\n\n", caseID)
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case evt := <-client.Ch:
			b, _ := json.Marshal(evt)
			fmt.Fprintf(w, "id: %s\nevent: timeline_event\ndata: %s\n\n", evt.ID, b)
			flusher.Flush()
		case t := <-heartbeat.C:
			fmt.Fprintf(w, "event: ping\ndata: {\"ts\":\"%s\"}\n\n", t.UTC().Format(time.RFC3339))
			flusher.Flush()
		}
	}
}
