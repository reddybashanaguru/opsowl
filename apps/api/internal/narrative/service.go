package narrative

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/timeline"
	"github.com/reddybashanaguru/opsowl/internal/worker"
)

const (
	evtNarrativeGenerated = db.TimelineEventType("narrative_generated")
	evtNarrativeApproved  = db.TimelineEventType("narrative_approved")
	evtNarrativeRejected  = db.TimelineEventType("narrative_rejected")
	evtNarrativeEdited    = db.TimelineEventType("narrative_edited")
)

type Service struct {
	repo          *Repository
	queries       *db.Queries
	emitter       timeline.Emitter
	aiServiceURL  string
	serviceSecret string
	httpClient    *http.Client
}

func NewService(repo *Repository, queries *db.Queries, emitter timeline.Emitter, aiServiceURL string) *Service {
	return &Service{
		repo:         repo,
		queries:      queries,
		emitter:      emitter,
		aiServiceURL: aiServiceURL,
		httpClient:   &http.Client{Timeout: 120 * time.Second},
	}
}

func (s *Service) WithServiceSecret(secret string) *Service {
	s.serviceSecret = secret
	return s
}

// ── Generate ──────────────────────────────────────────────────────────────────

// Generate calls the AI service to produce a narrative, persists it, and emits
// a timeline event. The heavy AI call runs synchronously (caller controls timeout).
func (s *Service) Generate(ctx context.Context, claims *auth.Claims, caseID uuid.UUID, req GenerateRequest) (NarrativeResponse, error) {
	ntype := req.NarrativeType
	if ntype == "" {
		ntype = TypeInvestigationSummary
	}
	if !validTypes[ntype] {
		return NarrativeResponse{}, fmt.Errorf("%w: invalid narrative_type", ErrBadRequest)
	}

	// Gather context
	caseCtx, err := s.repo.GetCaseContext(ctx, caseID)
	if err != nil {
		return NarrativeResponse{}, ErrNotFound
	}
	findings, _ := s.repo.GetApprovedFindings(ctx, caseID)
	txns, _ := s.repo.GetCaseTransactions(ctx, caseID)
	subjects, _ := s.repo.GetSubjectNames(ctx, caseID)

	if subjects == nil {
		subjects = []string{}
	}

	// Build AI request — use empty slices (not nil) so JSON marshals to [] not null
	aiReq := aiNarrativeRequest{
		CaseID:            caseID.String(),
		CaseNumber:        caseCtx.CaseNumber,
		CaseTitle:         caseCtx.Title,
		CaseDescription:   caseCtx.Description,
		RiskLevel:         caseCtx.RiskLevel,
		NarrativeType:     ntype,
		SubjectNames:      subjects,
		InvestigatorNotes: req.InvestigatorNotes,
		Findings:          []aiNarrativeFinding{},
		Transactions:      []aiNarrativeTransaction{},
	}
	for _, f := range findings {
		aiReq.Findings = append(aiReq.Findings, aiNarrativeFinding{
			Title:           f.Title,
			Category:        f.Category,
			Severity:        f.Severity,
			Description:     f.Description,
			NarrativeWeight: f.NarrativeWeight,
		})
	}
	for _, t := range txns {
		aiReq.Transactions = append(aiReq.Transactions, aiNarrativeTransaction{
			TxnRef:     t.TxnRef,
			TxnDate:    t.TxnDate.Format("2006-01-02"),
			Amount:     t.Amount,
			Currency:   t.Currency,
			TxnType:    t.TxnType,
			Channel:    t.Channel,
			FromName:   t.FromName,
			ToName:     t.ToName,
			Narration:  t.Narration,
			IsFlagged:  t.IsFlagged,
			FlagReason: t.FlagReason,
		})
	}

	start := time.Now()
	aiResp, aiErr := s.callAIService(ctx, aiReq)
	latencyMs := int32(time.Since(start).Milliseconds())

	// Persist AI log first (regardless of success)
	promptJSON, _ := json.Marshal(aiReq)
	var aiLogID *uuid.UUID
	var modelUsed, promptVersion string
	if aiErr == nil {
		respJSON, _ := json.Marshal(aiResp)
		logRow, logErr := s.queries.CreateAILog(ctx, db.CreateAILogParams{
			CaseID:          caseID,
			TriggeredBy:     claims.UserID,
			ActionType:      "generate_narrative",
			PromptText:      string(promptJSON),
			ResponseText:    string(respJSON),
			ModelUsed:       aiResp.ModelUsed,
			Provider:        "internal_ai_service",
			LatencyMs:       latencyMs,
			ConfidenceScore: sql.NullString{},
			Citations:       json.RawMessage(`[]`),
		})
		if logErr == nil {
			id := logRow.ID
			aiLogID = &id
		}
		modelUsed = aiResp.ModelUsed
		promptVersion = aiResp.PromptVersion
		if promptVersion == "" {
			promptVersion = "v1"
		}
	} else {
		// Log the failed attempt
		s.queries.CreateAILog(ctx, db.CreateAILogParams{ //nolint:errcheck
			CaseID:          caseID,
			TriggeredBy:     claims.UserID,
			ActionType:      "generate_narrative",
			PromptText:      string(promptJSON),
			ResponseText:    fmt.Sprintf("error: %v", aiErr),
			ModelUsed:       "unknown",
			Provider:        "internal_ai_service",
			LatencyMs:       latencyMs,
			ConfidenceScore: sql.NullString{},
			Citations:       json.RawMessage(`[]`),
		})
		return NarrativeResponse{}, fmt.Errorf("AI service unavailable: %w", aiErr)
	}

	body := aiResp.Narrative
	row, err := s.repo.Create(ctx, caseID, claims.UserID, aiLogID,
		body, ntype, modelUsed, promptVersion,
		len(findings), len(txns))
	if err != nil {
		return NarrativeResponse{}, err
	}

	actorID := claims.UserID
	worker.TrySubmit(func() {
		s.emitter.Emit(context.Background(), timeline.Event{ //nolint:errcheck
			CaseID:     caseID,
			EventType:  evtNarrativeGenerated,
			Source:     db.TimelineEventSourceAi,
			ActorID:    &actorID,
			OccurredAt: time.Now(),
			Payload: map[string]any{
				"narrative_id":   row.ID.String(),
				"narrative_type": ntype,
				"model_used":     modelUsed,
				"findings_count": len(findings),
				"txns_count":     len(txns),
			},
		})
	})

	return toResponse(row), nil
}

