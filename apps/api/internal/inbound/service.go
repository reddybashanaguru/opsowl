package inbound

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/reddybashanaguru/opsowl/internal/auth"
	"github.com/reddybashanaguru/opsowl/internal/cases"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

var (
	ErrSourceNotFound    = errors.New("inbound source not found")
	ErrSourceInactive    = errors.New("inbound source is disabled")
	ErrInvalidSignature  = errors.New("invalid HMAC signature")
	ErrDuplicateEvent    = errors.New("duplicate event — idempotency key already processed")
	ErrRuleNotFound      = errors.New("rule not found")
	ErrNoMatchingRule    = errors.New("no matching rule for event_type")
)

// Source represents a registered external system that can send events to OpsOwl.
type Source struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	Description   string    `json:"description"`
	SigningSecret string    `json:"signing_secret,omitempty"` // only included on create
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Rule tells OpsOwl how to turn a matching inbound event into a case.
type Rule struct {
	ID                  uuid.UUID  `json:"id"`
	SourceID            uuid.UUID  `json:"source_id"`
	EventTypePattern    string     `json:"event_type_pattern"`
	CasePriority        string     `json:"case_priority"`
	Tags                []string   `json:"tags"`
	TitleTemplate       string     `json:"title_template"`
	DescriptionTemplate string     `json:"description_template"`
	AutoAssignUserID    *uuid.UUID `json:"auto_assign_user_id"`
	IsActive            bool       `json:"is_active"`
	SortOrder           int        `json:"sort_order"`
	CreatedAt           time.Time  `json:"created_at"`
}

// Event is one raw inbound event (before or after processing).
type Event struct {
	ID             uuid.UUID       `json:"id"`
	SourceID       uuid.UUID       `json:"source_id"`
	SourceName     string          `json:"source_name"`
	IdempotencyKey string          `json:"idempotency_key"`
	EventType      string          `json:"event_type"`
	Payload        json.RawMessage `json:"payload"`
	Status         string          `json:"status"`
	CaseID         *uuid.UUID      `json:"case_id"`
	ErrorMessage   *string         `json:"error_message"`
	AttemptCount   int             `json:"attempt_count"`
	ProcessedAt    *time.Time      `json:"processed_at"`
	CreatedAt      time.Time       `json:"created_at"`
}

// IncomingPayload is the JSON body the caller sends to POST /inbound/events.
type IncomingPayload struct {
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
}

type Service struct {
	db       *sql.DB
	casesSvc *cases.Service
	logger   *zap.Logger
	queue    chan uuid.UUID // event IDs waiting to be processed into cases
}

func NewService(sqlDB *sql.DB, casesSvc *cases.Service, logger *zap.Logger) *Service {
	s := &Service{
		db:       sqlDB,
		casesSvc: casesSvc,
		logger:   logger,
		queue:    make(chan uuid.UUID, 512),
	}
	// Two background workers pick events off the queue and create cases.
	for i := 0; i < 2; i++ {
		go s.worker()
	}
	return s
}

// worker drains the processing queue indefinitely.
func (s *Service) worker() {
	for eventID := range s.queue {
		if err := s.processEvent(context.Background(), eventID); err != nil {
			s.logger.Error("inbound event processing failed",
				zap.String("event_id", eventID.String()),
				zap.Error(err))
		}
	}
}

// ─── Receiving ───────────────────────────────────────────────────────────────

