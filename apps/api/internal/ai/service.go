package ai

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/reddybashanaguru/opsowl/internal/auth"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/timeline"
)

const insufficientContextSentinel = "INSUFFICIENT_CONTEXT"

// pyAskRequest is what we POST to the Python copilot service.
type pyAskRequest struct {
	CaseID   string `json:"case_id"`
	Question string `json:"question"`
	TopK     int    `json:"top_k"`
}

type pyCopilotRequest struct {
	CaseID     string `json:"case_id"`
	SourceType string `json:"source_type,omitempty"`
}

// pyCopilotResponse is the common envelope returned by all Python copilot endpoints.
type pyCopilotResponse struct {
	Answer              string     `json:"answer"`
	ConfidenceScore     *float64   `json:"confidence_score"`
	Citations           []Citation `json:"citations"`
	ModelUsed           string     `json:"model_used"`
	Provider            string     `json:"provider"`
	TokensInput         int32      `json:"tokens_input"`
	TokensOutput        int32      `json:"tokens_output"`
	LatencyMs           int32      `json:"latency_ms"`
	InsufficientContext bool       `json:"insufficient_context"`
	PromptText          string     `json:"prompt_text"`
}

type Service struct {
	repo           *Repository
	emitter        timeline.Emitter
	logger         *zap.Logger
	httpClient     *http.Client
	aiServiceURL   string
	serviceSecret  string         // shared secret for Go→Python auth (AI_SERVICE_SECRET)
	cb             *circuitBreaker
}

func NewService(repo *Repository, emitter timeline.Emitter, logger *zap.Logger, aiServiceURL string) *Service {
	return &Service{
		repo:         repo,
		emitter:      emitter,
		logger:       logger,
		httpClient:   &http.Client{Timeout: 120 * time.Second},
		aiServiceURL: aiServiceURL,
		// Open after 5 consecutive failures; try one probe after 30s.
		cb: newCircuitBreaker(5, 30*time.Second),
	}
}

// WithServiceSecret sets the shared secret sent to the Python AI service.
func (s *Service) WithServiceSecret(secret string) *Service {
	s.serviceSecret = secret
	return s
}

// ── Core AI actions ───────────────────────────────────────────────────────────

func (s *Service) Ask(ctx context.Context, claims *auth.Claims, caseID uuid.UUID, question string) (*AILogResponse, error) {
	if len(question) == 0 || len(question) > 500 {
		return nil, fmt.Errorf("%w: question must be 1–500 characters", ErrBadRequest)
	}

	req := pyAskRequest{CaseID: caseID.String(), Question: question, TopK: 8}
	resp, err := s.callPython(ctx, "/copilot/ask", req)
	if err != nil {
		return nil, fmt.Errorf("AI service error: %w", err)
	}

	return s.storeLog(ctx, claims, caseID, "ask", resp, db.TimelineEventTypeAiQuery)
}

func (s *Service) Summarize(ctx context.Context, claims *auth.Claims, caseID uuid.UUID) (*AILogResponse, error) {
	req := pyCopilotRequest{CaseID: caseID.String()}
	resp, err := s.callPython(ctx, "/copilot/summarize", req)
	if err != nil {
		return nil, fmt.Errorf("AI service error: %w", err)
	}

	return s.storeLog(ctx, claims, caseID, "summarize", resp, db.TimelineEventTypeAiSummaryGenerated)
}

func (s *Service) GenerateNote(ctx context.Context, claims *auth.Claims, caseID uuid.UUID) (*AILogResponse, error) {
	req := pyCopilotRequest{CaseID: caseID.String()}
	resp, err := s.callPython(ctx, "/copilot/generate-note", req)
	if err != nil {
		return nil, fmt.Errorf("AI service error: %w", err)
	}

	return s.storeLog(ctx, claims, caseID, "generate_note", resp, db.TimelineEventTypeAiQuery)
}

