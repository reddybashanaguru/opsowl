package findings

import (
	"time"

	"github.com/google/uuid"
)

// ── Request types ─────────────────────────────────────────────────────────────

type CreateFindingRequest struct {
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	Category        string  `json:"category"`
	Severity        string  `json:"severity"`
	NarrativeWeight string  `json:"narrative_weight"`
	AIGenerated     bool    `json:"ai_generated"`
	SourceAILogID   *string `json:"source_ai_log_id"`
}

type UpdateFindingRequest struct {
	Title           *string `json:"title"`
	Description     *string `json:"description"`
	Category        *string `json:"category"`
	Severity        *string `json:"severity"`
	NarrativeWeight *string `json:"narrative_weight"`
}

type ReviewFindingRequest struct {
	// Used for reject and severity override actions
	Reason string `json:"reason"`
}

type SeverityOverrideRequest struct {
	NewSeverity string `json:"new_severity"`
	Reason      string `json:"reason"`
}

type LinkEvidenceRequest struct {
	EvidenceType  string  `json:"evidence_type"` // document | note | transaction | ai_log | subject | alert
	EvidenceID    string  `json:"evidence_id"`
	Citation      string  `json:"citation"`
	PageNumber    *int    `json:"page_number"`
	ChunkID       *string `json:"chunk_id"`
	RelevanceNote string  `json:"relevance_note"`
}

// ── Response types ────────────────────────────────────────────────────────────

type UserRef struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
}

type EvidenceLinkResponse struct {
	ID            string    `json:"id"`
	FindingID     string    `json:"finding_id"`
	EvidenceType  string    `json:"evidence_type"`
	EvidenceID    string    `json:"evidence_id"`
	Citation      string    `json:"citation"`
	PageNumber    *int32    `json:"page_number,omitempty"`
	ChunkID       *string   `json:"chunk_id,omitempty"`
	RelevanceNote string    `json:"relevance_note"`
	LinkedBy      string    `json:"linked_by"`
	LinkedAt      time.Time `json:"linked_at"`
}

type FindingResponse struct {
	ID              string                 `json:"id"`
	CaseID          string                 `json:"case_id"`
	Title           string                 `json:"title"`
	Description     string                 `json:"description"`
	Category        string                 `json:"category"`
	Severity        string                 `json:"severity"`
	Status          string                 `json:"status"`
	NarrativeWeight string                 `json:"narrative_weight"`
	ConfidenceScore *float64               `json:"confidence_score,omitempty"`
	AIGenerated     bool                   `json:"ai_generated"`
	SourceAILogID   *string                `json:"source_ai_log_id,omitempty"`
	EvidenceCount   int64                  `json:"evidence_count"`
	CreatedBy       UserRef                `json:"created_by"`
	ReviewedBy      *UserRef               `json:"reviewed_by,omitempty"`
	ApprovedBy      *UserRef               `json:"approved_by,omitempty"`
	ReviewedAt      *time.Time             `json:"reviewed_at,omitempty"`
	ApprovedAt      *time.Time             `json:"approved_at,omitempty"`
	EvidenceLinks   []EvidenceLinkResponse `json:"evidence_links,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

type ListMeta struct {
	TotalCount int64   `json:"total_count"`
	HasMore    bool    `json:"has_more"`
	NextCursor *string `json:"next_cursor"`
}

// FindingsSummary is embedded in case detail to show finding counts without
// loading the full list.
type FindingsSummary struct {
	Total    int `json:"total"`
	Approved int `json:"approved"`
	Critical int `json:"critical"`
	High     int `json:"high"`
}
