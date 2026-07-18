package findings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/timeline"
	"github.com/reddybashanaguru/opsowl/internal/worker"
)

// Timeline event type constants for findings.
// Added by migration 00039_findings.sql — defined here as string casts
// to avoid requiring sqlc regeneration before the module can compile.
const (
	evtFindingCreated        = db.TimelineEventType("finding_created")
	evtFindingStatusChanged  = db.TimelineEventType("finding_status_changed")
	evtFindingEvidenceLinked = db.TimelineEventType("finding_evidence_linked")
	evtFindingEvidenceRemoved = db.TimelineEventType("finding_evidence_removed")
)

var (
	ErrNotFound         = errors.New("finding not found")
	ErrForbidden        = errors.New("forbidden")
	ErrBadRequest       = errors.New("bad request")
	ErrInvalidStatus    = errors.New("invalid status transition")
	ErrEvidenceRequired = errors.New("high/critical findings require at least one evidence link before submission")
	ErrCaseClosed       = errors.New("case is closed — no new findings may be added")
)

// validCategories is the allowed set for finding category.
var validCategories = map[string]bool{
	"structuring": true, "layering": true, "integration": true,
	"smurfing": true, "round_tripping": true, "pep_exposure": true,
	"sanctions_evasion": true, "unusual_cash": true, "rapid_movement": true,
	"identity_fraud": true, "account_takeover": true, "other": true,
}

// validSeverities is the allowed set for finding severity.
var validSeverities = map[string]bool{
	"low": true, "medium": true, "high": true, "critical": true,
}

type Service struct {
	repo    *Repository
	emitter timeline.Emitter
}

func NewService(repo *Repository, emitter timeline.Emitter) *Service {
	return &Service{repo: repo, emitter: emitter}
}

// ── CRUD ──────────────────────────────────────────────────────────────────────

func (s *Service) Create(ctx context.Context, claims *auth.Claims, caseID uuid.UUID, req CreateFindingRequest) (FindingResponse, error) {
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Description) == "" {
		return FindingResponse{}, ErrBadRequest
	}
	if req.Category != "" && !validCategories[req.Category] {
		return FindingResponse{}, ErrBadRequest
	}
	if req.Severity != "" && !validSeverities[req.Severity] {
		return FindingResponse{}, ErrBadRequest
	}
	if s.repo.IsCaseClosed(ctx, caseID) {
		return FindingResponse{}, ErrCaseClosed
	}

	row, err := s.repo.Create(ctx, caseID, claims.UserID, req)
	if err != nil {
		return FindingResponse{}, err
	}

	actorID := claims.UserID
	worker.TrySubmit(func() {
		s.emitter.Emit(context.Background(), timeline.Event{ //nolint:errcheck
			CaseID:     caseID,
			EventType:  evtFindingCreated,
			Source:     db.TimelineEventSourceUser,
			ActorID:    &actorID,
			OccurredAt: time.Now(),
			Payload: map[string]any{
				"finding_id": row.ID.String(),
				"title":      row.Title,
				"category":   row.Category,
				"severity":   row.Severity,
				"ai_generated": row.AIGenerated,
			},
		})
	})

	return toResponse(row), nil
}

func (s *Service) Get(ctx context.Context, caseID, findingID uuid.UUID) (FindingResponse, error) {
	row, err := s.repo.GetFindingByCaseAndID(ctx, caseID, findingID)
	if err != nil {
		return FindingResponse{}, ErrNotFound
	}

	resp := toResponse(row)

	// Enrich with evidence links
	links, err := s.repo.ListEvidence(ctx, findingID)
	if err == nil && len(links) > 0 {
		details := s.repo.ResolveEvidenceDetails(ctx, links)
		for _, link := range links {
			el := toEvidenceLinkResponse(link)
			// Attach resolved title/preview if available
			_ = details // used by caller if they need it
			resp.EvidenceLinks = append(resp.EvidenceLinks, el)
		}
	}

	return resp, nil
}

func (s *Service) List(ctx context.Context, caseID uuid.UUID, statusFilter, severityFilter string, limit int, cursorStr string) ([]FindingResponse, ListMeta, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	var cursorID uuid.UUID
	if cursorStr != "" {
		cursorID, _ = uuid.Parse(cursorStr)
	}

	rows, err := s.repo.List(ctx, caseID, statusFilter, severityFilter, limit+1, cursorID)
	if err != nil {
		return nil, ListMeta{}, err
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	total, _ := s.repo.Count(ctx, caseID, statusFilter, severityFilter)

	var nextCursor *string
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1].ID.String()
		nextCursor = &last
	}

	out := make([]FindingResponse, len(rows))
	for i, r := range rows {
		out[i] = toResponse(r)
	}

	return out, ListMeta{TotalCount: total, HasMore: hasMore, NextCursor: nextCursor}, nil
}