func (s *Service) ExtractActions(ctx context.Context, claims *auth.Claims, caseID uuid.UUID, sourceType string) (*AILogResponse, error) {
	if sourceType == "" {
		sourceType = "documents"
	}
	req := pyCopilotRequest{CaseID: caseID.String(), SourceType: sourceType}
	resp, err := s.callPython(ctx, "/copilot/extract-actions", req)
	if err != nil {
		return nil, fmt.Errorf("AI service error: %w", err)
	}

	return s.storeLog(ctx, claims, caseID, "extract_actions", resp, db.TimelineEventTypeAiActionSuggested)
}

// StreamAgent proxies the autonomous agent SSE stream from the Python service to the HTTP client.
// The response is never buffered — bytes are flushed as they arrive.
func (s *Service) StreamAgent(ctx context.Context, claims *auth.Claims, caseID uuid.UUID, goal string, w http.ResponseWriter) error {
	if len(goal) == 0 || len(goal) > 1000 {
		return fmt.Errorf("%w: goal must be 1–1000 characters", ErrBadRequest)
	}

	type pyAgentReq struct {
		Goal string `json:"goal"`
	}
	body, _ := json.Marshal(pyAgentReq{Goal: goal})

	// No timeout — agent may run for several minutes
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/agent/%s/run", s.aiServiceURL, caseID.String()),
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.serviceSecret != "" {
		req.Header.Set("Authorization", "Bearer "+s.serviceSecret)
	}

	// Use a no-timeout client for streaming
	streamClient := &http.Client{}
	resp, err := streamClient.Do(req)
	if err != nil {
		return fmt.Errorf("agent stream request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		var errBody struct{ Detail string `json:"detail"` }
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return fmt.Errorf("agent service returned %d: %s", resp.StatusCode, errBody.Detail)
	}

	// Set SSE headers before writing anything
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, canFlush := w.(http.Flusher)
	buf := make([]byte, 4096)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				return nil // client disconnected
			}
			if canFlush {
				flusher.Flush()
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}

	s.emitTimeline(ctx, caseID, claims.UserID, db.TimelineEventTypeAiQuery, map[string]interface{}{
		"action_type": "agent_run",
		"goal":        goal,
	})
	return nil
}

// ── Human review actions ──────────────────────────────────────────────────────

func (s *Service) Approve(ctx context.Context, claims *auth.Claims, logID uuid.UUID) (*AILogResponse, error) {
	_, err := s.repo.Approve(ctx, logID, claims.UserID)
	if err != nil {
		return nil, err
	}
	return s.getLogResponse(ctx, logID)
}

func (s *Service) Reject(ctx context.Context, claims *auth.Claims, logID uuid.UUID, reason string) (*AILogResponse, error) {
	if reason == "" {
		return nil, fmt.Errorf("%w: rejection reason is required", ErrBadRequest)
	}
	_, err := s.repo.Reject(ctx, logID, claims.UserID, reason)
	if err != nil {
		return nil, err
	}
	return s.getLogResponse(ctx, logID)
}

func (s *Service) Edit(ctx context.Context, claims *auth.Claims, logID uuid.UUID, editedResponse string) (*AILogResponse, error) {
	if editedResponse == "" {
		return nil, fmt.Errorf("%w: edited response cannot be empty", ErrBadRequest)
	}
	_, err := s.repo.Edit(ctx, logID, claims.UserID, editedResponse)
	if err != nil {
		return nil, err
	}
	return s.getLogResponse(ctx, logID)
}

func (s *Service) GetLog(ctx context.Context, logID uuid.UUID) (*AILogResponse, error) {
	return s.getLogResponse(ctx, logID)
}

