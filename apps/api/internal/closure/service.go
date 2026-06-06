package closure

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

// Timeline event type constants for closure (added by migration 00040_case_closures.sql).
const (
	evtClosureSubmitted = db.TimelineEventType("case_closure_submitted")
	evtClosureApproved  = db.TimelineEventType("case_closure_approved")
	evtClosureRejected  = db.TimelineEventType("case_closure_rejected")
	evtCaseReopened     = db.TimelineEventType("case_reopened")
)

var (
	ErrNotFound              = errors.New("closure not found")
	ErrForbidden             = errors.New("forbidden")
	ErrBadRequest            = errors.New("bad request")
	ErrInvalidTransition     = errors.New("invalid status transition")
	ErrChecklistFailed       = errors.New("pre-closure checklist has blocking failures")
	ErrSummaryTooShort       = errors.New("summary must be at least 100 characters")
	ErrSARRejectionRequired  = errors.New("sar_rejection_reason required when high-risk case is not filed as SAR")
	ErrSamePersonApproval    = errors.New("submitter and approver must be different people")
)

// validDispositions is the allowed set for closure disposition.
var validDispositions = map[string]bool{
	"sar_filed": true, "no_sar_warranted": true, "false_positive": true,
	"closed_monitoring": true, "escalated_law_enforcement": true, "referred_internal": true,
}

type Service struct {
	repo    *Repository
	emitter timeline.Emitter
}

func NewService(repo *Repository, emitter timeline.Emitter) *Service {
	return &Service{repo: repo, emitter: emitter}
}

// ── Checklist ─────────────────────────────────────────────────────────────────

// Checklist evaluates all pre-closure conditions and returns a structured result.
// Blocking failures must be resolved before a closure can be submitted.
func (s *Service) Checklist(ctx context.Context, caseID uuid.UUID) (ChecklistResult, error) {
	var checks []ChecklistItem
	canProceed := true

	// Check 1: At least one approved finding
	approvedFindings, _ := s.repo.CountApprovedFindings(ctx, caseID)
	findingsCheck := ChecklistItem{
		Name:     "approved_finding_required",
		Passed:   approvedFindings > 0,
		Blocking: true,
		Detail:   fmt.Sprintf("%d approved finding(s) found", approvedFindings),
	}
	if !findingsCheck.Passed {
		findingsCheck.Detail = "No approved findings — investigate and create at least one finding before closing"
		canProceed = false
	}
	checks = append(checks, findingsCheck)

	// Check 2: No open tasks
	openTasks, _ := s.repo.CountOpenTasks(ctx, caseID)
	tasksCheck := ChecklistItem{
		Name:     "no_open_tasks",
		Passed:   openTasks == 0,
		Blocking: true,
		Detail:   fmt.Sprintf("%d open task(s) remaining", openTasks),
	}
	if !tasksCheck.Passed {
		tasksCheck.Detail = fmt.Sprintf("%d task(s) still open — complete or cancel them before closing", openTasks)
		canProceed = false
	}
	checks = append(checks, tasksCheck)

	// Check 3: Risk level snapshot (non-blocking — informational)
	riskLevel, _ := s.repo.GetCaseRiskLevel(ctx, caseID)
	checks = append(checks, ChecklistItem{
		Name:   "risk_level_snapshot",
		Passed: true,
		Blocking: false,
		Detail: fmt.Sprintf("Risk level at time of closure: %s", riskLevel),
	})

	return ChecklistResult{CanProceed: canProceed, Checks: checks}, nil
}

// ── CRUD ──────────────────────────────────────────────────────────────────────

// Create creates a closure draft after validating the checklist.
func (s *Service) Create(ctx context.Context, claims *auth.Claims, caseID uuid.UUID, req CreateClosureRequest) (ClosureResponse, error) {
	if strings.TrimSpace(req.Summary) == "" {
		return ClosureResponse{}, ErrBadRequest
	}
	if len(strings.TrimSpace(req.Summary)) < 100 {
		return ClosureResponse{}, ErrSummaryTooShort
	}
	if req.Disposition != "" && !validDispositions[req.Disposition] {
		return ClosureResponse{}, ErrBadRequest
	}

	// Checklist: blocking checks must pass
	checklist, err := s.Checklist(ctx, caseID)
	if err != nil {
		return ClosureResponse{}, err
	}
	if !checklist.CanProceed {
		return ClosureResponse{}, ErrChecklistFailed
	}

	// If risk is high and disposition is not SAR_FILED, require explicit SAR consideration
	riskLevel, _ := s.repo.GetCaseRiskLevel(ctx, caseID)
	if riskLevel == "high" || riskLevel == "critical" {
		if req.Disposition != "sar_filed" && !req.SARConsidered {
			return ClosureResponse{}, ErrSARRejectionRequired
		}
		if req.Disposition != "sar_filed" && req.SARConsidered && strings.TrimSpace(req.SARRejectionReason) == "" {
			return ClosureResponse{}, ErrSARRejectionRequired
		}
	}

	// Set the risk rating at close from the AI snapshot if not provided
	if req.RiskRatingAtClose == "" {
		req.RiskRatingAtClose = riskLevel
	}

	row, err := s.repo.Create(ctx, caseID, claims.UserID, req)
	if err != nil {
		return ClosureResponse{}, err
	}
	return toResponse(row), nil
}