// Receive validates an inbound HTTP event and queues it for async case creation.
// Returns the event ID on success; ErrDuplicateEvent if idempotency key was seen before.
func (s *Service) Receive(ctx context.Context, sourceSlug, signature, idempotencyKey string, body []byte) (uuid.UUID, bool, error) {
	src, err := s.getSourceBySlug(ctx, sourceSlug)
	if err != nil {
		return uuid.Nil, false, ErrSourceNotFound
	}
	if !src.IsActive {
		return uuid.Nil, false, ErrSourceInactive
	}

	// ── HMAC validation ───────────────────────────────────────────────────
	// The caller must send X-OpsOwl-Signature: sha256=<hex(HMAC-SHA256(secret, body))>
	// This proves the request came from a holder of the signing secret.
	if !validateHMAC(src.SigningSecret, body, signature) {
		return uuid.Nil, false, ErrInvalidSignature
	}

	var incoming IncomingPayload
	if err := json.Unmarshal(body, &incoming); err != nil {
		return uuid.Nil, false, fmt.Errorf("invalid JSON body: %w", err)
	}
	if incoming.EventType == "" {
		return uuid.Nil, false, fmt.Errorf("event_type is required")
	}
	if incoming.Payload == nil {
		incoming.Payload = json.RawMessage("{}")
	}

	payload, _ := json.Marshal(incoming.Payload)

	// ── Idempotency insert ────────────────────────────────────────────────
	// ON CONFLICT DO NOTHING means a re-submitted event is silently ignored.
	var eventID uuid.UUID
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO inbound_events (source_id, idempotency_key, event_type, payload)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (source_id, idempotency_key) DO NOTHING
		RETURNING id`,
		src.ID, idempotencyKey, incoming.EventType, payload,
	).Scan(&eventID)

	if errors.Is(err, sql.ErrNoRows) {
		// Conflict — this idempotency key was already processed.
		return uuid.Nil, true, ErrDuplicateEvent
	}
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("failed to store event: %w", err)
	}

	// Queue for async processing — non-blocking.
	select {
	case s.queue <- eventID:
	default:
		s.logger.Warn("inbound processing queue full — event will be retried on next boot",
			zap.String("event_id", eventID.String()))
	}

	return eventID, false, nil
}

// processEvent finds the matching rule and creates a case for the event.
func (s *Service) processEvent(ctx context.Context, eventID uuid.UUID) error {
	// Mark as processing (increment attempt_count).
	if _, err := s.db.ExecContext(ctx, `
		UPDATE inbound_events
		SET attempt_count = attempt_count + 1, updated_at = NOW()
		WHERE id = $1`, eventID); err != nil {
		return err
	}

	// Fetch the event + source in one query.
	var (
		sourceID       uuid.UUID
		sourceSlug     string
		sourceName     string
		signingSecret  string
		eventType      string
		payloadBytes   []byte
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT e.source_id, s.slug, s.name, s.signing_secret, e.event_type, e.payload
		FROM inbound_events e
		JOIN inbound_sources s ON s.id = e.source_id
		WHERE e.id = $1`, eventID).Scan(
		&sourceID, &sourceSlug, &sourceName, &signingSecret, &eventType, &payloadBytes,
	)
	if err != nil {
		return s.failEvent(ctx, eventID, fmt.Sprintf("could not load event: %v", err))
	}

	// Find matching rule.
	rule, err := s.findMatchingRule(ctx, sourceID, eventType)
	if err != nil {
		return s.failEvent(ctx, eventID, fmt.Sprintf("no matching rule for event_type=%q", eventType))
	}

	// Render title + description from templates.
	var payload map[string]any
	_ = json.Unmarshal(payloadBytes, &payload)
	title := renderTemplate(rule.TitleTemplate, eventType, sourceName, payload)
	description := renderTemplate(rule.DescriptionTemplate, eventType, sourceName, payload)

	// Build synthetic claims so the case is attributed to the source creator
	// (or a system admin if the source has no creator).
	creatorID, creatorRole := s.resolveCreator(ctx, sourceID)
	syntheticClaims := &auth.Claims{
		UserID: creatorID,
		Email:  "inbound-system@opsowl",
		Role:   creatorRole,
	}

	priority := db.CasePriority(rule.CasePriority)
	req := cases.CreateCaseRequest{
		Title:       title,
		Description: description,
		Priority:    priority,
		Tags:        append(rule.Tags, "inbound", sourceSlug),
	}
	if rule.AutoAssignUserID != nil {
		req.AssignedTo = rule.AutoAssignUserID
	}

	caseResp, err := s.casesSvc.Create(ctx, syntheticClaims, req)
	if err != nil {
		return s.failEvent(ctx, eventID, fmt.Sprintf("case creation failed: %v", err))
	}

	// Mark as processed.
	now := time.Now()
	_, err = s.db.ExecContext(ctx, `
		UPDATE inbound_events
		SET status = 'processed', case_id = $2, processed_at = $3, updated_at = NOW()
		WHERE id = $1`, eventID, caseResp.ID, now)
	return err
}

func (s *Service) failEvent(ctx context.Context, eventID uuid.UUID, msg string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE inbound_events
		SET status = 'failed', error_message = $2, updated_at = NOW()
		WHERE id = $1`, eventID, msg)
	if err != nil {
		s.logger.Error("failed to mark event as failed", zap.Error(err))
	}
	return fmt.Errorf(msg)
}

// ─── Source management ───────────────────────────────────────────────────────

type CreateSourceRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

type UpdateSourceRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"is_active"`
}