func (s *Service) ListByCase(ctx context.Context, caseID uuid.UUID, f ListLogsRequest) ([]*AILogResponse, ListMeta, error) {
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	cursorParam := uuid.Nil // sentinel: no cursor
	if f.CursorID != nil {
		cursorParam = *f.CursorID
	}

	rows, err := s.repo.ListByCase(ctx, db.ListAILogsByCaseParams{
		CaseID:  caseID,
		Column2: f.Status,
		Column3: cursorParam,
		Limit:   limit + 1,
	})
	if err != nil {
		return nil, ListMeta{}, err
	}

	hasMore := len(rows) > int(limit)
	if hasMore {
		rows = rows[:limit]
	}

	total, _ := s.repo.CountByCase(ctx, caseID, f.Status)

	var nextCursor *uuid.UUID
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1].ID
		nextCursor = &last
	}

	out := make([]*AILogResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, buildListResponse(row))
	}

	return out, ListMeta{TotalCount: total, HasMore: hasMore, NextCursor: nextCursor}, nil
}

func (s *Service) ListPending(ctx context.Context, limit int32) ([]*AILogResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.repo.ListPending(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]*AILogResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, buildPendingResponse(row))
	}
	return out, nil
}

// ── Private helpers ───────────────────────────────────────────────────────────

func (s *Service) callPython(ctx context.Context, path string, body interface{}) (*pyCopilotResponse, error) {
	if !s.cb.allow() {
		return nil, ErrCircuitOpen
	}

	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.aiServiceURL+path, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.serviceSecret != "" {
		req.Header.Set("Authorization", "Bearer "+s.serviceSecret)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.cb.recordFailure()
		s.logger.Warn("AI service call failed", zap.String("path", path), zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		s.cb.recordFailure()
		var errBody struct{ Detail string `json:"detail"` }
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return nil, fmt.Errorf("python service %s returned %d: %s", path, resp.StatusCode, errBody.Detail)
	}

	var out pyCopilotResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		s.cb.recordFailure()
		return nil, fmt.Errorf("decode python response: %w", err)
	}
	s.cb.recordSuccess()
	return &out, nil
}

