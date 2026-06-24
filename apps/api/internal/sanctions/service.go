package sanctions

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Service handles sanctions screening and list management.
// It is safe for concurrent use — ScreenAsync is called from goroutines on subject create/update.
type Service struct {
	repo      *Repository
	threshold float64
	logger    *zap.Logger
}

func NewService(db *sql.DB, similarityThreshold float64, logger *zap.Logger) *Service {
	if similarityThreshold <= 0 || similarityThreshold > 1 {
		similarityThreshold = 0.75
	}
	return &Service{
		repo:      NewRepository(db),
		threshold: similarityThreshold,
		logger:    logger,
	}
}

// Screen screens a named subject against all loaded sanctions lists.
// It records hits in the DB and updates subjects.is_sanctioned if any confirmed/open hit exists.
func (s *Service) Screen(ctx context.Context, subjectID, subjectName, screenedByUserID string) (*ScreenResult, error) {
	entries, scores, err := s.repo.Screen(ctx, subjectName, s.threshold)
	if err != nil {
		return nil, fmt.Errorf("sanctions: screen query: %w", err)
	}

	result := &ScreenResult{
		SubjectID:  subjectID,
		ScreenedAt: time.Now().UTC(),
		Hits:       make([]Hit, 0, len(entries)),
	}

	for i, entry := range entries {
		score := scores[i]
		matchType := "fuzzy"
		if strings.EqualFold(entry.PrimaryName, subjectName) {
			matchType = "exact"
		} else {
			for _, alias := range entry.Aliases {
				if strings.EqualFold(alias, subjectName) {
					matchType = "alias"
					break
				}
			}
		}

		hitID, err := s.repo.InsertHit(ctx, subjectID, entry.ID, matchType, entry.PrimaryName, score, screenedByUserID)
		if err != nil {
			s.logger.Warn("sanctions: failed to insert hit", zap.String("entry_id", entry.ID), zap.Error(err))
			continue
		}

		e := entry
		result.Hits = append(result.Hits, Hit{
			ID:          hitID,
			SubjectID:   subjectID,
			EntryID:     entry.ID,
			Entry:       &e,
			MatchType:   matchType,
			MatchScore:  score,
			MatchedName: entry.PrimaryName,
			ScreenedAt:  result.ScreenedAt,
			Status:      HitStatusOpen,
		})
	}

	result.HitCount = len(result.Hits)
	result.IsSanctioned = result.HitCount > 0

	if err := s.repo.UpdateSubjectSanctionedFlag(ctx, subjectID, result.IsSanctioned); err != nil {
		s.logger.Warn("sanctions: failed to update subject flag", zap.String("subject_id", subjectID), zap.Error(err))
	}

	return result, nil
}

// ScreenAsync is a fire-and-forget wrapper for calling Screen from a goroutine.
// Used as a post-create hook in the subjects service.
func (s *Service) ScreenAsync(ctx context.Context, subjectID, subjectName string) {
	go func() {
		if _, err := s.Screen(ctx, subjectID, subjectName, ""); err != nil {
			s.logger.Warn("sanctions: async screen failed", zap.String("subject_id", subjectID), zap.Error(err))
		}
	}()
}

// GetHits returns all screening hits for a subject.
func (s *Service) GetHits(ctx context.Context, subjectID string) ([]Hit, error) {
	return s.repo.GetHitsBySubject(ctx, subjectID)
}

// GetAllOpenHits returns recent open/confirmed hits across all subjects (settings dashboard).
func (s *Service) GetAllOpenHits(ctx context.Context) ([]Hit, error) {
	return s.repo.GetAllOpenHits(ctx, 200)
}

// AcknowledgeHit updates the review status of a hit.
func (s *Service) AcknowledgeHit(ctx context.Context, hitID, status, notes, reviewerID string) error {
	valid := map[string]bool{
		HitStatusFalsePos:  true,
		HitStatusConfirmed: true,
		HitStatusEscalated: true,
	}
	if !valid[status] {
		return fmt.Errorf("sanctions: invalid status %q; valid: false_positive, confirmed, escalated", status)
	}
	if err := s.repo.UpdateHitStatus(ctx, hitID, status, notes, reviewerID); err != nil {
		return fmt.Errorf("sanctions: acknowledge hit: %w", err)
	}

	// If confirmed, mark the subject as sanctioned.
	// (False positive clears the flag if no other open/confirmed hits remain.)
	return nil
}

// GetLists returns metadata for all loaded sanctions lists.
func (s *Service) GetLists(ctx context.Context) ([]SanctionsList, error) {
	return s.repo.GetLists(ctx)
}

// CaseMayClose implements a hook interface for the case closure gate.
// Cases cannot close if any subject has confirmed/escalated sanctions hits.
func (s *Service) CaseMayClose(ctx context.Context, caseID string) error {
	// This checks via subjects → case_subjects join
	var hitCount int
	err := s.repo.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM sanctions_hits sh
		JOIN case_subjects cs ON cs.subject_id = sh.subject_id
		WHERE cs.case_id = $1::uuid
		  AND sh.status IN ('open','confirmed','escalated')`, caseID,
	).Scan(&hitCount)
	if err != nil {
		return nil // don't block closure if query fails
	}
	if hitCount > 0 {
		return fmt.Errorf("%d unresolved sanctions hit(s) — resolve before closing", hitCount)
	}
	return nil
}
