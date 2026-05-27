package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

// backoffSchedule controls retry delays after each failure.
// 5 attempts: immediately, +30s, +5m, +30m, +2h → then exhausted.
var backoffSchedule = []time.Duration{
	0,
	30 * time.Second,
	5 * time.Minute,
	30 * time.Minute,
	2 * time.Hour,
}

const maxAttempts = int32(5)

// Dispatcher is the interface timeline.DBEmitter calls to fan out events.
type Dispatcher interface {
	Dispatch(ctx context.Context, eventType string, caseID *uuid.UUID, payload map[string]any)
}

type Service struct {
	q      *db.Queries
	logger *zap.Logger
	client *http.Client
}

func NewService(q *db.Queries, logger *zap.Logger) *Service {
	return &Service{
		q:      q,
		logger: logger,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// ─── CRUD ──────────────────────────────────────────────────────────────────────

func (s *Service) Create(ctx context.Context, req CreateWebhookRequest, actorID *uuid.UUID) (*Webhook, error) {
	if err := validateRequest(req.URL, req.SigningSecret); err != nil {
		return nil, err
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	var createdBy uuid.NullUUID
	if actorID != nil {
		createdBy = uuid.NullUUID{UUID: *actorID, Valid: true}
	}
	row, err := s.q.InsertWebhook(ctx, db.InsertWebhookParams{
		Name:          req.Name,
		URL:           req.URL,
		SigningSecret: req.SigningSecret,
		EventTypes:    normalizeEventTypes(req.EventTypes),
		IsActive:      active,
		CreatedBy:     createdBy,
	})
	if err != nil {
		return nil, err
	}
	return webhookFromRow(row, true), nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Webhook, error) {
	row, err := s.q.GetWebhook(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return webhookFromRow(row, true), nil
}

func (s *Service) List(ctx context.Context) ([]*Webhook, error) {
	rows, err := s.q.ListWebhooks(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*Webhook, len(rows))
	for i, r := range rows {
		out[i] = webhookFromRow(r, false)
	}
	return out, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateWebhookRequest) (*Webhook, error) {
	existing, err := s.q.GetWebhook(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	if err := validateRequest(req.URL, req.SigningSecret); err != nil {
		return nil, err
	}
	active := existing.IsActive
	if req.IsActive != nil {
		active = *req.IsActive
	}
	row, err := s.q.UpdateWebhook(ctx, db.UpdateWebhookParams{
		ID:            id,
		Name:          req.Name,
		URL:           req.URL,
		SigningSecret: req.SigningSecret,
		EventTypes:    normalizeEventTypes(req.EventTypes),
		IsActive:      active,
	})
	if err != nil {
		return nil, err
	}
	return webhookFromRow(row, true), nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.q.GetWebhook(ctx, id)
	if err != nil {
		return ErrNotFound
	}
	return s.q.DeleteWebhook(ctx, id)
}

func (s *Service) ListDeliveries(ctx context.Context, webhookID uuid.UUID, limit, offset int32) (*DeliveriesPage, error) {
	if limit <= 0 {
		limit = 25
	}
	rows, err := s.q.ListDeliveriesByWebhook(ctx, db.ListDeliveriesByWebhookParams{
		WebhookID: webhookID,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, err
	}
	total, _ := s.q.CountDeliveriesByWebhook(ctx, webhookID)
	deliveries := make([]*Delivery, len(rows))
	for i, r := range rows {
		deliveries[i] = deliveryFromRow(r)
	}
	return &DeliveriesPage{
		Deliveries: deliveries,
		HasMore:    int64(offset)+int64(len(rows)) < total,
		Total:      total,
	}, nil
}

func (s *Service) Redeliver(ctx context.Context, deliveryID uuid.UUID) error {
	_, err := s.q.RequeueDelivery(ctx, deliveryID)
	return err
}

// ─── Dispatch (called by emitter, non-blocking) ────────────────────────────────

func (s *Service) Dispatch(ctx context.Context, eventType string, caseID *uuid.UUID, payload map[string]any) {
	hooks, err := s.q.ListActiveWebhooksForEvent(context.Background(), eventType)
	if err != nil || len(hooks) == 0 {
		return
	}
	raw, _ := json.Marshal(payload)
	for _, hook := range hooks {
		var cid uuid.NullUUID
		if caseID != nil {
			cid = uuid.NullUUID{UUID: *caseID, Valid: true}
		}
		delivery, err := s.q.InsertDelivery(context.Background(), db.InsertDeliveryParams{
			WebhookID: hook.ID,
			EventType: eventType,
			CaseID:    cid,
			Payload:   raw,
		})
		if err != nil {
			s.logger.Error("webhook: failed to queue delivery", zap.Error(err))
			continue
		}
		go s.deliver(hook, delivery)
	}
}

// ─── Background worker ─────────────────────────────────────────────────────────

func (s *Service) StartWorker(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runBatch(ctx)
		}
	}
}

func (s *Service) runBatch(ctx context.Context) {
	pending, err := s.q.GetPendingDeliveries(ctx)
	if err != nil {
		s.logger.Error("webhook worker: failed to fetch pending", zap.Error(err))
		return
	}
	for _, delivery := range pending {
		hook, err := s.q.GetWebhook(ctx, delivery.WebhookID)
		if err != nil {
			continue
		}
		delivery := delivery
		go s.deliver(hook, delivery)
	}
}

// ─── Delivery ──────────────────────────────────────────────────────────────────

func (s *Service) deliver(hook db.OutboundWebhook, delivery db.OutboundWebhookDelivery) {
	ctx := context.Background()

	body := buildBody(delivery)
	sig := sign(hook.SigningSecret, body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hook.URL, bytes.NewReader(body))
	if err != nil {
		s.markFailed(ctx, delivery, 0, err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-OpsOwl-Signature", "sha256="+sig)
	req.Header.Set("X-OpsOwl-Event", delivery.EventType)
	req.Header.Set("X-OpsOwl-Delivery", delivery.ID.String())

	resp, err := s.client.Do(req)
	if err != nil {
		s.markFailed(ctx, delivery, 0, err.Error())
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		s.q.MarkDeliverySuccess(ctx, delivery.ID, sql.NullInt32{Int32: int32(resp.StatusCode), Valid: true}) //nolint
		return
	}
	s.markFailed(ctx, delivery, resp.StatusCode, fmt.Sprintf("non-2xx status: %d", resp.StatusCode))
}

func (s *Service) markFailed(ctx context.Context, delivery db.OutboundWebhookDelivery, status int, errMsg string) {
	attempt := delivery.AttemptCount
	var delay time.Duration
	if int(attempt) < len(backoffSchedule) {
		delay = backoffSchedule[attempt]
	} else {
		delay = backoffSchedule[len(backoffSchedule)-1]
	}

	var lastStatus sql.NullInt32
	if status > 0 {
		lastStatus = sql.NullInt32{Int32: int32(status), Valid: true}
	}
	s.q.MarkDeliveryFailed(ctx, db.MarkDeliveryFailedParams{ //nolint
		ID:            delivery.ID,
		LastStatus:    lastStatus,
		NextAttemptAt: time.Now().Add(delay),
		MaxAttempts:   maxAttempts,
		LastError:     sql.NullString{String: errMsg, Valid: true},
	})
}

// ─── Helpers ───────────────────────────────────────────────────────────────────

func buildBody(d db.OutboundWebhookDelivery) []byte {
	env := map[string]any{
		"id":         d.ID.String(),
		"event_type": d.EventType,
		"payload":    json.RawMessage(d.Payload),
		"timestamp":  d.CreatedAt.UTC().Format(time.RFC3339),
	}
	if d.CaseID.Valid {
		env["case_id"] = d.CaseID.UUID.String()
	}
	out, _ := json.Marshal(env)
	return out
}

func sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func validateRequest(rawURL, secret string) error {
	if !strings.HasPrefix(rawURL, "https://") {
		return ErrInvalidURL
	}
	if len(secret) < 16 {
		return ErrInvalidSecret
	}
	return nil
}

func normalizeEventTypes(types []string) []string {
	if len(types) == 0 {
		return []string{}
	}
	return types
}

// GenerateSecret produces a cryptographically random 32-byte hex secret.
func GenerateSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