func (s *Service) Get(ctx context.Context, caseID uuid.UUID) (ClosureResponse, error) {
	row, err := s.repo.GetByCaseID(ctx, caseID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ClosureResponse{}, ErrNotFound
		}
		return ClosureResponse{}, err
	}
	return toResponse(row), nil
}

func (s *Service) Update(ctx context.Context, claims *auth.Claims, caseID, closureID uuid.UUID, req UpdateClosureRequest) (ClosureResponse, error) {
	row, err := s.repo.GetByID(ctx, closureID)
	if err != nil {
		return ClosureResponse{}, ErrNotFound
	}
	if row.CaseID != caseID {
		return ClosureResponse{}, ErrNotFound
	}
	if row.CreatedBy != claims.UserID {
		if claims.Role != db.UserRoleManager && claims.Role != db.UserRoleAdmin && claims.Role != db.UserRoleSuperAdmin {
			return ClosureResponse{}, ErrForbidden
		}
	}
	if row.Status != "draft" {
		return ClosureResponse{}, ErrInvalidTransition
	}

	if req.Summary != nil && len(strings.TrimSpace(*req.Summary)) < 100 {
		return ClosureResponse{}, ErrSummaryTooShort
	}
	if req.Disposition != nil && !validDispositions[*req.Disposition] {
		return ClosureResponse{}, ErrBadRequest
	}

	if err := s.repo.Update(ctx, closureID, req); err != nil {
		return ClosureResponse{}, err
	}
	updated, err := s.repo.GetByID(ctx, closureID)
	if err != nil {
		return ClosureResponse{}, err
	}
	return toResponse(updated), nil
}

// ── Status transitions ────────────────────────────────────────────────────────

// Submit moves a closure draft to submitted status for manager review.
func (s *Service) Submit(ctx context.Context, claims *auth.Claims, caseID, closureID uuid.UUID) (ClosureResponse, error) {
	row, err := s.repo.GetByID(ctx, closureID)
	if err != nil || row.CaseID != caseID {
		return ClosureResponse{}, ErrNotFound
	}
	if row.CreatedBy != claims.UserID {
		if claims.Role != db.UserRoleManager && claims.Role != db.UserRoleAdmin && claims.Role != db.UserRoleSuperAdmin {
			return ClosureResponse{}, ErrForbidden
		}
	}
	if row.Status != "draft" {
		return ClosureResponse{}, ErrInvalidTransition
	}

	if err := s.repo.UpdateStatus(ctx, closureID, claims.UserID, "submitted", ""); err != nil {
		return ClosureResponse{}, err
	}

	actorID := claims.UserID
	worker.TrySubmit(func() {
		s.emitter.Emit(context.Background(), timeline.Event{ //nolint:errcheck
			CaseID:     caseID,
			EventType:  evtClosureSubmitted,
			Source:     db.TimelineEventSourceUser,
			ActorID:    &actorID,
			OccurredAt: time.Now(),
			Payload: map[string]any{
				"closure_id":  closureID.String(),
				"disposition": row.Disposition,
			},
		})
	})

	updated, _ := s.repo.GetByID(ctx, closureID)
	return toResponse(updated), nil
}

// Approve approves a submitted closure, transitions the case to 'closed',
// and creates the authoritative closure record stamped on the case.
func (s *Service) Approve(ctx context.Context, claims *auth.Claims, caseID, closureID uuid.UUID) (ClosureResponse, error) {
	if claims.Role != db.UserRoleManager && claims.Role != db.UserRoleAdmin && claims.Role != db.UserRoleSuperAdmin {
		return ClosureResponse{}, ErrForbidden
	}

	row, err := s.repo.GetByID(ctx, closureID)
	if err != nil || row.CaseID != caseID {
		return ClosureResponse{}, ErrNotFound
	}
	if row.Status != "submitted" {
		return ClosureResponse{}, ErrInvalidTransition
	}
	// Maker-checker: approver cannot be the submitter
	if row.CreatedBy == claims.UserID {
		return ClosureResponse{}, ErrSamePersonApproval
	}

	if err := s.repo.UpdateStatus(ctx, closureID, claims.UserID, "approved", ""); err != nil {
		return ClosureResponse{}, err
	}
	// Stamp the case as closed
	if err := s.repo.CloseCaseRecord(ctx, caseID, closureID, claims.UserID); err != nil {
		return ClosureResponse{}, err
	}

	actorID := claims.UserID
	worker.TrySubmit(func() {
		s.emitter.Emit(context.Background(), timeline.Event{ //nolint:errcheck
			CaseID:     caseID,
			EventType:  evtClosureApproved,
			Source:     db.TimelineEventSourceUser,
			ActorID:    &actorID,
			OccurredAt: time.Now(),
			Payload: map[string]any{
				"closure_id":  closureID.String(),
				"disposition": row.Disposition,
				"summary_len": len(row.Summary),
			},
		})
	})

	updated, _ := s.repo.GetByID(ctx, closureID)
	return toResponse(updated), nil
}

