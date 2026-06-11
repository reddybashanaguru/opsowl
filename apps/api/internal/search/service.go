package search

import (
	"context"
	"database/sql"
	"sync"

	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

type Service struct {
	queries *db.Queries
	sqlDB   *sql.DB
}

func NewService(queries *db.Queries) *Service {
	return &Service{queries: queries}
}

func (s *Service) WithDB(sqlDB *sql.DB) *Service {
	s.sqlDB = sqlDB
	return s
}

func (s *Service) Search(ctx context.Context, orgID uuid.UUID, req SearchRequest) (*SearchResults, error) {
	if req.Query == "" {
		return &SearchResults{
			Cases: []CaseResult{}, Notes: []NoteResult{},
			Tasks: []TaskResult{}, Documents: []DocumentResult{},
			Query: req.Query,
		}, nil
	}

	limit := req.Limit
	if limit <= 0 || limit > 20 {
		limit = 10
	}

	filter := req.Filter

	var (
		mu    sync.Mutex
		wg    sync.WaitGroup
		errs  []error
		cases []CaseResult
		notes []NoteResult
		tasks []TaskResult
		docs  []DocumentResult
	)

	runCases := filter == "" || filter == "cases"
	runNotes := filter == "" || filter == "notes"
	runTasks := filter == "" || filter == "tasks"
	runDocs := filter == "" || filter == "documents"

	if runCases {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rows, err := s.queries.SearchCases(ctx, db.SearchCasesParams{
				Query: req.Query, OrganizationID: orgID, RowLimit: limit,
			})
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			for _, r := range rows {
				cases = append(cases, toCaseResult(r))
			}
		}()
	}

	if runNotes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rows, err := s.queries.SearchNotes(ctx, db.SearchNotesParams{
				Query: req.Query, OrganizationID: orgID, RowLimit: limit,
			})
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			for _, r := range rows {
				notes = append(notes, toNoteResult(r))
			}
		}()
	}

	if runTasks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rows, err := s.queries.SearchTasks(ctx, db.SearchTasksParams{
				Query: req.Query, OrganizationID: orgID, RowLimit: limit,
			})
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			for _, r := range rows {
				tasks = append(tasks, toTaskResult(r))
			}
		}()
	}

	if runDocs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rows, err := s.queries.SearchDocuments(ctx, db.SearchDocumentsParams{
				Query: req.Query, OrganizationID: orgID, RowLimit: limit,
			})
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			for _, r := range rows {
				docs = append(docs, toDocumentResult(r))
			}
		}()
	}

	wg.Wait()

	// ILIKE fallback: tsvector can miss rows when the index is stale or freshly inserted.
	// If a category returned no results, retry with a plain ILIKE to guarantee freshness.
	if s.sqlDB != nil {
		likeParam := "%" + req.Query + "%"
		if runCases && len(cases) == 0 {
			cases = s.ilikeCases(ctx, orgID, likeParam, limit)
		}
		if runNotes && len(notes) == 0 {
			notes = s.ilikeNotes(ctx, orgID, likeParam, limit)
		}
		if runTasks && len(tasks) == 0 {
			tasks = s.ilikeTasks(ctx, orgID, likeParam, limit)
		}
	}

	if cases == nil {
		cases = []CaseResult{}
	}
	if notes == nil {
		notes = []NoteResult{}
	}
	if tasks == nil {
		tasks = []TaskResult{}
	}
	if docs == nil {
		docs = []DocumentResult{}
	}

	total := len(cases) + len(notes) + len(tasks) + len(docs)
	return &SearchResults{
		Cases:     cases,
		Notes:     notes,
		Tasks:     tasks,
		Documents: docs,
		Query:     req.Query,
		Total:     total,
	}, nil
}

func (s *Service) ilikeCases(ctx context.Context, orgID uuid.UUID, like string, limit int32) []CaseResult {
	const q = `
		SELECT c.id, c.case_number, c.title, c.description, c.status, c.priority,
		       c.sla_breached, c.created_at, u.full_name AS created_by_name
		FROM cases c
		JOIN users u ON u.id = c.created_by
		WHERE c.deleted_at IS NULL
		  AND c.organization_id = $1
		  AND (c.title ILIKE $2 OR c.description ILIKE $2 OR c.case_number ILIKE $2)
		ORDER BY c.created_at DESC
		LIMIT $3`

	rows, err := s.sqlDB.QueryContext(ctx, q, orgID, like, limit)
	if err != nil {
		return nil
	}
	defer rows.Close() //nolint:errcheck
	var out []CaseResult
	for rows.Next() {
		var r CaseResult
		r.EntityType = "case"
		if err := rows.Scan(&r.ID, &r.CaseNumber, &r.Title, &r.Description,
			&r.Status, &r.Priority, &r.SLABreached, &r.CreatedAt, &r.CreatedByName); err != nil {
			continue
		}
		out = append(out, r)
	}
	return out
}

func (s *Service) ilikeNotes(ctx context.Context, orgID uuid.UUID, like string, limit int32) []NoteResult {
	const q = `
		SELECT n.id, n.case_id, n.title, n.content_text, n.note_type,
		       n.is_pinned, n.ai_generated, n.created_at,
		       c.case_number, c.title AS case_title
		FROM notes n
		JOIN cases c ON c.id = n.case_id AND c.deleted_at IS NULL
		WHERE n.deleted_at IS NULL
		  AND c.organization_id = $1
		  AND (n.title ILIKE $2 OR n.content_text ILIKE $2)
		ORDER BY n.created_at DESC
		LIMIT $3`

	rows, err := s.sqlDB.QueryContext(ctx, q, orgID, like, limit)
	if err != nil {
		return nil
	}
	defer rows.Close() //nolint:errcheck
	var out []NoteResult
	for rows.Next() {
		var r NoteResult
		r.EntityType = "note"
		if err := rows.Scan(&r.ID, &r.CaseID, &r.Title, &r.ContentText, &r.NoteType,
			&r.IsPinned, &r.AIGenerated, &r.CreatedAt, &r.CaseNumber, &r.CaseTitle); err != nil {
			continue
		}
		out = append(out, r)
	}
	return out
}

func (s *Service) ilikeTasks(ctx context.Context, orgID uuid.UUID, like string, limit int32) []TaskResult {
	const q = `
		SELECT t.id, t.case_id, t.title, COALESCE(t.description, ''), t.status,
		       t.priority, t.is_overdue, t.created_at,
		       c.case_number, c.title AS case_title
		FROM tasks t
		JOIN cases c ON c.id = t.case_id AND c.deleted_at IS NULL
		WHERE t.deleted_at IS NULL
		  AND c.organization_id = $1
		  AND (t.title ILIKE $2 OR t.description ILIKE $2)
		ORDER BY t.created_at DESC
		LIMIT $3`

	rows, err := s.sqlDB.QueryContext(ctx, q, orgID, like, limit)
	if err != nil {
		return nil
	}
	defer rows.Close() //nolint:errcheck
	var out []TaskResult
	for rows.Next() {
		var r TaskResult
		r.EntityType = "task"
		if err := rows.Scan(&r.ID, &r.CaseID, &r.Title, &r.Description, &r.Status,
			&r.Priority, &r.IsOverdue, &r.CreatedAt, &r.CaseNumber, &r.CaseTitle); err != nil {
			continue
		}
		out = append(out, r)
	}
	return out
}

