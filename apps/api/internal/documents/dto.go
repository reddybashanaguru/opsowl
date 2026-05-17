package documents

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

var (
	ErrNotFound        = errors.New("document not found")
	ErrForbidden       = errors.New("forbidden")
	ErrFileTooLarge    = errors.New("file too large")
	ErrFileTypeBlocked = errors.New("file type not allowed")
)

const (
	MaxFileSizeBytes = 50 * 1024 * 1024 // 50 MB
	PresignExpiry    = time.Hour
)

type ListFilters struct {
	Status   *db.DocumentProcessingStatus
	Limit    int32
	CursorID *uuid.UUID
}

type ListMeta struct {
	TotalCount int64      `json:"total_count"`
	HasMore    bool       `json:"has_more"`
	NextCursor *uuid.UUID `json:"next_cursor"`
}

type UploaderRef struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
	Email    string    `json:"email"`
}

type DocumentResponse struct {
	ID               uuid.UUID                   `json:"id"`
	CaseID           uuid.UUID                   `json:"case_id"`
	OriginalFilename string                      `json:"original_filename"`
	FileType         string                      `json:"file_type"`
	MimeType         string                      `json:"mime_type"`
	FileSizeBytes    int64                       `json:"file_size_bytes"`
	ProcessingStatus db.DocumentProcessingStatus `json:"processing_status"`
	PageCount        *int32                      `json:"page_count"`
	ChunkCount       *int32                      `json:"chunk_count"`
	ErrorMessage     *string                     `json:"error_message"`
	UploadedBy       UploaderRef                 `json:"uploaded_by"`
	DownloadURL      string                      `json:"download_url,omitempty"`
	CreatedAt        time.Time                   `json:"created_at"`
	UpdatedAt        time.Time                   `json:"updated_at"`
}

// ProcessingResult is sent from the Python AI service callback.
type ProcessingResult struct {
	DocumentID   uuid.UUID `json:"document_id"`
	Success      bool      `json:"success"`
	PageCount    *int32    `json:"page_count"`
	ChunkCount   *int32    `json:"chunk_count"`
	ErrorMessage *string   `json:"error_message"`
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func nullInt32(i *int32) sql.NullInt32 {
	if i == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: *i, Valid: true}
}

func int32Ptr(n sql.NullInt32) *int32 {
	if !n.Valid {
		return nil
	}
	v := n.Int32
	return &v
}

func strPtr(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	v := s.String
	return &v
}