// Reject returns a submitted closure to draft status for the analyst to revise.
func (s *Service) Reject(ctx context.Context, claims *auth.Claims, caseID, closureID uuid.UUID, reason string) (ClosureResponse, error) {
	if claims.Role != db.UserRoleManager && claims.Role != db.UserRoleAdmin && claims.Role != db.UserRoleSuperAdmin {
		return ClosureResponse{}, ErrForbidden
	}

	row, err := s.repo.GetByID(ctx, closureID)
	if err != nil || row.CaseID != caseID {
		return ClosureResponse{}, ErrNotFound
	}
	if row.Status != "submitted" {
		return ClosureResponse{}, ErrInvalidTransition
	}

	if err := s.repo.UpdateStatus(ctx, closureID, claims.UserID, "rejected", reason); err != nil {
		return ClosureResponse{}, err
	}

	actorID := claims.UserID
	worker.TrySubmit(func() {
		s.emitter.Emit(context.Background(), timeline.Event{ //nolint:errcheck
			CaseID:     caseID,
			EventType:  evtClosureRejected,
			Source:     db.TimelineEventSourceUser,
			ActorID:    &actorID,
			OccurredAt: time.Now(),
			Payload: map[string]any{
				"closure_id": closureID.String(),
				"reason":     reason,
			},
		})
	})

	// Revert closure back to draft so analyst can fix it
	if err := s.repo.UpdateStatus(ctx, closureID, claims.UserID, "draft", ""); err != nil {
		return ClosureResponse{}, err
	}

	updated, _ := s.repo.GetByID(ctx, closureID)
	return toResponse(updated), nil
}

// Reopen voids the current approved closure and reopens the case.
// Requires supervisor+ role. Preserves the voided closure record for audit.
func (s *Service) Reopen(ctx context.Context, claims *auth.Claims, caseID uuid.UUID, reason string) error {
	if claims.Role != db.UserRoleAdmin && claims.Role != db.UserRoleSuperAdmin {
		return ErrForbidden
	}
	if strings.TrimSpace(reason) == "" {
		return ErrBadRequest
	}

	row, err := s.repo.GetByCaseID(ctx, caseID)
	if err != nil {
		return ErrNotFound
	}

	if err := s.repo.Void(ctx, row.ID, claims.UserID, reason); err != nil {
		return err
	}
	if err := s.repo.ReopenCase(ctx, caseID); err != nil {
		return err
	}

	actorID := claims.UserID
	worker.TrySubmit(func() {
		s.emitter.Emit(context.Background(), timeline.Event{ //nolint:errcheck
			CaseID:     caseID,
			EventType:  evtCaseReopened,
			Source:     db.TimelineEventSourceUser,
			ActorID:    &actorID,
			OccurredAt: time.Now(),
			Payload: map[string]any{
				"voided_closure_id": row.ID.String(),
				"reason":            reason,
			},
		})
	})
	return nil
}

// ── Response mapper ───────────────────────────────────────────────────────────

func toResponse(row closureRow) ClosureResponse {
	resp := ClosureResponse{
		ID:          row.ID.String(),
		CaseID:      row.CaseID.String(),
		Disposition: row.Disposition,
		Summary:     row.Summary,
		ReasonCode:  row.ReasonCode,
		FiledSTR:    row.FiledSTR,
		Status:      row.Status,
		SARConsidered: row.SARConsidered,
		CreatedBy:   UserRef{ID: row.CreatedBy, FullName: row.CreatedByName},
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
	if row.STRID.Valid {
		s := row.STRID.UUID.String()
		resp.STRFileID = &s
	}
	if row.RiskRatingAtClose.Valid {
		resp.RiskRatingAtClose = row.RiskRatingAtClose.String
	}
	if row.SARRejectionReason.Valid {
		resp.SARRejectionReason = row.SARRejectionReason.String
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
	if row.SubmittedAt.Valid {
		resp.SubmittedAt = &row.SubmittedAt.Time
	}
	if row.ApprovedAt.Valid {
		resp.ApprovedAt = &row.ApprovedAt.Time
	}
	if row.VoidedAt.Valid {
		resp.VoidedAt = &row.VoidedAt.Time
	}
	if row.VoidedReason.Valid {
		resp.VoidedReason = row.VoidedReason.String
	}
	return resp
}

// suppress unused import warning
var _ = fmt.Sprintf
