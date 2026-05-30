package playbooks

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

// defaultOrgID is the singleton org used until multi-tenancy is enabled.
var defaultOrgID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// TaskCreator is the narrow interface the playbook applier uses to create tasks
// without importing the tasks package directly.
type TaskCreator interface {
	CreateRaw(ctx context.Context, p CreateTaskRawParams) error
}

// CreateTaskRawParams carries everything needed to insert a playbook task.
type CreateTaskRawParams struct {
	CaseID          uuid.UUID
	CreatedBy       uuid.UUID
	Title           string
	Description     string
	Priority        db.TaskPriority
	DueDate         *time.Time
	PlaybookStepID  *uuid.UUID
	PlaybookStepOrder int
	DependsOnStep   int
}

type Service struct {
	repo    *Repository
	creator TaskCreator
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) WithTaskCreator(tc TaskCreator) *Service {
	s.creator = tc
	return s
}

func (s *Service) Create(ctx context.Context, claims *auth.Claims, req CreatePlaybookRequest) (*PlaybookResponse, error) {
	if req.Name == "" {
		return nil, ErrNameRequired
	}
	if req.CaseType == "" {
		return nil, ErrCaseTypeRequired
	}

	pb, err := s.repo.Create(ctx, req.Name, req.Description, req.CaseType, defaultOrgID, claims.UserID)
	if err != nil {
		return nil, err
	}

	steps := make([]stepRow, 0, len(req.Steps))
	for _, step := range req.Steps {
		sr, err := s.repo.CreateStep(ctx, pb.ID, step)
		if err != nil {
			return nil, err
		}
		steps = append(steps, sr)
	}

	return toPlaybookResponse(pb, steps), nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*PlaybookResponse, error) {
	pb, err := s.repo.GetByID(ctx, id, defaultOrgID)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	steps, err := s.repo.ListSteps(ctx, id)
	if err != nil {
		return nil, err
	}
	return toPlaybookResponse(pb, steps), nil
}

func (s *Service) List(ctx context.Context) ([]PlaybookResponse, error) {
	pbs, err := s.repo.List(ctx, defaultOrgID)
	if err != nil {
		return nil, err
	}
	out := make([]PlaybookResponse, 0, len(pbs))
	for _, pb := range pbs {
		steps, err := s.repo.ListSteps(ctx, pb.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, *toPlaybookResponse(pb, steps))
	}
	return out, nil
}

func (s *Service) Update(ctx context.Context, claims *auth.Claims, id uuid.UUID, req UpdatePlaybookRequest) (*PlaybookResponse, error) {
	if claims.Role != db.UserRoleAdmin && claims.Role != db.UserRoleSuperAdmin {
		return nil, ErrForbidden
	}
	pb, err := s.repo.Update(ctx, id, defaultOrgID, req)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	steps, err := s.repo.ListSteps(ctx, pb.ID)
	if err != nil {
		return nil, err
	}
	return toPlaybookResponse(pb, steps), nil
}

func (s *Service) ReplaceSteps(ctx context.Context, claims *auth.Claims, id uuid.UUID, stepReqs []CreateStepRequest) (*PlaybookResponse, error) {
	if claims.Role != db.UserRoleAdmin && claims.Role != db.UserRoleSuperAdmin {
		return nil, ErrForbidden
	}
	pb, err := s.repo.GetByID(ctx, id, defaultOrgID)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if err := s.repo.DeleteSteps(ctx, id); err != nil {
		return nil, err
	}

	steps := make([]stepRow, 0, len(stepReqs))
	for _, step := range stepReqs {
		sr, err := s.repo.CreateStep(ctx, id, step)
		if err != nil {
			return nil, err
		}
		steps = append(steps, sr)
	}
	return toPlaybookResponse(pb, steps), nil
}

func (s *Service) Delete(ctx context.Context, claims *auth.Claims, id uuid.UUID) error {
	if claims.Role != db.UserRoleAdmin && claims.Role != db.UserRoleSuperAdmin {
		return ErrForbidden
	}
	_, err := s.repo.GetByID(ctx, id, defaultOrgID)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, id, defaultOrgID)
}

// ApplyToCase is called by the cases service after case creation.
// It finds the active playbook for caseType and creates tasks on the new case.
func (s *Service) ApplyToCase(ctx context.Context, caseID uuid.UUID, createdBy uuid.UUID, caseType string, caseCreatedAt time.Time) error {
	if caseType == "" || s.creator == nil {
		return nil
	}

	pb, err := s.repo.FindActiveByType(ctx, caseType, defaultOrgID)
	if err != nil || pb == nil {
		return nil // no playbook configured — not an error
	}

	steps, err := s.repo.ListSteps(ctx, pb.ID)
	if err != nil {
		return err
	}

	for _, step := range steps {
		var dueDate *time.Time
		if step.DueHours.Valid {
			t := caseCreatedAt.Add(time.Duration(step.DueHours.Int32) * time.Hour)
			dueDate = &t
		}
		stepID := step.ID
		p := CreateTaskRawParams{
			CaseID:            caseID,
			CreatedBy:         createdBy,
			Title:             step.Title,
			Description:       step.Description,
			Priority:          step.Priority,
			DueDate:           dueDate,
			PlaybookStepID:    &stepID,
			PlaybookStepOrder: step.StepOrder,
			DependsOnStep:     step.DependsOnStep,
		}
		if err := s.creator.CreateRaw(ctx, p); err != nil {
			return err
		}
	}
	return nil
}

// ── Response builder ──────────────────────────────────────────────────────────

func toPlaybookResponse(pb playbookRow, steps []stepRow) *PlaybookResponse {
	sr := make([]StepResponse, 0, len(steps))
	for _, s := range steps {
		var dh *int
		if s.DueHours.Valid {
			v := int(s.DueHours.Int32)
			dh = &v
		}
		sr = append(sr, StepResponse{
			ID:            s.ID,
			PlaybookID:    s.PlaybookID,
			Title:         s.Title,
			Description:   s.Description,
			Priority:      s.Priority,
			StepOrder:     s.StepOrder,
			DueHours:      dh,
			DependsOnStep: s.DependsOnStep,
			CreatedAt:     s.CreatedAt,
		})
	}
	return &PlaybookResponse{
		ID:          pb.ID,
		Name:        pb.Name,
		Description: pb.Description,
		CaseType:    pb.CaseType,
		IsActive:    pb.IsActive,
		OrgID:       pb.OrgID,
		Steps:       sr,
		CreatedAt:   pb.CreatedAt,
		UpdatedAt:   pb.UpdatedAt,
	}
}
