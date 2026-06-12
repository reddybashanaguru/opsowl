package ai

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound    = errors.New("ai_log not found")
	ErrForbidden   = errors.New("forbidden")
	ErrBadRequest  = errors.New("bad request")
)

// ── Requests ─────────────────────────────────────────────────────────────────

type AskRequest struct {
	Question string `json:"question"`
}

type SummarizeRequest struct{}

type GenerateNoteRequest struct{}

type ExtractActionsRequest struct {
	SourceType string `json:"source_type"` // "documents" | "all"
}

type AgentRunRequest struct {
	Goal string `json:"goal"`
}

type ApproveRequest struct{}

type RejectRequest struct {
	Reason string `json:"reason"`
}

type EditRequest struct {
	EditedResponse string `json:"edited_response"`
}

type ListLogsRequest struct {
	Status   string
	CursorID *uuid.UUID
	Limit    int32
}

// ── Citation —json both stored in DB and returned to client ──────────────────

type Citation struct {
	DocID      string  `json:"doc_id"`
	DocName    string  `json:"doc_name"`
	ChunkIndex int     `json:"chunk_index"`
	PageNumber *int    `json:"page_number,omitempty"`
	Excerpt    string  `json:"excerpt"`
	Similarity float64 `json:"similarity"`
}

// ── Responses ────────────────────────────────────────────────────────────────

type ActorRef struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
	Email    string    `json:"email"`
}

type ReviewerRef struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
}

type AILogResponse struct {
	ID                  uuid.UUID    `json:"id"`
	CaseID              uuid.UUID    `json:"case_id"`
	ActionType          string       `json:"action_type"`
	ResponseText        string       `json:"response_text"`
	ModelUsed           string       `json:"model_used"`
	Provider            string       `json:"provider"`
	TokensInput         int32        `json:"tokens_input"`
	TokensOutput        int32        `json:"tokens_output"`
	LatencyMs           int32        `json:"latency_ms"`
	ConfidenceScore     *float64     `json:"confidence_score,omitempty"`
	Citations           []Citation   `json:"citations"`
	HumanReviewStatus   string       `json:"human_review_status"`
	TriggeredBy         ActorRef     `json:"triggered_by"`
	ReviewedBy          *ReviewerRef `json:"reviewed_by,omitempty"`
	ReviewedAt          *time.Time   `json:"reviewed_at,omitempty"`
	HumanEditedResponse *string      `json:"human_edited_response,omitempty"`
	InsufficientContext bool         `json:"insufficient_context"`
	CreatedAt           time.Time    `json:"created_at"`
}

type ActionSuggestion struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	Source      string `json:"source_citation"`
}

type ExtractActionsResponse struct {
	LogID       uuid.UUID          `json:"log_id"`
	Suggestions []ActionSuggestion `json:"suggestions"`
}

type ListMeta struct {
	TotalCount int64      `json:"total_count"`
	HasMore    bool       `json:"has_more"`
	NextCursor *uuid.UUID `json:"next_cursor,omitempty"`
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func parseCitations(raw json.RawMessage) []Citation {
	var out []Citation
	_ = json.Unmarshal(raw, &out)
	return out
}

func parseConfidence(s sql.NullString) *float64 {
	if !s.Valid || s.String == "" {
		return nil
	}
	f, _ := strconv.ParseFloat(s.String, 64)
	return &f
}

