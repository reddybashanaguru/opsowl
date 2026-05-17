package audit

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

type Handler struct {
	queries *db.Queries
}

func NewHandler(queries *db.Queries) *Handler {
	return &Handler{queries: queries}
}

type AuditLogEntry struct {
	ID           uuid.UUID       `json:"id"`
	EventCode    string          `json:"event_code"`
	ActorID      *uuid.UUID      `json:"actor_id,omitempty"`
	ActorName    *string         `json:"actor_name,omitempty"`
	ActorEmail   *string         `json:"actor_email,omitempty"`
	IPAddress    *string         `json:"ip_address,omitempty"`
	ResourceType *string         `json:"resource_type,omitempty"`
	ResourceID   *uuid.UUID      `json:"resource_id,omitempty"`
	Metadata     json.RawMessage `json:"metadata"`
	Severity     string          `json:"severity"`
	CreatedAt    time.Time       `json:"created_at"`
}

// Cursor is encoded as "RFC3339Nano|uuid" for stable (created_at, id) pagination.
func encodeCursor(t time.Time, id uuid.UUID) string {
	return fmt.Sprintf("%s|%s", t.UTC().Format(time.RFC3339Nano), id.String())
}

func decodeCursor(s string) (sql.NullTime, uuid.NullUUID) {
	parts := strings.SplitN(s, "|", 2)
	if len(parts) != 2 {
		return sql.NullTime{}, uuid.NullUUID{}
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return sql.NullTime{}, uuid.NullUUID{}
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return sql.NullTime{}, uuid.NullUUID{}
	}
	return sql.NullTime{Time: t, Valid: true}, uuid.NullUUID{UUID: id, Valid: true}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit := int32(50)
	if lStr := q.Get("limit"); lStr != "" {
		if l, err := strconv.ParseInt(lStr, 10, 32); err == nil && l > 0 && l <= 200 {
			limit = int32(l)
		}
	}

	var severity, eventCode sql.NullString
	var actorID uuid.NullUUID
	var cursorTime sql.NullTime
	var cursorID uuid.NullUUID

	if s := q.Get("severity"); s != "" {
		severity = sql.NullString{String: s, Valid: true}
	}
	if ec := q.Get("event_code"); ec != "" {
		eventCode = sql.NullString{String: ec, Valid: true}
	}
	if aStr := q.Get("actor_id"); aStr != "" {
		if id, err := uuid.Parse(aStr); err == nil {
			actorID = uuid.NullUUID{UUID: id, Valid: true}
		}
	}
	if cStr := q.Get("cursor"); cStr != "" {
		cursorTime, cursorID = decodeCursor(cStr)
	}

	rows, err := h.queries.ListAuditLogs(r.Context(), db.ListAuditLogsParams{
		Severity:   severity,
		EventCode:  eventCode,
		ActorID:    actorID,
		CursorTime: cursorTime,
		CursorID:   cursorID,
		RowLimit:   limit + 1,
	})
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load audit logs")
		return
	}

	hasMore := len(rows) > int(limit)
	if hasMore {
		rows = rows[:limit]
	}

	var nextCursor *string
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		c := encodeCursor(last.CreatedAt, last.ID)
		nextCursor = &c
	}

	out := make([]AuditLogEntry, 0, len(rows))
	for _, row := range rows {
		e := AuditLogEntry{
			ID:        row.ID,
			EventCode: row.EventCode,
			Metadata:  row.Metadata,
			Severity:  row.Severity,
			CreatedAt: row.CreatedAt,
		}
		if row.ActorID.Valid {
			id := row.ActorID.UUID
			e.ActorID = &id
		}
		if row.ActorName.Valid {
			e.ActorName = &row.ActorName.String
		}
		if row.ActorEmail.Valid {
			e.ActorEmail = &row.ActorEmail.String
		}
		if row.IpAddress.Valid {
			e.IPAddress = &row.IpAddress.String
		}
		if row.ResourceType.Valid {
			e.ResourceType = &row.ResourceType.String
		}
		if row.ResourceID.Valid {
			id := row.ResourceID.UUID
			e.ResourceID = &id
		}
		out = append(out, e)
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"logs":        out,
		"has_more":    hasMore,
		"next_cursor": nextCursor,
	})
}

func (h *Handler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	rows, err := h.queries.ListAuditLogs(r.Context(), db.ListAuditLogsParams{
		RowLimit: 5000,
	})
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "export failed")
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=audit_logs.csv")

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"id", "event_code", "severity", "actor_email", "ip_address", "resource_type", "created_at"})
	for _, row := range rows {
		actorEmail := ""
		if row.ActorEmail.Valid {
			actorEmail = row.ActorEmail.String
		}
		ipAddr := ""
		if row.IpAddress.Valid {
			ipAddr = row.IpAddress.String
		}
		resType := ""
		if row.ResourceType.Valid {
			resType = row.ResourceType.String
		}
		_ = cw.Write([]string{
			row.ID.String(),
			row.EventCode,
			row.Severity,
			actorEmail,
			ipAddr,
			resType,
			row.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	cw.Flush()
}

func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	total, err := h.queries.CountAuditLogs(r.Context(), db.CountAuditLogsParams{})
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load stats")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"total": total})
}
