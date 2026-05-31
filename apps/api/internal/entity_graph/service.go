package entity_graph

import (
	"context"

	"github.com/google/uuid"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, claims *auth.Claims, fromID uuid.UUID, req CreateRelationshipRequest) (*RelationshipResponse, error) {
	if fromID == req.ToSubjectID {
		return nil, ErrSelfLink
	}
	if !ValidRelationshipType(req.RelationshipType) {
		return nil, ErrInvalidType
	}
	return s.repo.Create(ctx, fromID, req.ToSubjectID, claims.UserID, req)
}

func (s *Service) List(ctx context.Context, subjectID uuid.UUID) ([]RelationshipResponse, error) {
	return s.repo.List(ctx, subjectID)
}

func (s *Service) Delete(ctx context.Context, _ *auth.Claims, subjectID, relID uuid.UUID) error {
	return s.repo.Deactivate(ctx, relID, subjectID)
}

func (s *Service) GetNetwork(ctx context.Context, subjectID uuid.UUID, depth int) (*NetworkResponse, error) {
	return s.repo.Traverse(ctx, subjectID, depth)
}