func (s *Service) CreateSource(ctx context.Context, creatorID uuid.UUID, req CreateSourceRequest) (*Source, error) {
	if req.Name == "" || req.Slug == "" {
		return nil, fmt.Errorf("name and slug are required")
	}
	secret, err := generateSecret()
	if err != nil {
		return nil, err
	}

	var src Source
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO inbound_sources (name, slug, description, signing_secret, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, slug, description, signing_secret, is_active, created_at, updated_at`,
		req.Name, req.Slug, req.Description, secret, creatorID,
	).Scan(&src.ID, &src.Name, &src.Slug, &src.Description, &src.SigningSecret,
		&src.IsActive, &src.CreatedAt, &src.UpdatedAt)
	return &src, err
}

func (s *Service) ListSources(ctx context.Context) ([]*Source, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, slug, description, is_active, created_at, updated_at
		FROM inbound_sources
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Source
	for rows.Next() {
		var src Source
		if err := rows.Scan(&src.ID, &src.Name, &src.Slug, &src.Description,
			&src.IsActive, &src.CreatedAt, &src.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &src)
	}
	return out, rows.Err()
}

func (s *Service) GetSource(ctx context.Context, id uuid.UUID) (*Source, error) {
	var src Source
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, slug, description, is_active, created_at, updated_at
		FROM inbound_sources WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&src.ID, &src.Name, &src.Slug, &src.Description,
		&src.IsActive, &src.CreatedAt, &src.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSourceNotFound
	}
	return &src, err
}

func (s *Service) UpdateSource(ctx context.Context, id uuid.UUID, req UpdateSourceRequest) (*Source, error) {
	var src Source
	err := s.db.QueryRowContext(ctx, `
		UPDATE inbound_sources
		SET name        = COALESCE($2, name),
		    description = COALESCE($3, description),
		    is_active   = COALESCE($4, is_active),
		    updated_at  = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, name, slug, description, is_active, created_at, updated_at`,
		id, req.Name, req.Description, req.IsActive,
	).Scan(&src.ID, &src.Name, &src.Slug, &src.Description,
		&src.IsActive, &src.CreatedAt, &src.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSourceNotFound
	}
	return &src, err
}

func (s *Service) DeleteSource(ctx context.Context, id uuid.UUID) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE inbound_sources SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrSourceNotFound
	}
	return nil
}

// RegenerateSecret creates a new signing secret for the source.
// The old secret is immediately invalidated.
func (s *Service) RegenerateSecret(ctx context.Context, id uuid.UUID) (string, error) {
	secret, err := generateSecret()
	if err != nil {
		return "", err
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE inbound_sources SET signing_secret = $2, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`, id, secret)
	if err != nil {
		return "", err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return "", ErrSourceNotFound
	}
	return secret, nil
}

// ─── Rule management ─────────────────────────────────────────────────────────

type CreateRuleRequest struct {
	EventTypePattern    string     `json:"event_type_pattern"`
	CasePriority        string     `json:"case_priority"`
	Tags                []string   `json:"tags"`
	TitleTemplate       string     `json:"title_template"`
	DescriptionTemplate string     `json:"description_template"`
	AutoAssignUserID    *uuid.UUID `json:"auto_assign_user_id"`
	SortOrder           int        `json:"sort_order"`
}

type UpdateRuleRequest struct {
	EventTypePattern    *string    `json:"event_type_pattern"`
	CasePriority        *string    `json:"case_priority"`
	Tags                []string   `json:"tags"`
	TitleTemplate       *string    `json:"title_template"`
	DescriptionTemplate *string    `json:"description_template"`
	AutoAssignUserID    *uuid.UUID `json:"auto_assign_user_id"`
	IsActive            *bool      `json:"is_active"`
	SortOrder           *int       `json:"sort_order"`
}