func (s *Service) Update(ctx context.Context, claims *auth.Claims, caseID, findingID uuid.UUID, req UpdateFindingRequest) (FindingResponse, error) {
	row, err := s.repo.GetFindingByCaseAndID(ctx, caseID, findingID)
	if err != nil {
		return FindingResponse{}, ErrNotFound
	}

	// Only creator or manager+ can update; only draft findings can be edited
	if row.Status != "draft" {
		if claims.Role != db.UserRoleManager && claims.Role != db.UserRoleAdmin && claims.Role != db.UserRoleSuperAdmin {
			return FindingResponse{}, ErrForbidden
		}
	} else if row.CreatedBy != claims.UserID {
		if claims.Role != db.UserRoleManager && claims.Role != db.UserRoleAdmin && claims.Role != db.UserRoleSuperAdmin {
			return FindingResponse{}, ErrForbidden
		}
	}

	if req.Category != nil && !validCategories[*req.Category] {
		return FindingResponse{}, ErrBadRequest
	}
	if req.Severity != nil && !validSeverities[*req.Severity] {
		return FindingResponse{}, ErrBadRequest
	}

	if err := s.repo.Update(ctx, findingID, req); err != nil {
		return FindingResponse{}, err
	}

	updated, err := s.repo.GetByID(ctx, findingID)
	if err != nil {
		return FindingResponse{}, err
	}
	return toResponse(updated), nil
}

func (s *Service) Delete(ctx context.Context, claims *auth.Claims, caseID, findingID uuid.UUID) error {
	row, err := s.repo.GetFindingByCaseAndID(ctx, caseID, findingID)
	if err != nil {
		return ErrNotFound
	}
	// Only creator (draft only) or admin+ can delete
	if row.Status != "draft" && claims.Role != db.UserRoleAdmin && claims.Role != db.UserRoleSuperAdmin {
		return ErrForbidden
	}
	if row.CreatedBy != claims.UserID && claims.Role != db.UserRoleAdmin && claims.Role != db.UserRoleSuperAdmin {
		return ErrForbidden
	}
	return s.repo.SoftDelete(ctx, findingID)
}

// ── Status transitions ────────────────────────────────────────────────────────

// Submit moves a finding from draft → under_review.
// For HIGH/CRITICAL findings, at least one piece of evidence must be linked first.
func (s *Service) Submit(ctx context.Context, claims *auth.Claims, caseID, findingID uuid.UUID) (FindingResponse, error) {
	row, err := s.repo.GetFindingByCaseAndID(ctx, caseID, findingID)
	if err != nil {
		return FindingResponse{}, ErrNotFound
	}
	if claims.Role == db.UserRoleViewer {
		return FindingResponse{}, ErrForbidden
	}
	if row.Status != "draft" && row.Status != "contested" {
		return FindingResponse{}, ErrInvalidStatus
	}

	// Evidence required for high/critical
	if row.Severity == "high" || row.Severity == "critical" {
		links, _ := s.repo.ListEvidence(ctx, findingID)
		if len(links) == 0 {
			return FindingResponse{}, ErrEvidenceRequired
		}
	}

	now := time.Now()
	if err := s.repo.UpdateStatus(ctx, findingID, claims.UserID, "under_review", &now, nil); err != nil {
		return FindingResponse{}, err
	}

	s.emitStatusChange(ctx, caseID, findingID, claims.UserID, "draft", "under_review", row.Title)
	return s.Get(ctx, caseID, findingID)
}

// Approve moves a finding from under_review → approved.
// Requires manager+ role. Creates an official signed-off finding record.
func (s *Service) Approve(ctx context.Context, claims *auth.Claims, caseID, findingID uuid.UUID) (FindingResponse, error) {
	if claims.Role != db.UserRoleManager && claims.Role != db.UserRoleAdmin && claims.Role != db.UserRoleSuperAdmin {
		return FindingResponse{}, ErrForbidden
	}

	row, err := s.repo.GetFindingByCaseAndID(ctx, caseID, findingID)
	if err != nil {
		return FindingResponse{}, ErrNotFound
	}
	if row.Status != "under_review" {
		return FindingResponse{}, ErrInvalidStatus
	}
	if row.CreatedBy == claims.UserID {
		return FindingResponse{}, fmt.Errorf("%w: approver must be different from the finding creator (4-eye principle)", ErrForbidden)
	}

	now := time.Now()
	if err := s.repo.UpdateStatus(ctx, findingID, claims.UserID, "approved", &now, &now); err != nil {
		return FindingResponse{}, err
	}

	s.emitStatusChange(ctx, caseID, findingID, claims.UserID, "under_review", "approved", row.Title)
	return s.Get(ctx, caseID, findingID)
}