// ── List / Get ────────────────────────────────────────────────────────────────

func (s *Service) List(ctx context.Context, caseID uuid.UUID) ([]NarrativeResponse, error) {
	rows, err := s.repo.ListForCase(ctx, caseID)
	if err != nil {
		return nil, err
	}
	result := make([]NarrativeResponse, len(rows))
	for i, r := range rows {
		result[i] = toResponse(r)
	}
	return result, nil
}

func (s *Service) Get(ctx context.Context, caseID, id uuid.UUID) (NarrativeResponse, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return NarrativeResponse{}, ErrNotFound
		}
		return NarrativeResponse{}, err
	}
	if row.CaseID != caseID {
		return NarrativeResponse{}, ErrNotFound
	}
	return toResponse(row), nil
}

// ── Edit ──────────────────────────────────────────────────────────────────────

func (s *Service) Edit(ctx context.Context, claims *auth.Claims, caseID, id uuid.UUID, req EditRequest) (NarrativeResponse, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil || row.CaseID != caseID {
		return NarrativeResponse{}, ErrNotFound
	}
	if row.Status == "approved" {
		return NarrativeResponse{}, ErrAlreadyApproved
	}
	if err := s.repo.UpdateBody(ctx, id, req.Body); err != nil {
		return NarrativeResponse{}, err
	}

	actorID := claims.UserID
	worker.TrySubmit(func() {
		s.emitter.Emit(context.Background(), timeline.Event{ //nolint:errcheck
			CaseID:     caseID,
			EventType:  evtNarrativeEdited,
			Source:     db.TimelineEventSourceUser,
			ActorID:    &actorID,
			OccurredAt: time.Now(),
			Payload:    map[string]any{"narrative_id": id.String()},
		})
	})

	updated, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return NarrativeResponse{}, err
	}
	return toResponse(updated), nil
}

// ── Approve / Reject ──────────────────────────────────────────────────────────

