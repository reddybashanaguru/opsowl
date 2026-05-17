package search

import (
	"context"
	"sync"

	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

type Service struct {
	queries *db.Queries
}

func NewService(queries *db.Queries) *Service {
	return &Service{queries: queries}
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
