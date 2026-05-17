package search

import (
	"time"

	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

type SearchRequest struct {
	Query  string `json:"query"`
	Limit  int32  `json:"limit"`
	Filter string `json:"filter"` // "cases" | "notes" | "tasks" | "documents" | "" (all)
}

type CaseResult struct {
	ID            uuid.UUID    `json:"id"`
	CaseNumber    string       `json:"case_number"`
	Title         string       `json:"title"`
	Description   string       `json:"description"`
	Status        string       `json:"status"`
	Priority      string       `json:"priority"`
	SLABreached   bool         `json:"sla_breached"`
	CreatedByName string       `json:"created_by_name"`
	CreatedAt     time.Time    `json:"created_at"`
	Rank          float32      `json:"rank"`
	EntityType    string       `json:"entity_type"`
}

type NoteResult struct {
	ID          uuid.UUID `json:"id"`
	CaseID      uuid.UUID `json:"case_id"`
	CaseNumber  string    `json:"case_number"`
	CaseTitle   string    `json:"case_title"`
	Title       string    `json:"title"`
	ContentText string    `json:"content_text"`
	NoteType    string    `json:"note_type"`
	IsPinned    bool      `json:"is_pinned"`
	AIGenerated bool      `json:"ai_generated"`
	CreatedAt   time.Time `json:"created_at"`
	Rank        float32   `json:"rank"`
	EntityType  string    `json:"entity_type"`
}

type TaskResult struct {
	ID          uuid.UUID `json:"id"`
	CaseID      uuid.UUID `json:"case_id"`
	CaseNumber  string    `json:"case_number"`
	CaseTitle   string    `json:"case_title"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Priority    string    `json:"priority"`
	IsOverdue   bool      `json:"is_overdue"`
	CreatedAt   time.Time `json:"created_at"`
	Rank        float32   `json:"rank"`
	EntityType  string    `json:"entity_type"`
}

type DocumentResult struct {
	ID               uuid.UUID `json:"id"`
	CaseID           uuid.UUID `json:"case_id"`
	CaseNumber       string    `json:"case_number"`
	CaseTitle        string    `json:"case_title"`
	OriginalFilename string    `json:"original_filename"`
	MimeType         string    `json:"mime_type"`
	ProcessingStatus string    `json:"processing_status"`
	FileSizeBytes    int64     `json:"file_size_bytes"`
	CreatedAt        time.Time `json:"created_at"`
	Rank             float32   `json:"rank"`
	EntityType       string    `json:"entity_type"`
}

type SearchResults struct {
	Cases     []CaseResult     `json:"cases"`
	Notes     []NoteResult     `json:"notes"`
	Tasks     []TaskResult     `json:"tasks"`
	Documents []DocumentResult `json:"documents"`
	Query     string           `json:"query"`
	Total     int              `json:"total"`
}

func toCaseResult(r db.SearchCasesRow) CaseResult {
	return CaseResult{
		ID:            r.ID,
		CaseNumber:    r.CaseNumber,
		Title:         r.Title,
		Description:   r.Description,
		Status:        string(r.Status),
		Priority:      string(r.Priority),
		SLABreached:   r.SlaBreached,
		CreatedByName: r.CreatedByName,
		CreatedAt:     r.CreatedAt,
		Rank:          r.Rank,
		EntityType:    "case",
	}
}

func toNoteResult(r db.SearchNotesRow) NoteResult {
	return NoteResult{
		ID:          r.ID,
		CaseID:      r.CaseID,
		CaseNumber:  r.CaseNumber,
		CaseTitle:   r.CaseTitle,
		Title:       r.Title,
		ContentText: r.ContentText,
		NoteType:    string(r.NoteType),
		IsPinned:    r.IsPinned,
		AIGenerated: r.AiGenerated,
		CreatedAt:   r.CreatedAt,
		Rank:        r.Rank,
		EntityType:  "note",
	}
}

func toTaskResult(r db.SearchTasksRow) TaskResult {
	return TaskResult{
		ID:          r.ID,
		CaseID:      r.CaseID,
		CaseNumber:  r.CaseNumber,
		CaseTitle:   r.CaseTitle,
		Title:       r.Title,
		Description: r.Description.String,
		Status:      string(r.Status),
		Priority:    string(r.Priority),
		IsOverdue:   r.IsOverdue,
		CreatedAt:   r.CreatedAt,
		Rank:        r.Rank,
		EntityType:  "task",
	}
}

func toDocumentResult(r db.SearchDocumentsRow) DocumentResult {
	return DocumentResult{
		ID:               r.ID,
		CaseID:           r.CaseID,
		CaseNumber:       r.CaseNumber,
		CaseTitle:        r.CaseTitle,
		OriginalFilename: r.OriginalFilename,
		MimeType:         r.MimeType,
		ProcessingStatus: string(r.ProcessingStatus),
		FileSizeBytes:    r.FileSizeBytes,
		CreatedAt:        r.CreatedAt,
		Rank:             r.Rank,
		EntityType:       "document",
	}
}
