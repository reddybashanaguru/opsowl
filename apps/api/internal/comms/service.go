package comms

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/timeline"
)

type Service struct {
	queries      *db.Queries
	aiServiceURL string
	httpClient   *http.Client
	logger       *zap.Logger
	emitter      timeline.Emitter
}

func NewService(queries *db.Queries, aiServiceURL string, logger *zap.Logger, emitter timeline.Emitter) *Service {
	return &Service{
		queries:      queries,
		aiServiceURL: aiServiceURL,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		logger:       logger,
		emitter:      emitter,
	}
}

type CreateParams struct {
	CaseID        uuid.UUID
	LoggedBy      uuid.UUID
	Channel       db.CommChannel
	Direction     db.CommDirection
	ContactName   string
	ContactDetail string
	Subject       string
	Body          string
	Outcome       string
	OccurredAt    time.Time
}

type CommLogResponse struct {
	ID            uuid.UUID   `json:"id"`
	CaseID        uuid.UUID   `json:"case_id"`
	Channel       string      `json:"channel"`
	Direction     string      `json:"direction"`
	ContactName   string      `json:"contact_name"`
	ContactDetail *string     `json:"contact_detail"`
	Subject       *string     `json:"subject"`
	Body          string      `json:"body"`
	Outcome       *string     `json:"outcome"`
	OccurredAt    time.Time   `json:"occurred_at"`
	CreatedAt     time.Time   `json:"created_at"`
	LoggedBy      LoggerRef   `json:"logged_by"`
	AIExtraction  *AIResult   `json:"ai_extraction,omitempty"`
}

type LoggerRef struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
}

type AIResult struct {
	Entities    interface{} `json:"entities"`
	ActionItems interface{} `json:"action_items"`
	Sentiment   *string     `json:"sentiment"`
	ModelUsed   *string     `json:"model_used"`
}

type aiExtractRequest struct {
	Body    string `json:"body"`
	Subject string `json:"subject,omitempty"`
	Channel string `json:"channel"`
}

type aiExtractResponse struct {
	Entities    interface{} `json:"entities"`
	ActionItems interface{} `json:"action_items"`
	Sentiment   string      `json:"sentiment"`
	ModelUsed   string      `json:"model_used"`
}

func (s *Service) Create(ctx context.Context, p CreateParams) (*CommLogResponse, error) {
	row, err := s.queries.CreateCommLog(ctx, db.CreateCommLogParams{
		CaseID:        p.CaseID,
		Channel:       p.Channel,
		Direction:     p.Direction,
		ContactName:   p.ContactName,
		ContactDetail: nullStr(p.ContactDetail),
		Subject:       nullStr(p.Subject),
		Body:          p.Body,
		Outcome:       nullStr(p.Outcome),
		LoggedBy:      p.LoggedBy,
		OccurredAt:    p.OccurredAt,
	})
	if err != nil {
		return nil, err
	}

	resp := rowToResponse(row, p.LoggedBy, "", "")

	// Async AI extraction — don't block the response
	go s.extractAI(row.ID, p.Body, p.Subject, string(p.Channel))

	actorID := p.LoggedBy
	go s.emitter.Emit(context.Background(), timeline.Event{
		CaseID:     p.CaseID,
		EventType:  db.TimelineEventTypeCommLogged,
		Source:     db.TimelineEventSourceUser,
		ActorID:    &actorID,
		OccurredAt: time.Now(),
		Payload: map[string]any{
			"comm_id":   row.ID.String(),
			"channel":   string(p.Channel),
			"direction": string(p.Direction),
			"contact":   p.ContactName,
			"subject":   p.Subject,
		},
	})

	return resp, nil
}

func (s *Service) List(ctx context.Context, caseID uuid.UUID, limit int32) ([]*CommLogResponse, error) {
	rows, err := s.queries.ListCommLogs(ctx, db.ListCommLogsParams{
		CaseID:   caseID,
		RowLimit: limit,
	})
	if err != nil {
		return nil, err
	}

	out := make([]*CommLogResponse, 0, len(rows))
	for _, row := range rows {
		resp := rowToResponse(db.CommunicationLog{
			ID:            row.ID,
			CaseID:        row.CaseID,
			Channel:       row.Channel,
			Direction:     row.Direction,
			ContactName:   row.ContactName,
			ContactDetail: row.ContactDetail,
			Subject:       row.Subject,
			Body:          row.Body,
			Outcome:       row.Outcome,
			LoggedBy:      row.LoggedBy,
			OccurredAt:    row.OccurredAt,
			CreatedAt:     row.CreatedAt,
		}, row.LoggedBy, row.LoggedByName, row.LoggedByEmail)

		if row.AiEntities.Valid || row.AiActions.Valid {
			ai := &AIResult{}
			if row.AiEntities.Valid {
				var ent interface{}
				_ = json.Unmarshal(row.AiEntities.RawMessage, &ent)
				ai.Entities = ent
			}
			if row.AiActions.Valid {
				var acts interface{}
				_ = json.Unmarshal(row.AiActions.RawMessage, &acts)
				ai.ActionItems = acts
			}
			resp.AIExtraction = ai
		}
		out = append(out, resp)
	}
	return out, nil
}

func (s *Service) Delete(ctx context.Context, commID uuid.UUID, loggedBy uuid.UUID) error {
	return s.queries.DeleteCommLog(ctx, db.DeleteCommLogParams{
		ID:       commID,
		LoggedBy: loggedBy,
	})
}

func (s *Service) extractAI(commID uuid.UUID, body, subject, channel string) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	payload, _ := json.Marshal(aiExtractRequest{Body: body, Subject: subject, Channel: channel})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.aiServiceURL+"/extract/comms-entities", bytes.NewReader(payload))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Warn("comms AI extraction failed", zap.Error(err))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		s.logger.Warn("comms AI extraction non-200", zap.Int("status", resp.StatusCode))
		return
	}

	var out aiExtractResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return
	}

	entJSON, _ := json.Marshal(out.Entities)
	actJSON, _ := json.Marshal(out.ActionItems)

	if _, err := s.queries.UpsertCommAIExtraction(context.Background(), db.UpsertCommAIExtractionParams{
		CommLogID:   commID,
		Entities:    entJSON,
		ActionItems: actJSON,
		Sentiment:   sql.NullString{String: out.Sentiment, Valid: out.Sentiment != ""},
		ModelUsed:   sql.NullString{String: out.ModelUsed, Valid: out.ModelUsed != ""},
	}); err != nil {
		s.logger.Warn("failed to store comm AI extraction", zap.Error(err))
	}
}

func rowToResponse(row db.CommunicationLog, loggedByID uuid.UUID, name, email string) *CommLogResponse {
	r := &CommLogResponse{
		ID:          row.ID,
		CaseID:      row.CaseID,
		Channel:     string(row.Channel),
		Direction:   string(row.Direction),
		ContactName: row.ContactName,
		Body:        row.Body,
		OccurredAt:  row.OccurredAt,
		CreatedAt:   row.CreatedAt,
		LoggedBy:    LoggerRef{ID: loggedByID, Name: name, Email: email},
	}
	if row.ContactDetail.Valid {
		r.ContactDetail = &row.ContactDetail.String
	}
	if row.Subject.Valid {
		r.Subject = &row.Subject.String
	}
	if row.Outcome.Valid {
		r.Outcome = &row.Outcome.String
	}
	return r
}

func nullStr(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

