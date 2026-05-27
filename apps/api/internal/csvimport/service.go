package csvimport

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/reddybashanaguru/opsowl/internal/auth"
	"github.com/reddybashanaguru/opsowl/internal/cases"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

// CaseCreator is the narrow interface the import service needs from cases.Service.
type CaseCreator interface {
	Create(ctx context.Context, claims *auth.Claims, req cases.CreateCaseRequest) (*cases.CaseResponse, error)
}

type Service struct {
	q        *db.Queries
	casesSvc CaseCreator
	logger   *zap.Logger
}

func NewService(q *db.Queries, casesSvc CaseCreator, logger *zap.Logger) *Service {
	return &Service{q: q, casesSvc: casesSvc, logger: logger}
}

// ─── Preview (parse only, no DB write) ────────────────────────────────────────

func (s *Service) Preview(r io.Reader) (*ParsePreview, error) {
	headers, rows, err := parseCSV(r)
	if err != nil {
		return nil, err
	}
	sample := rows
	if len(sample) > 5 {
		sample = sample[:5]
	}
	sampleMaps := make([]map[string]string, len(sample))
	for i, row := range sample {
		sampleMaps[i] = rowToMap(headers, row)
	}
	return &ParsePreview{
		Headers:     headers,
		SampleRows:  sampleMaps,
		TotalRows:   len(rows),
		DetectedMap: autoDetectMapping(headers),
	}, nil
}

// ─── Start import job ──────────────────────────────────────────────────────────

// StartImport parses the CSV, writes all rows to DB, then processes them
// asynchronously. Returns the job immediately.
func (s *Service) StartImport(
	ctx context.Context,
	filename string,
	r io.Reader,
	mapping []ColumnMapping,
	claims *auth.Claims,
) (*ImportJob, error) {
	// Validate mapping has title
	hasTitleMapping := false
	for _, m := range mapping {
		if m.CaseField == "title" {
			hasTitleMapping = true
			break
		}
	}
	if !hasTitleMapping {
		return nil, ErrNoTitleColumn
	}

	headers, rows, err := parseCSV(r)
	if err != nil {
		return nil, err
	}
	_ = headers // used implicitly via rowToMap

	var createdBy uuid.NullUUID
	if claims != nil {
		createdBy = uuid.NullUUID{UUID: claims.UserID, Valid: true}
	}

	job, err := s.q.InsertCsvImport(ctx, db.InsertCsvImportParams{
		Filename:  filename,
		TotalRows: int32(len(rows)),
		CreatedBy: createdBy,
	})
	if err != nil {
		return nil, err
	}

	// Persist all rows as pending
	dbRows := make([]db.CsvImportRow, 0, len(rows))
	for i, record := range rows {
		rawMap := rowToMap(headers, record)
		raw, _ := json.Marshal(rawMap)
		dbRow, err := s.q.InsertCsvImportRow(ctx, db.InsertCsvImportRowParams{
			ImportID:  job.ID,
			RowNumber: int32(i + 1),
			RawData:   raw,
		})
		if err != nil {
			s.logger.Error("csvimport: failed to insert row", zap.Int("row", i+1), zap.Error(err))
			continue
		}
		dbRows = append(dbRows, dbRow)
	}

	// Mark as processing immediately
	s.q.UpdateCsvImportStatus(ctx, job.ID, "processing") //nolint

	// Process asynchronously
	go s.processRows(job.ID, dbRows, mapping, claims)

	return jobFromRow(job), nil
}

// ─── Status + rows ─────────────────────────────────────────────────────────────

