package csvimport

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound      = errors.New("import job not found")
	ErrNoTitleColumn = errors.New("column mapping must include 'title'")
	ErrFileTooLarge  = errors.New("CSV file exceeds 5 MB limit")
	ErrTooManyRows   = errors.New("CSV file exceeds 5000 row limit")
	ErrEmptyFile     = errors.New("CSV file is empty or has no data rows")
)

// SupportedColumns are the case fields a CSV column can map to.
var SupportedColumns = []string{
	"title",
	"description",
	"priority",
	"tags",
	"due_date",
	"assigned_to_email",
}

// ColumnMapping maps a CSV header to a case field.
// e.g. {"csv_header": "Alert Title", "case_field": "title"}
type ColumnMapping struct {
	CSVHeader string `json:"csv_header"`
	CaseField string `json:"case_field"`
}

type ImportJob struct {
	ID           uuid.UUID  `json:"id"`
	Filename     string     `json:"filename"`
	TotalRows    int32      `json:"total_rows"`
	ImportedRows int32      `json:"imported_rows"`
	FailedRows   int32      `json:"failed_rows"`
	Status       string     `json:"status"` // pending, processing, done, partial, failed
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type ImportRowResult struct {
	ID        uuid.UUID  `json:"id"`
	RowNumber int32      `json:"row_number"`
	RawData   map[string]string `json:"raw_data"`
	Status    string     `json:"status"` // pending, success, failed
	CaseID    *uuid.UUID `json:"case_id,omitempty"`
	Error     *string    `json:"error,omitempty"`
}

type ImportRowsPage struct {
	Rows    []*ImportRowResult `json:"rows"`
	HasMore bool               `json:"has_more"`
	Total   int64              `json:"total"`
}

// ParsePreview is returned from the parse step before the user confirms mapping.
type ParsePreview struct {
	Headers      []string            `json:"headers"`
	SampleRows   []map[string]string `json:"sample_rows"` // first 5 rows
	TotalRows    int                 `json:"total_rows"`
	DetectedMap  []ColumnMapping     `json:"detected_map"` // best-guess auto-mapping
}