// Reject moves a finding from under_review → rejected.
func (s *Service) Reject(ctx context.Context, claims *auth.Claims, caseID, findingID uuid.UUID, reason string) (FindingResponse, error) {
	if claims.Role != db.UserRoleManager && claims.Role != db.UserRoleAdmin && claims.Role != db.UserRoleSuperAdmin {
		return FindingResponse{}, ErrForbidden
	}

	row, err := s.repo.GetFindingByCaseAndID(ctx, caseID, findingID)
	if err != nil {
		return FindingResponse{}, ErrNotFound
	}
	if row.Status != "under_review" {
		return FindingResponse{}, ErrInvalidStatus
	}

	now := time.Now()
	if err := s.repo.UpdateStatus(ctx, findingID, claims.UserID, "rejected", &now, nil); err != nil {
		return FindingResponse{}, err
	}

	actorID := claims.UserID
	worker.TrySubmit(func() {
		s.emitter.Emit(context.Background(), timeline.Event{ //nolint:errcheck
			CaseID:     caseID,
			EventType:  evtFindingStatusChanged,
			Source:     db.TimelineEventSourceUser,
			ActorID:    &actorID,
			OccurredAt: time.Now(),
			Payload: map[string]any{
				"finding_id": findingID.String(),
				"title":      row.Title,
				"from":       "under_review",
				"to":         "rejected",
				"reason":     reason,
			},
		})
	})

	return s.Get(ctx, caseID, findingID)
}

// Contest allows an analyst to dispute a rejection and provide a rebuttal.
func (s *Service) Contest(ctx context.Context, claims *auth.Claims, caseID, findingID uuid.UUID, reason string) (FindingResponse, error) {
	row, err := s.repo.GetFindingByCaseAndID(ctx, caseID, findingID)
	if err != nil {
		return FindingResponse{}, ErrNotFound
	}
	if row.CreatedBy != claims.UserID {
		return FindingResponse{}, ErrForbidden
	}
	if row.Status != "rejected" {
		return FindingResponse{}, ErrInvalidStatus
	}

	now := time.Now()
	if err := s.repo.UpdateStatus(ctx, findingID, claims.UserID, "contested", &now, nil); err != nil {
		return FindingResponse{}, err
	}

	s.emitStatusChange(ctx, caseID, findingID, claims.UserID, "rejected", "contested", row.Title)
	return s.Get(ctx, caseID, findingID)
}

// ── Evidence management ───────────────────────────────────────────────────────

func (s *Service) LinkEvidence(ctx context.Context, claims *auth.Claims, caseID, findingID uuid.UUID, req LinkEvidenceRequest) (EvidenceLinkResponse, error) {
	_, err := s.repo.GetFindingByCaseAndID(ctx, caseID, findingID)
	if err != nil {
		return EvidenceLinkResponse{}, ErrNotFound
	}

	// Validate evidence_type
	validTypes := map[string]bool{
		"document": true, "note": true, "transaction": true,
		"ai_log": true, "subject": true, "alert": true,
	}
	if !validTypes[req.EvidenceType] {
		return EvidenceLinkResponse{}, ErrBadRequest
	}

	link, err := s.repo.LinkEvidence(ctx, findingID, claims.UserID, req)
	if err != nil {
		return EvidenceLinkResponse{}, err
	}

	actorID := claims.UserID
	worker.TrySubmit(func() {
		s.emitter.Emit(context.Background(), timeline.Event{ //nolint:errcheck
			CaseID:     caseID,
			EventType:  evtFindingEvidenceLinked,
			Source:     db.TimelineEventSourceUser,
			ActorID:    &actorID,
			OccurredAt: time.Now(),
			Payload: map[string]any{
				"finding_id":    findingID.String(),
				"evidence_type": req.EvidenceType,
				"evidence_id":   req.EvidenceID,
				"citation":      req.Citation,
			},
		})
	})

	return toEvidenceLinkResponse(link), nil
}