func (s *Service) CreateRule(ctx context.Context, sourceID, creatorID uuid.UUID, req CreateRuleRequest) (*Rule, error) {
	if req.EventTypePattern == "" {
		return nil, fmt.Errorf("event_type_pattern is required")
	}
	if req.TitleTemplate == "" {
		req.TitleTemplate = "{{event_type}} from {{source_name}}"
	}
	if req.DescriptionTemplate == "" {
		req.DescriptionTemplate = "Source: {{source_name}}\nEvent: {{event_type}}\n\n{{payload}}"
	}
	if req.CasePriority == "" {
		req.CasePriority = "medium"
	}
	if req.Tags == nil {
		req.Tags = []string{}
	}

	var rule Rule
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO inbound_rules
		  (source_id, event_type_pattern, case_priority, tags,
		   title_template, description_template, auto_assign_user_id, sort_order, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, source_id, event_type_pattern, case_priority, tags,
		          title_template, description_template, auto_assign_user_id,
		          is_active, sort_order, created_at`,
		sourceID, req.EventTypePattern, req.CasePriority, stringSliceToArray(req.Tags),
		req.TitleTemplate, req.DescriptionTemplate, nullUUID(req.AutoAssignUserID),
		req.SortOrder, creatorID,
	).Scan(&rule.ID, &rule.SourceID, &rule.EventTypePattern, &rule.CasePriority, &rule.Tags,
		&rule.TitleTemplate, &rule.DescriptionTemplate, &rule.AutoAssignUserID,
		&rule.IsActive, &rule.SortOrder, &rule.CreatedAt)
	return &rule, err
}

func (s *Service) ListRules(ctx context.Context, sourceID uuid.UUID) ([]*Rule, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, source_id, event_type_pattern, case_priority, tags,
		       title_template, description_template, auto_assign_user_id,
		       is_active, sort_order, created_at
		FROM inbound_rules
		WHERE source_id = $1
		ORDER BY sort_order ASC, created_at ASC`, sourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Rule
	for rows.Next() {
		var r Rule
		if err := rows.Scan(&r.ID, &r.SourceID, &r.EventTypePattern, &r.CasePriority, &r.Tags,
			&r.TitleTemplate, &r.DescriptionTemplate, &r.AutoAssignUserID,
			&r.IsActive, &r.SortOrder, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &r)
	}
	return out, rows.Err()
}

func (s *Service) UpdateRule(ctx context.Context, ruleID uuid.UUID, req UpdateRuleRequest) (*Rule, error) {
	var tagsExpr interface{} = nil
	if req.Tags != nil {
		tagsExpr = stringSliceToArray(req.Tags)
	}
	var r Rule
	err := s.db.QueryRowContext(ctx, `
		UPDATE inbound_rules
		SET event_type_pattern   = COALESCE($2, event_type_pattern),
		    case_priority         = COALESCE($3, case_priority),
		    tags                  = COALESCE($4, tags),
		    title_template        = COALESCE($5, title_template),
		    description_template  = COALESCE($6, description_template),
		    auto_assign_user_id   = COALESCE($7, auto_assign_user_id),
		    is_active             = COALESCE($8, is_active),
		    sort_order            = COALESCE($9, sort_order),
		    updated_at            = NOW()
		WHERE id = $1
		RETURNING id, source_id, event_type_pattern, case_priority, tags,
		          title_template, description_template, auto_assign_user_id,
		          is_active, sort_order, created_at`,
		ruleID, req.EventTypePattern, req.CasePriority, tagsExpr,
		req.TitleTemplate, req.DescriptionTemplate, nullUUID(req.AutoAssignUserID),
		req.IsActive, req.SortOrder,
	).Scan(&r.ID, &r.SourceID, &r.EventTypePattern, &r.CasePriority, &r.Tags,
		&r.TitleTemplate, &r.DescriptionTemplate, &r.AutoAssignUserID,
		&r.IsActive, &r.SortOrder, &r.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRuleNotFound
	}
	return &r, err
}

func (s *Service) DeleteRule(ctx context.Context, ruleID uuid.UUID) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM inbound_rules WHERE id = $1`, ruleID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrRuleNotFound
	}
	return nil
}

// ─── Event log ───────────────────────────────────────────────────────────────

func (s *Service) ListEvents(ctx context.Context, limit int, sourceID *uuid.UUID) ([]*Event, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query := `
		SELECT e.id, e.source_id, s.name, e.idempotency_key, e.event_type,
		       e.payload, e.status, e.case_id, e.error_message,
		       e.attempt_count, e.processed_at, e.created_at
		FROM inbound_events e
		JOIN inbound_sources s ON s.id = e.source_id
		WHERE ($1::uuid IS NULL OR e.source_id = $1)
		ORDER BY e.created_at DESC
		LIMIT $2`

	rows, err := s.db.QueryContext(ctx, query, nullUUID(sourceID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Event
	for rows.Next() {
		var e Event
		var caseID uuid.NullUUID
		var errMsg sql.NullString
		var processedAt sql.NullTime
		if err := rows.Scan(&e.ID, &e.SourceID, &e.SourceName, &e.IdempotencyKey,
			&e.EventType, &e.Payload, &e.Status, &caseID, &errMsg,
			&e.AttemptCount, &processedAt, &e.CreatedAt); err != nil {
			return nil, err
		}
		if caseID.Valid {
			e.CaseID = &caseID.UUID
		}
		if errMsg.Valid {
			e.ErrorMessage = &errMsg.String
		}
		if processedAt.Valid {
			e.ProcessedAt = &processedAt.Time
		}
		out = append(out, &e)
	}
	return out, rows.Err()
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func (s *Service) getSourceBySlug(ctx context.Context, slug string) (*sourceInternal, error) {
	var src sourceInternal
	err := s.db.QueryRowContext(ctx, `
		SELECT id, slug, name, signing_secret, is_active
		FROM inbound_sources
		WHERE slug = $1 AND deleted_at IS NULL`, slug,
	).Scan(&src.ID, &src.Slug, &src.Name, &src.SigningSecret, &src.IsActive)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSourceNotFound
	}
	return &src, err
}

type sourceInternal struct {
	ID            uuid.UUID
	Slug          string
	Name          string
	SigningSecret string
	IsActive      bool
}

func (s *Service) findMatchingRule(ctx context.Context, sourceID uuid.UUID, eventType string) (*Rule, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, source_id, event_type_pattern, case_priority, tags,
		       title_template, description_template, auto_assign_user_id,
		       is_active, sort_order, created_at
		FROM inbound_rules
		WHERE source_id = $1 AND is_active = true
		ORDER BY sort_order ASC, created_at ASC`, sourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var r Rule
		if err := rows.Scan(&r.ID, &r.SourceID, &r.EventTypePattern, &r.CasePriority, &r.Tags,
			&r.TitleTemplate, &r.DescriptionTemplate, &r.AutoAssignUserID,
			&r.IsActive, &r.SortOrder, &r.CreatedAt); err != nil {
			return nil, err
		}
		if matchPattern(r.EventTypePattern, eventType) {
			return &r, nil
		}
	}
	return nil, ErrNoMatchingRule
}

