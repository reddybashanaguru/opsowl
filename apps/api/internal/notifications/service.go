package notifications

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/reddybashanaguru/opsowl/internal/mail"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

// Service is the single point for creating and delivering notifications.
// Other modules call Notify() — they never write to the notifications table directly.
type Service struct {
	q      *db.Queries
	broker *UserBroker
	mailer *mail.Sender
}

func NewService(q *db.Queries, broker *UserBroker) *Service {
	return &Service{q: q, broker: broker}
}

// WithMailer attaches an SMTP sender for email delivery of notifications.
func (s *Service) WithMailer(m *mail.Sender) *Service {
	s.mailer = m
	return s
}

// NotifyInput is the data any module passes to create a notification.
type NotifyInput struct {
	RecipientID uuid.UUID
	Type        db.NotificationType
	Priority    db.NotificationPriority
	Title       string
	Message     string
	LinkURL     string
	CaseID      *uuid.UUID
	TaskID      *uuid.UUID
}

// NotificationResponse is the JSON-serialisable form sent to the frontend.
type NotificationResponse struct {
	ID          string    `json:"id"`
	RecipientID string    `json:"recipient_id"`
	Type        string    `json:"type"`
	Priority    string    `json:"priority"`
	Title       string    `json:"title"`
	Message     string    `json:"message"`
	LinkURL     *string   `json:"link_url,omitempty"`
	IsRead      bool      `json:"is_read"`
	ReadAt      *string   `json:"read_at,omitempty"`
	CaseID      *string   `json:"case_id,omitempty"`
	TaskID      *string   `json:"task_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type ListMeta struct {
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
}

// Notify creates a notification and pushes it over SSE to the recipient if connected.
// It runs synchronously but is designed to be called from a goroutine when non-blocking is desired.
func (s *Service) Notify(ctx context.Context, in NotifyInput) error {
	params := db.CreateNotificationParams{
		RecipientID: in.RecipientID,
		Type:        in.Type,
		Priority:    in.Priority,
		Title:       in.Title,
		Message:     in.Message,
	}
	if in.LinkURL != "" {
		params.LinkUrl = sql.NullString{String: in.LinkURL, Valid: true}
	}
	if in.CaseID != nil {
		params.CaseID = uuid.NullUUID{UUID: *in.CaseID, Valid: true}
	}
	if in.TaskID != nil {
		params.TaskID = uuid.NullUUID{UUID: *in.TaskID, Valid: true}
	}

	n, err := s.q.CreateNotification(ctx, params)
	if err != nil {
		return err
	}

	// Push to any connected SSE clients for this user
	s.broker.Push(in.RecipientID.String(), toResponse(n))

	// Email delivery — only if the user has email_enabled for this notification type
	if s.mailer != nil {
		go s.deliverEmail(ctx, in, n) //nolint:errcheck
	}
	return nil
}

// deliverEmail checks the user's email preference for this notification type
// and sends an email if enabled.
func (s *Service) deliverEmail(ctx context.Context, in NotifyInput, n db.Notification) {
	prefs, err := s.q.ListNotificationPreferences(ctx, in.RecipientID)
	if err != nil {
		return
	}
	for _, p := range prefs {
		if p.NotificationType == in.Type && p.EmailEnabled {
			// Fetch recipient email address
			user, err := s.q.GetUserByID(ctx, in.RecipientID)
			if err != nil {
				return
			}
			body := fmt.Sprintf("%s\n\n%s\n\n---\nThis is an automated notification from OpsOwl AI.", in.Title, in.Message)
			if in.LinkURL != "" {
				body += fmt.Sprintf("\n\nView: %s", in.LinkURL)
			}
			_ = s.mailer.Send(mail.Message{
				To:      []string{user.Email},
				Subject: fmt.Sprintf("[OpsOwl] %s", in.Title),
				Body:    body,
			})
			return
		}
	}
}

// NotifyAsync fires Notify in a goroutine — use when the caller cannot block.
func (s *Service) NotifyAsync(ctx context.Context, in NotifyInput) {
	go func() {
		_ = s.Notify(context.Background(), in)
	}()
}

func (s *Service) List(ctx context.Context, recipientID uuid.UUID, cursor uuid.UUID, limit int32) ([]NotificationResponse, ListMeta, error) {
	if limit <= 0 || limit > 50 {
		limit = 25
	}
	rows, err := s.q.ListNotifications(ctx, db.ListNotificationsParams{
		RecipientID: recipientID,
		Column2:     cursor,
		Limit:       limit + 1,
	})
	if err != nil {
		return nil, ListMeta{}, err
	}

	hasMore := len(rows) > int(limit)
	if hasMore {
		rows = rows[:limit]
	}
	var nextCursor *string
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1].ID.String()
		nextCursor = &last
	}

	out := make([]NotificationResponse, len(rows))
	for i, r := range rows {
		out[i] = toResponse(r)
	}
	return out, ListMeta{NextCursor: nextCursor, HasMore: hasMore}, nil
}

func (s *Service) UnreadCount(ctx context.Context, recipientID uuid.UUID) (int64, error) {
	return s.q.CountUnreadNotifications(ctx, recipientID)
}

func (s *Service) MarkRead(ctx context.Context, recipientID uuid.UUID, notifID uuid.UUID) error {
	_, err := s.q.MarkNotificationRead(ctx, db.MarkNotificationReadParams{
		ID:          notifID,
		RecipientID: recipientID,
	})
	return err
}

func (s *Service) MarkAllRead(ctx context.Context, recipientID uuid.UUID) error {
	return s.q.MarkAllNotificationsRead(ctx, recipientID)
}

func (s *Service) GetPreferences(ctx context.Context, userID uuid.UUID) ([]db.ListNotificationPreferencesRow, error) {
	return s.q.ListNotificationPreferences(ctx, userID)
}

func (s *Service) UpsertPreference(ctx context.Context, userID uuid.UUID, notifType db.NotificationType, inApp, email bool) error {
	_, err := s.q.UpsertNotificationPreference(ctx, db.UpsertNotificationPreferenceParams{
		UserID:           userID,
		NotificationType: notifType,
		InAppEnabled:     inApp,
		EmailEnabled:     email,
	})
	return err
}

// ── helpers ──────────────────────────────────────────────────────────────────

func toResponse(n db.Notification) NotificationResponse {
	r := NotificationResponse{
		ID:          n.ID.String(),
		RecipientID: n.RecipientID.String(),
		Type:        string(n.Type),
		Priority:    string(n.Priority),
		Title:       n.Title,
		Message:     n.Message,
		IsRead:      n.IsRead,
		CreatedAt:   n.CreatedAt,
	}
	if n.LinkUrl.Valid {
		r.LinkURL = &n.LinkUrl.String
	}
	if n.ReadAt.Valid {
		s := n.ReadAt.Time.Format(time.RFC3339)
		r.ReadAt = &s
	}
	if n.CaseID.Valid {
		s := n.CaseID.UUID.String()
		r.CaseID = &s
	}
	if n.TaskID.Valid {
		s := n.TaskID.UUID.String()
		r.TaskID = &s
	}
	return r
}
