package webhooks

import (
	"errors"
	"time"

	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

var (
	ErrNotFound      = errors.New("webhook not found")
	ErrInvalidURL    = errors.New("webhook URL must start with https://")
	ErrInvalidSecret = errors.New("signing secret must be at least 16 characters")
)

// KnownEventTypes lists all event types the delivery worker can dispatch.
var KnownEventTypes = []string{
	"case.created",
	"case.status_changed",
	"case.assigned",
	"case.closed",
	"case.str_filed",
	"document.uploaded",
	"note.added",
	"task.completed",
}

type Webhook struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	URL           string    `json:"url"`
	SigningSecret string    `json:"signing_secret,omitempty"`
	EventTypes    []string  `json:"event_types"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Delivery struct {
	ID            uuid.UUID  `json:"id"`
	WebhookID     uuid.UUID  `json:"webhook_id"`
	EventType     string     `json:"event_type"`
	CaseID        *uuid.UUID `json:"case_id,omitempty"`
	Status        string     `json:"status"`
	AttemptCount  int32      `json:"attempt_count"`
	NextAttemptAt time.Time  `json:"next_attempt_at"`
	LastStatus    *int32     `json:"last_status,omitempty"`
	LastError     *string    `json:"last_error,omitempty"`
	DeliveredAt   *time.Time `json:"delivered_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type DeliveriesPage struct {
	Deliveries []*Delivery `json:"deliveries"`
	HasMore    bool        `json:"has_more"`
	Total      int64       `json:"total"`
}

type CreateWebhookRequest struct {
	Name          string   `json:"name"`
	URL           string   `json:"url"`
	SigningSecret string   `json:"signing_secret"`
	EventTypes    []string `json:"event_types"`
	IsActive      *bool    `json:"is_active"`
}

type UpdateWebhookRequest struct {
	Name          string   `json:"name"`
	URL           string   `json:"url"`
	SigningSecret string   `json:"signing_secret"`
	EventTypes    []string `json:"event_types"`
	IsActive      *bool    `json:"is_active"`
}

func webhookFromRow(r db.OutboundWebhook, reveal bool) *Webhook {
	w := &Webhook{
		ID:         r.ID,
		Name:       r.Name,
		URL:        r.URL,
		EventTypes: r.EventTypes,
		IsActive:   r.IsActive,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
	if reveal {
		w.SigningSecret = r.SigningSecret
	}
	return w
}

func deliveryFromRow(r db.OutboundWebhookDelivery) *Delivery {
	d := &Delivery{
		ID:            r.ID,
		WebhookID:     r.WebhookID,
		EventType:     r.EventType,
		Status:        r.Status,
		AttemptCount:  r.AttemptCount,
		NextAttemptAt: r.NextAttemptAt,
		CreatedAt:     r.CreatedAt,
	}
	if r.CaseID.Valid {
		id := r.CaseID.UUID
		d.CaseID = &id
	}
	if r.LastStatus.Valid {
		v := r.LastStatus.Int32
		d.LastStatus = &v
	}
	if r.LastError.Valid {
		d.LastError = &r.LastError.String
	}
	if r.DeliveredAt.Valid {
		t := r.DeliveredAt.Time
		d.DeliveredAt = &t
	}
	return d
}