// matchPattern checks if eventType matches the rule pattern.
// Supports exact match ("fraud_alert") and wildcard suffix ("fraud_*").
func matchPattern(pattern, eventType string) bool {
	if strings.HasSuffix(pattern, "*") {
		prefix := pattern[:len(pattern)-1]
		return strings.HasPrefix(eventType, prefix)
	}
	return pattern == eventType
}

// renderTemplate replaces {{event_type}}, {{source_name}}, {{source}},
// and {{payload.field}} placeholders in a template string.
func renderTemplate(tmpl, eventType, sourceName string, payload map[string]any) string {
	result := tmpl
	result = strings.ReplaceAll(result, "{{event_type}}", eventType)
	result = strings.ReplaceAll(result, "{{source_name}}", sourceName)
	result = strings.ReplaceAll(result, "{{source}}", sourceName)
	for k, v := range payload {
		result = strings.ReplaceAll(result, "{{payload."+k+"}}", fmt.Sprintf("%v", v))
	}
	// Replace {{payload}} with the full JSON blob.
	if payloadJSON, err := json.MarshalIndent(payload, "", "  "); err == nil {
		result = strings.ReplaceAll(result, "{{payload}}", string(payloadJSON))
	}
	return result
}

// resolveCreator returns the user ID + role to attribute system-created cases to.
func (s *Service) resolveCreator(ctx context.Context, sourceID uuid.UUID) (uuid.UUID, db.UserRole) {
	var creatorID uuid.NullUUID
	_ = s.db.QueryRowContext(ctx, `
		SELECT created_by FROM inbound_sources WHERE id = $1`, sourceID,
	).Scan(&creatorID)

	if creatorID.Valid {
		var role db.UserRole
		err := s.db.QueryRowContext(ctx, `SELECT role FROM users WHERE id = $1`, creatorID.UUID).Scan(&role)
		if err == nil {
			return creatorID.UUID, role
		}
	}

	// Fallback: any active super_admin.
	var adminID uuid.UUID
	_ = s.db.QueryRowContext(ctx, `
		SELECT id FROM users WHERE role = 'super_admin' AND is_active = true LIMIT 1`,
	).Scan(&adminID)
	return adminID, db.UserRoleSuperAdmin
}

// validateHMAC verifies the HMAC-SHA256 signature sent by the caller.
// Expected format: "sha256=<hex-encoded-digest>"
// Uses constant-time comparison to prevent timing attacks.
func validateHMAC(secret string, body []byte, signature string) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	return subtle.ConstantTimeCompare([]byte(signature), []byte(expected)) == 1
}

// generateSecret creates a cryptographically random 32-byte hex secret.
func generateSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate secret: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func nullUUID(id *uuid.UUID) uuid.NullUUID {
	if id == nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: *id, Valid: true}
}

// stringSliceToArray converts []string to a PostgreSQL text[] literal.
func stringSliceToArray(s []string) interface{} {
	if s == nil {
		return "{}"
	}
	return fmt.Sprintf("{%s}", strings.Join(quotePGStrings(s), ","))
}

func quotePGStrings(ss []string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}
	return out
}
