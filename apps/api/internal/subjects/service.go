package subjects

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

var defaultOrgID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ── Subject CRUD ──────────────────────────────────────────────────────────────

func (s *Service) Create(ctx context.Context, claims *auth.Claims, req CreateSubjectRequest) (*SubjectResponse, error) {
	if req.CanonicalName == "" {
		return nil, ErrNameRequired
	}
	return s.repo.Create(ctx, defaultOrgID, claims.UserID, req)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*SubjectResponse, error) {
	return s.repo.GetByID(ctx, id, defaultOrgID)
}

func (s *Service) List(ctx context.Context, f ListFilters) ([]SubjectResponse, int64, error) {
	return s.repo.List(ctx, defaultOrgID, f)
}

func (s *Service) Update(ctx context.Context, claims *auth.Claims, id uuid.UUID, req UpdateSubjectRequest) (*SubjectResponse, error) {
	// Record risk score change in history if it changed
	existing, err := s.repo.GetByID(ctx, id, defaultOrgID)
	if err != nil {
		return nil, err
	}

	updated, err := s.repo.Update(ctx, id, defaultOrgID, req)
	if err != nil {
		return nil, err
	}

	if req.RiskScore != nil && *req.RiskScore != existing.RiskScore {
		level := updated.RiskLevel
		_ = s.repo.AppendRiskHistory(ctx, id, *req.RiskScore, level, "manual update", claims.UserID.String())
	}

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, _ *auth.Claims, id uuid.UUID) error {
	return s.repo.Delete(ctx, id, defaultOrgID)
}

// ── Case-Subject linking ──────────────────────────────────────────────────────

// ResolveOrCreate finds an existing subject by identity or creates a new one.
// Used by alert ingestion and case subject linking to ensure deduplication.
func (s *Service) ResolveOrCreate(ctx context.Context, claims *auth.Claims, req CreateSubjectRequest) (*SubjectResponse, bool, error) {
	if req.IDType != "" && req.IDNumber != "" {
		existing, err := s.repo.GetByIdentity(ctx, defaultOrgID, req.IDType, req.IDNumber)
		if err == nil {
			return existing, false, nil // found existing
		}
		if !errors.Is(err, ErrNotFound) {
			return nil, false, err
		}
	}
	sub, err := s.repo.Create(ctx, defaultOrgID, claims.UserID, req)
	if err != nil {
		return nil, false, err
	}
	return sub, true, nil
}

func (s *Service) LinkToCase(ctx context.Context, claims *auth.Claims, caseID, subjectID uuid.UUID, role string) error {
	// Ensure subject exists
	if _, err := s.repo.GetByID(ctx, subjectID, defaultOrgID); err != nil {
		return err
	}
	return s.repo.LinkToCase(ctx, caseID, subjectID, claims.UserID, role)
}

func (s *Service) UnlinkFromCase(ctx context.Context, caseID, subjectID uuid.UUID) error {
	return s.repo.UnlinkFromCase(ctx, caseID, subjectID)
}

func (s *Service) ListForCase(ctx context.Context, caseID uuid.UUID) ([]CaseSubjectRef, error) {
	return s.repo.ListForCase(ctx, caseID)
}

// ── Subject 360 data ──────────────────────────────────────────────────────────

func (s *Service) CasesForSubject(ctx context.Context, subjectID uuid.UUID) ([]SubjectCaseRef, error) {
	return s.repo.ListCasesForSubject(ctx, subjectID)
}

func (s *Service) FilingsForSubject(ctx context.Context, subjectID uuid.UUID) ([]map[string]any, error) {
	return s.repo.ListFilingsForSubject(ctx, subjectID, defaultOrgID)
}

func (s *Service) RiskHistory(ctx context.Context, subjectID uuid.UUID) ([]RiskHistoryEntry, error) {
	// Ensure subject exists
	if _, err := s.repo.GetByID(ctx, subjectID, defaultOrgID); err != nil {
		return nil, err
	}
	return s.repo.RiskHistory(ctx, subjectID)
}

func (s *Service) SetWatchlist(ctx context.Context, claims *auth.Claims, subjectID uuid.UUID, on bool) (*SubjectResponse, error) {
	return s.repo.Update(ctx, subjectID, defaultOrgID, UpdateSubjectRequest{IsWatchlisted: &on})
}

// ── Fuzzy Entity Resolution ───────────────────────────────────────────────────

func (s *Service) FindSimilar(ctx context.Context, subjectID uuid.UUID, threshold float64, limit int) ([]SimilarSubject, error) {
	return s.repo.FindSimilar(ctx, subjectID, threshold, limit)
}

type ConfirmMatchRequest struct {
	TargetSubjectID string `json:"target_subject_id"`
	RelationshipType string `json:"relationship_type"` // SAME_ENTITY | ALIAS_OF
	Notes           string `json:"notes"`
}

func (s *Service) ConfirmMatch(ctx context.Context, claims *auth.Claims, fromID uuid.UUID, req ConfirmMatchRequest) error {
	toID, err := uuid.Parse(req.TargetSubjectID)
	if err != nil {
		return errors.New("invalid target_subject_id")
	}
	if fromID == toID {
		return errors.New("cannot link a subject to itself")
	}
	relType := req.RelationshipType
	if relType == "" {
		relType = "SAME_ENTITY"
	}
	if relType != "SAME_ENTITY" && relType != "ALIAS_OF" {
		return errors.New("relationship_type must be SAME_ENTITY or ALIAS_OF")
	}
	return s.repo.ConfirmMatch(ctx, fromID, toID, claims.UserID, req.Notes, relType)
}