func (s *Service) storeLog(ctx context.Context, claims *auth.Claims, caseID uuid.UUID, actionType string, py *pyCopilotResponse, evtType db.TimelineEventType) (*AILogResponse, error) {
	citationsJSON, _ := json.Marshal(py.Citations)

	var confScore sql.NullString
	if py.ConfidenceScore != nil {
		confScore = sql.NullString{String: fmt.Sprintf("%.2f", *py.ConfidenceScore), Valid: true}
	}

	row, err := s.repo.Create(ctx, db.CreateAILogParams{
		CaseID:          caseID,
		TriggeredBy:     claims.UserID,
		ActionType:      actionType,
		PromptText:      py.PromptText,
		ResponseText:    py.Answer,
		ModelUsed:       py.ModelUsed,
		Provider:        py.Provider,
		TokensInput:     py.TokensInput,
		TokensOutput:    py.TokensOutput,
		LatencyMs:       py.LatencyMs,
		ConfidenceScore: confScore,
		Citations:       citationsJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store AI log: %w", err)
	}

	s.emitTimeline(ctx, caseID, claims.UserID, evtType, map[string]interface{}{
		"log_id":      row.ID.String(),
		"action_type": actionType,
		"model_used":  py.ModelUsed,
	})

	resp := buildFullResponse(row)
	resp.InsufficientContext = py.InsufficientContext
	return resp, nil
}

func (s *Service) getLogResponse(ctx context.Context, logID uuid.UUID) (*AILogResponse, error) {
	row, err := s.repo.GetByID(ctx, logID)
	if err != nil {
		return nil, err
	}
	return buildFullResponse(row), nil
}

func (s *Service) emitTimeline(ctx context.Context, caseID, actorID uuid.UUID, evtType db.TimelineEventType, payload map[string]interface{}) {
	if err := s.emitter.Emit(ctx, timeline.Event{
		CaseID:     caseID,
		ActorID:    &actorID,
		EventType:  evtType,
		Source:     db.TimelineEventSourceAi,
		OccurredAt: time.Now().UTC(),
		Payload:    payload,
	}); err != nil {
		s.logger.Warn("failed to emit AI timeline event", zap.Error(err))
	}
}

// ── Response builders ─────────────────────────────────────────────────────────

func buildFullResponse(row db.GetAILogRow) *AILogResponse {
	r := &AILogResponse{
		ID:                uuid.UUID(row.ID),
		CaseID:            uuid.UUID(row.CaseID),
		ActionType:        row.ActionType,
		ResponseText:      row.ResponseText,
		ModelUsed:         row.ModelUsed,
		Provider:          row.Provider,
		TokensInput:       row.TokensInput,
		TokensOutput:      row.TokensOutput,
		LatencyMs:         row.LatencyMs,
		ConfidenceScore:   parseConfidence(row.ConfidenceScore),
		Citations:         parseCitations(row.Citations),
		HumanReviewStatus: row.HumanReviewStatus,
		TriggeredBy: ActorRef{
			ID:       row.TriggeredBy,
			FullName: row.TriggeredByName,
			Email:    row.TriggeredByEmail,
		},
		CreatedAt: row.CreatedAt,
	}
	if row.ReviewedBy.Valid {
		rr := &ReviewerRef{ID: row.ReviewedBy.UUID}
		if row.ReviewedByName.Valid {
			rr.FullName = row.ReviewedByName.String
		}
		r.ReviewedBy = rr
	}
	if row.ReviewedAt.Valid {
		t := row.ReviewedAt.Time
		r.ReviewedAt = &t
	}
	if row.HumanEditedResponse.Valid {
		s := row.HumanEditedResponse.String
		r.HumanEditedResponse = &s
	}
	return r
}

func buildListResponse(row db.ListAILogsByCaseRow) *AILogResponse {
	r := &AILogResponse{
		ID:                row.ID,
		CaseID:            row.CaseID,
		ActionType:        row.ActionType,
		ResponseText:      row.ResponseText,
		ModelUsed:         row.ModelUsed,
		Provider:          row.Provider,
		TokensInput:       row.TokensInput,
		TokensOutput:      row.TokensOutput,
		LatencyMs:         row.LatencyMs,
		ConfidenceScore:   parseConfidence(row.ConfidenceScore),
		Citations:         parseCitations(row.Citations),
		HumanReviewStatus: row.HumanReviewStatus,
		TriggeredBy: ActorRef{
			ID:       row.TriggeredBy,
			FullName: row.TriggeredByName,
			Email:    row.TriggeredByEmail,
		},
		CreatedAt: row.CreatedAt,
	}
	if row.ReviewedBy.Valid {
		rr := &ReviewerRef{ID: row.ReviewedBy.UUID}
		if row.ReviewedByName.Valid {
			rr.FullName = row.ReviewedByName.String
		}
		r.ReviewedBy = rr
	}
	if row.ReviewedAt.Valid {
		t := row.ReviewedAt.Time
		r.ReviewedAt = &t
	}
	if row.HumanEditedResponse.Valid {
		s := row.HumanEditedResponse.String
		r.HumanEditedResponse = &s
	}
	return r
}

func buildPendingResponse(row db.ListPendingAILogsRow) *AILogResponse {
	return &AILogResponse{
		ID:                row.ID,
		CaseID:            row.CaseID,
		ActionType:        row.ActionType,
		ResponseText:      row.ResponseText,
		ModelUsed:         row.ModelUsed,
		Provider:          row.Provider,
		TokensInput:       row.TokensInput,
		TokensOutput:      row.TokensOutput,
		LatencyMs:         row.LatencyMs,
		ConfidenceScore:   parseConfidence(row.ConfidenceScore),
		Citations:         parseCitations(row.Citations),
		HumanReviewStatus: row.HumanReviewStatus,
		TriggeredBy: ActorRef{
			ID:       row.TriggeredBy,
			FullName: row.TriggeredByName,
			Email:    row.TriggeredByEmail,
		},
		CreatedAt: row.CreatedAt,
	}
}