func (s *Service) Approve(ctx context.Context, claims *auth.Claims, caseID, id uuid.UUID) (NarrativeResponse, error) {
	if claims.Role != db.UserRoleManager && claims.Role != db.UserRoleAdmin && claims.Role != db.UserRoleSuperAdmin {
		return NarrativeResponse{}, ErrForbidden
	}
	row, err := s.repo.GetByID(ctx, id)
	if err != nil || row.CaseID != caseID {
		return NarrativeResponse{}, ErrNotFound
	}
	if row.Status == "approved" {
		return NarrativeResponse{}, ErrAlreadyApproved
	}
	if err := s.repo.UpdateStatus(ctx, id, claims.UserID, "approved", ""); err != nil {
		return NarrativeResponse{}, err
	}

	actorID := claims.UserID
	worker.TrySubmit(func() {
		s.emitter.Emit(context.Background(), timeline.Event{ //nolint:errcheck
			CaseID:     caseID,
			EventType:  evtNarrativeApproved,
			Source:     db.TimelineEventSourceUser,
			ActorID:    &actorID,
			OccurredAt: time.Now(),
			Payload:    map[string]any{"narrative_id": id.String()},
		})
	})

	updated, _ := s.repo.GetByID(ctx, id)
	return toResponse(updated), nil
}

func (s *Service) Reject(ctx context.Context, claims *auth.Claims, caseID, id uuid.UUID, reason string) (NarrativeResponse, error) {
	if claims.Role != db.UserRoleManager && claims.Role != db.UserRoleAdmin && claims.Role != db.UserRoleSuperAdmin {
		return NarrativeResponse{}, ErrForbidden
	}
	row, err := s.repo.GetByID(ctx, id)
	if err != nil || row.CaseID != caseID {
		return NarrativeResponse{}, ErrNotFound
	}
	if row.Status != "draft" {
		return NarrativeResponse{}, ErrInvalidTransition
	}
	if err := s.repo.UpdateStatus(ctx, id, claims.UserID, "rejected", reason); err != nil {
		return NarrativeResponse{}, err
	}

	actorID := claims.UserID
	worker.TrySubmit(func() {
		s.emitter.Emit(context.Background(), timeline.Event{ //nolint:errcheck
			CaseID:     caseID,
			EventType:  evtNarrativeRejected,
			Source:     db.TimelineEventSourceUser,
			ActorID:    &actorID,
			OccurredAt: time.Now(),
			Payload:    map[string]any{"narrative_id": id.String(), "reason": reason},
		})
	})

	updated, _ := s.repo.GetByID(ctx, id)
	return toResponse(updated), nil
}

func (s *Service) Delete(ctx context.Context, claims *auth.Claims, caseID, id uuid.UUID) error {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil || row.CaseID != caseID {
		return ErrNotFound
	}
	if row.Status == "approved" {
		return ErrAlreadyApproved
	}
	return s.repo.Delete(ctx, id)
}

// ── AI service call ───────────────────────────────────────────────────────────

func (s *Service) callAIService(ctx context.Context, req aiNarrativeRequest) (*aiNarrativeResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.aiServiceURL+"/narrative/generate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if s.serviceSecret != "" {
		httpReq.Header.Set("Authorization", "Bearer "+s.serviceSecret)
	}

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ai service %d", resp.StatusCode)
	}
	var out aiNarrativeResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ── Response mapper ───────────────────────────────────────────────────────────

func toResponse(row narrativeRow) NarrativeResponse {
	resp := NarrativeResponse{
		ID:               row.ID.String(),
		CaseID:           row.CaseID.String(),
		NarrativeType:    row.NarrativeType,
		Status:           row.Status,
		Body:             row.Body,
		RejectionReason:  row.RejectionReason,
		ModelUsed:        row.ModelUsed,
		PromptVersion:    row.PromptVersion,
		FindingsIncluded: row.FindingsIncluded,
		TxnsIncluded:     row.TxnsIncluded,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}

	// Active body: prefer human edit if set
	if row.BodyEdited.Valid && row.BodyEdited.String != "" {
		resp.BodyEdited = &row.BodyEdited.String
		resp.ActiveBody = row.BodyEdited.String
	} else {
		resp.ActiveBody = row.Body
	}

	if row.AILogID.Valid {
		s := row.AILogID.UUID.String()
		resp.AILogID = &s
	}
	if row.GeneratedBy.Valid {
		name := ""
		if row.GeneratedByName.Valid {
			name = row.GeneratedByName.String
		}
		resp.GeneratedBy = &UserRef{ID: row.GeneratedBy.UUID, FullName: name}
	}
	if row.ReviewedBy.Valid {
		name := ""
		if row.ReviewedByName.Valid {
			name = row.ReviewedByName.String
		}
		resp.ReviewedBy = &UserRef{ID: row.ReviewedBy.UUID, FullName: name}
	}
	if row.ReviewedAt.Valid {
		resp.ReviewedAt = &row.ReviewedAt.Time
	}
	return resp
}