func (s *Service) RemoveEvidence(ctx context.Context, claims *auth.Claims, caseID, findingID uuid.UUID, linkIDStr string) error {
	row, err := s.repo.GetFindingByCaseAndID(ctx, caseID, findingID)
	if err != nil {
		return ErrNotFound
	}

	linkID, err := uuid.Parse(linkIDStr)
	if err != nil {
		return ErrBadRequest
	}

	// Only creator or manager+ can remove evidence
	if row.CreatedBy != claims.UserID && claims.Role != db.UserRoleManager &&
		claims.Role != db.UserRoleAdmin && claims.Role != db.UserRoleSuperAdmin {
		return ErrForbidden
	}

	if err := s.repo.RemoveEvidence(ctx, linkID, findingID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	actorID := claims.UserID
	worker.TrySubmit(func() {
		s.emitter.Emit(context.Background(), timeline.Event{ //nolint:errcheck
			CaseID:     caseID,
			EventType:  evtFindingEvidenceRemoved,
			Source:     db.TimelineEventSourceUser,
			ActorID:    &actorID,
			OccurredAt: time.Now(),
			Payload: map[string]any{
				"finding_id": findingID.String(),
				"link_id":    linkIDStr,
			},
		})
	})

	return nil
}

func (s *Service) ListEvidence(ctx context.Context, caseID, findingID uuid.UUID) ([]EvidenceLinkResponse, error) {
	_, err := s.repo.GetFindingByCaseAndID(ctx, caseID, findingID)
	if err != nil {
		return nil, ErrNotFound
	}

	links, err := s.repo.ListEvidence(ctx, findingID)
	if err != nil {
		return nil, err
	}

	out := make([]EvidenceLinkResponse, len(links))
	for i, l := range links {
		out[i] = toEvidenceLinkResponse(l)
	}
	return out, nil
}

// Summary returns quick counts for the case detail header.
func (s *Service) Summary(ctx context.Context, caseID uuid.UUID) (FindingsSummary, error) {
	return s.repo.Summary(ctx, caseID)
}

// ApprovedFindings returns all approved findings (used by closure pre-check and STR).
func (s *Service) ApprovedFindings(ctx context.Context, caseID uuid.UUID) ([]FindingResponse, error) {
	rows, err := s.repo.ApprovedFindings(ctx, caseID)
	if err != nil {
		return nil, err
	}
	out := make([]FindingResponse, len(rows))
	for i, r := range rows {
		out[i] = toResponse(r)
	}
	return out, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (s *Service) emitStatusChange(ctx context.Context, caseID, findingID, actorID uuid.UUID, from, to, title string) {
	worker.TrySubmit(func() {
		s.emitter.Emit(context.Background(), timeline.Event{ //nolint:errcheck
			CaseID:     caseID,
			EventType:  evtFindingStatusChanged,
			Source:     db.TimelineEventSourceUser,
			ActorID:    &actorID,
			OccurredAt: time.Now(),
			Payload: map[string]any{
				"finding_id": findingID.String(),
				"title":      title,
				"from":       from,
				"to":         to,
			},
		})
	})
}

func toResponse(row findingRow) FindingResponse {
	resp := FindingResponse{
		ID:              row.ID.String(),
		CaseID:          row.CaseID.String(),
		Title:           row.Title,
		Description:     row.Description,
		Category:        row.Category,
		Severity:        row.Severity,
		Status:          row.Status,
		NarrativeWeight: row.NarrativeWeight,
		AIGenerated:     row.AIGenerated,
		EvidenceCount:   row.EvidenceCount,
		CreatedBy:       UserRef{ID: row.CreatedBy, FullName: row.CreatedByName},
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		EvidenceLinks:   []EvidenceLinkResponse{},
	}

	if row.ConfidenceScore.Valid {
		var f float64
		_, _ = fmt.Sscanf(row.ConfidenceScore.String, "%f", &f)
		resp.ConfidenceScore = &f
	}
	if row.SourceAILogID.Valid {
		s := row.SourceAILogID.UUID.String()
		resp.SourceAILogID = &s
	}
	if row.ReviewedBy.Valid {
		name := ""
		if row.ReviewedByName.Valid {
			name = row.ReviewedByName.String
		}
		resp.ReviewedBy = &UserRef{ID: row.ReviewedBy.UUID, FullName: name}
	}
	if row.ApprovedBy.Valid {
		name := ""
		if row.ApprovedByName.Valid {
			name = row.ApprovedByName.String
		}
		resp.ApprovedBy = &UserRef{ID: row.ApprovedBy.UUID, FullName: name}
	}
	if row.ReviewedAt.Valid {
		resp.ReviewedAt = &row.ReviewedAt.Time
	}
	if row.ApprovedAt.Valid {
		resp.ApprovedAt = &row.ApprovedAt.Time
	}
	return resp
}

func toEvidenceLinkResponse(link evidenceLinkRow) EvidenceLinkResponse {
	resp := EvidenceLinkResponse{
		ID:            link.ID.String(),
		FindingID:     link.FindingID.String(),
		EvidenceType:  link.EvidenceType,
		EvidenceID:    link.EvidenceID.String(),
		Citation:      link.Citation,
		RelevanceNote: link.RelevanceNote,
		LinkedBy:      link.LinkedBy.String(),
		LinkedAt:      link.LinkedAt,
	}
	if link.PageNumber.Valid {
		n := link.PageNumber.Int32
		resp.PageNumber = &n
	}
	if link.ChunkID.Valid {
		s := link.ChunkID.UUID.String()
		resp.ChunkID = &s
	}
	return resp
}