func (s *Service) GetJob(ctx context.Context, id uuid.UUID) (*ImportJob, error) {
	row, err := s.q.GetCsvImport(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return jobFromRow(row), nil
}

func (s *Service) ListJobs(ctx context.Context, limit, offset int32) ([]*ImportJob, error) {
	rows, err := s.q.ListCsvImports(ctx, db.ListCsvImportsParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	out := make([]*ImportJob, len(rows))
	for i, r := range rows {
		out[i] = jobFromRow(r)
	}
	return out, nil
}

func (s *Service) ListRows(ctx context.Context, jobID uuid.UUID, statusFilter *string, limit, offset int32) (*ImportRowsPage, error) {
	var statusNull sql.NullString
	if statusFilter != nil && *statusFilter != "" {
		statusNull = sql.NullString{String: *statusFilter, Valid: true}
	}
	rows, err := s.q.ListImportRows(ctx, db.ListImportRowsParams{
		ImportID: jobID,
		Status:   statusNull,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, err
	}
	total, _ := s.q.CountImportRows(ctx, db.CountImportRowsParams{ImportID: jobID, Status: statusNull})
	out := make([]*ImportRowResult, len(rows))
	for i, r := range rows {
		out[i] = rowResultFromDB(r)
	}
	return &ImportRowsPage{
		Rows:    out,
		HasMore: int64(offset)+int64(len(rows)) < total,
		Total:   total,
	}, nil
}

// ─── Async processor ──────────────────────────────────────────────────────────

func (s *Service) processRows(jobID uuid.UUID, rows []db.CsvImportRow, mapping []ColumnMapping, claims *auth.Claims) {
	ctx := context.Background()
	var imported, failed int32

	for _, row := range rows {
		var rawMap map[string]string
		if err := json.Unmarshal(row.RawData, &rawMap); err != nil {
			s.q.MarkImportRowFailed(ctx, row.ID, "could not parse row data") //nolint
			failed++
			continue
		}

		req, err := buildCaseRequest(rawMap, mapping)
		if err != nil {
			s.q.MarkImportRowFailed(ctx, row.ID, err.Error()) //nolint
			failed++
			continue
		}

		// Resolve assigned_to_email → user ID
		if email := mappedValue(rawMap, mapping, "assigned_to_email"); email != "" {
			if u, err := s.q.GetUserByEmail(ctx, email); err == nil {
				req.AssignedTo = &u.ID
			}
		}

		created, err := s.casesSvc.Create(ctx, claims, req)
		if err != nil {
			s.q.MarkImportRowFailed(ctx, row.ID, err.Error()) //nolint
			failed++
		} else {
			s.q.MarkImportRowSuccess(ctx, row.ID, uuid.NullUUID{UUID: created.ID, Valid: true}) //nolint
			imported++
		}

		// Batch-flush counts every 50 rows to keep polling responsive
		if (imported+failed)%50 == 0 {
			s.q.IncrementImportCounts(ctx, db.IncrementImportCountsParams{ //nolint
				ID:           jobID,
				ImportedRows: imported,
				FailedRows:   failed,
			})
			imported, failed = 0, 0
		}
	}

	// Final count flush
	if imported > 0 || failed > 0 {
		s.q.IncrementImportCounts(ctx, db.IncrementImportCountsParams{ //nolint
			ID:           jobID,
			ImportedRows: imported,
			FailedRows:   failed,
		})
	}

	// Determine final status
	job, _ := s.q.GetCsvImport(ctx, jobID)
	finalStatus := "done"
	if job.FailedRows > 0 && job.ImportedRows == 0 {
		finalStatus = "failed"
	} else if job.FailedRows > 0 {
		finalStatus = "partial"
	}
	s.q.UpdateCsvImportStatus(ctx, jobID, finalStatus) //nolint
}

// ─── Row → case request ────────────────────────────────────────────────────────

func buildCaseRequest(rawMap map[string]string, mapping []ColumnMapping) (cases.CreateCaseRequest, error) {
	req := cases.CreateCaseRequest{
		Priority: db.CasePriorityMedium,
		Tags:     []string{},
	}

	title := mappedValue(rawMap, mapping, "title")
	if title == "" {
		return req, fmt.Errorf("title is empty")
	}
	if len(title) < 5 {
		return req, fmt.Errorf("title too short (min 5 chars): %q", title)
	}
	req.Title = title

	if v := mappedValue(rawMap, mapping, "description"); v != "" {
		req.Description = v
	}

	if v := mappedValue(rawMap, mapping, "priority"); v != "" {
		p := db.CasePriority(strings.ToLower(v))
		switch p {
		case db.CasePriorityCritical, db.CasePriorityHigh, db.CasePriorityMedium, db.CasePriorityLow:
			req.Priority = p
		}
	}

	if v := mappedValue(rawMap, mapping, "tags"); v != "" {
		for _, t := range strings.Split(v, ",") {
			if t := strings.TrimSpace(t); t != "" {
				req.Tags = append(req.Tags, t)
			}
		}
	}

	if v := mappedValue(rawMap, mapping, "due_date"); v != "" {
		for _, layout := range []string{"2006-01-02", "02/01/2006", time.RFC3339} {
			if t, err := time.Parse(layout, v); err == nil {
				req.DueDate = &t
				break
			}
		}
	}

	return req, nil
}

func mappedValue(rawMap map[string]string, mapping []ColumnMapping, caseField string) string {
	for _, m := range mapping {
		if m.CaseField == caseField {
			return rawMap[m.CSVHeader]
		}
	}
	return ""
}

// ─── Row converters ────────────────────────────────────────────────────────────

func jobFromRow(r db.CsvImport) *ImportJob {
	return &ImportJob{
		ID:           r.ID,
		Filename:     r.Filename,
		TotalRows:    r.TotalRows,
		ImportedRows: r.ImportedRows,
		FailedRows:   r.FailedRows,
		Status:       r.Status,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

func rowResultFromDB(r db.CsvImportRow) *ImportRowResult {
	out := &ImportRowResult{
		ID:        r.ID,
		RowNumber: r.RowNumber,
		Status:    r.Status,
	}
	if r.CaseID.Valid {
		id := r.CaseID.UUID
		out.CaseID = &id
	}
	if r.Error.Valid {
		out.Error = &r.Error.String
	}
	var rawMap map[string]string
	if json.Unmarshal(r.RawData, &rawMap) == nil {
		out.RawData = rawMap
	}
	return out
}
