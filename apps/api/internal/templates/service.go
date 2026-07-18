package templates

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"github.com/reddybashanaguru/opsowl/internal/auth"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/tasks"
)

// TaskCreator is the narrow interface needed from tasks.Service.
type TaskCreator interface {
	Create(ctx context.Context, claims *auth.Claims, caseID uuid.UUID, req tasks.CreateTaskRequest) (*tasks.TaskResponse, error)
}

type Service struct {
	q        *db.Queries
	tasksSvc TaskCreator
}

func NewService(q *db.Queries, tasksSvc TaskCreator) *Service {
	return &Service{q: q, tasksSvc: tasksSvc}
}

// ─── CRUD ──────────────────────────────────────────────────────────────────────

func (s *Service) Create(ctx context.Context, req CreateTemplateRequest, claims *auth.Claims) (*Template, error) {
	if req.Name == "" {
		return nil, ErrNameRequired
	}
	prio := db.CasePriorityMedium
	if p := db.CasePriority(req.Priority); p != "" {
		prio = p
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	var createdBy uuid.NullUUID
	if claims != nil {
		createdBy = uuid.NullUUID{UUID: claims.UserID, Valid: true}
	}
	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	tmpl, err := s.q.InsertCaseTemplate(ctx, db.InsertCaseTemplateParams{
		Name:                req.Name,
		DescriptionSkeleton: req.DescriptionSkeleton,
		Priority:            prio,
		Tags:                tags,
		IsActive:            active,
		CreatedBy:           createdBy,
	})
	if err != nil {
		return nil, err
	}

	taskRows, err := s.saveTasks(ctx, tmpl.ID, req.Tasks)
	if err != nil {
		return nil, err
	}
	return templateFromRow(tmpl, taskRows), nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Template, error) {
	tmpl, err := s.q.GetCaseTemplate(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	taskRows, _ := s.q.ListTemplateTasks(ctx, id)
	return templateFromRow(tmpl, taskRows), nil
}

func (s *Service) List(ctx context.Context, activeOnly bool) ([]*Template, error) {
	var filter sql.NullBool
	if activeOnly {
		filter = sql.NullBool{Bool: true, Valid: true}
	}
	rows, err := s.q.ListCaseTemplates(ctx, db.ListCaseTemplatesParams{ActiveOnly: filter})
	if err != nil {
		return nil, err
	}
	out := make([]*Template, len(rows))
	for i, r := range rows {
		taskRows, _ := s.q.ListTemplateTasks(ctx, r.ID)
		out[i] = templateFromRow(r, taskRows)
	}
	return out, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateTemplateRequest) (*Template, error) {
	if req.Name == "" {
		return nil, ErrNameRequired
	}
	if _, err := s.q.GetCaseTemplate(ctx, id); err != nil {
		return nil, ErrNotFound
	}
	prio := db.CasePriorityMedium
	if p := db.CasePriority(req.Priority); p != "" {
		prio = p
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	tmpl, err := s.q.UpdateCaseTemplate(ctx, db.UpdateCaseTemplateParams{
		ID:                  id,
		Name:                req.Name,
		DescriptionSkeleton: req.DescriptionSkeleton,
		Priority:            prio,
		Tags:                tags,
		IsActive:            active,
	})
	if err != nil {
		return nil, err
	}

	// Replace tasks: delete old, insert new
	_ = s.q.DeleteTemplateTasksByTemplate(ctx, id)
	taskRows, err := s.saveTasks(ctx, id, req.Tasks)
	if err != nil {
		return nil, err
	}
	return templateFromRow(tmpl, taskRows), nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.q.GetCaseTemplate(ctx, id); err != nil {
		return ErrNotFound
	}
	return s.q.DeleteCaseTemplate(ctx, id)
}

// ─── Apply ─────────────────────────────────────────────────────────────────────

// Apply creates all template tasks on the given case.
func (s *Service) Apply(ctx context.Context, templateID, caseID uuid.UUID, claims *auth.Claims) error {
	taskRows, err := s.q.ListTemplateTasks(ctx, templateID)
	if err != nil {
		return err
	}
	for _, t := range taskRows {
		prio := db.TaskPriority(t.Priority)
		s.tasksSvc.Create(ctx, claims, caseID, tasks.CreateTaskRequest{ //nolint
			Title:       t.Title,
			Description: t.Description,
			Priority:    prio,
			TemplateID:  &templateID,
		})
	}
	return nil
}

// ─── Helpers ───────────────────────────────────────────────────────────────────

func (s *Service) saveTasks(ctx context.Context, templateID uuid.UUID, stubs []TaskStub) ([]db.CaseTemplateTask, error) {
	var out []db.CaseTemplateTask
	for i, stub := range stubs {
		if stub.Title == "" {
			continue
		}
		prio := db.TaskPriorityMedium
		if p := db.TaskPriority(stub.Priority); p != "" {
			prio = p
		}
		sortOrder := int32(i)
		if stub.SortOrder > 0 {
			sortOrder = stub.SortOrder
		}
		row, err := s.q.InsertTemplateTask(ctx, db.InsertTemplateTaskParams{
			TemplateID:  templateID,
			Title:       stub.Title,
			Description: stub.Description,
			Priority:    prio,
			SortOrder:   sortOrder,
		})
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, nil
}

// CasePriorityFor returns cases.CreateCaseRequest-compatible fields from a template.
func CasePriorityFor(t *Template) db.CasePriority {
	if p := db.CasePriority(t.Priority); p != "" {
		return p
	}
	return db.CasePriorityMedium
}
