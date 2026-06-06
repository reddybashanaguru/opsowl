package sla

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/notifications"
	"go.uber.org/zap"
)

// Scheduler periodically marks cases as SLA-breached based on sla_policies thresholds.
// It runs immediately on Start, then every interval thereafter.
type Scheduler struct {
	queries  *db.Queries
	rawDB    *sql.DB
	notifSvc *notifications.Service
	logger   *zap.Logger
	interval time.Duration
}

func NewScheduler(queries *db.Queries, logger *zap.Logger) *Scheduler {
	return &Scheduler{
		queries:  queries,
		logger:   logger,
		interval: 5 * time.Minute,
	}
}

// WithNotifications injects the notification service so the scheduler can push
// SLA breach alerts to case assignees.
func (s *Scheduler) WithNotifications(rawDB *sql.DB, svc *notifications.Service) *Scheduler {
	s.rawDB = rawDB
	s.notifSvc = svc
	return s
}

func (s *Scheduler) Start(ctx context.Context) {
	s.sweep(ctx)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.sweep(ctx)
		case <-ctx.Done():
			return
		}
	}
}

type breachCandidate struct {
	caseID     uuid.UUID
	caseNumber string
	title      string
	assigneeID *uuid.UUID
}

// fetchBreachCandidates returns cases that are about to be marked breached.
// We query BEFORE the UPDATE so we know exactly which cases & assignees to notify.
func (s *Scheduler) fetchBreachCandidates(ctx context.Context) []breachCandidate {
	if s.rawDB == nil {
		return nil
	}
	const q = `
		SELECT c.id, c.case_number, c.title, c.assigned_to
		FROM cases c
		JOIN sla_policies sp ON sp.priority = c.priority::text
		WHERE c.sla_breached = false
		  AND c.status NOT IN ('closed', 'draft')
		  AND c.deleted_at IS NULL
		  AND c.created_at + make_interval(hours => sp.breach_hours) <= NOW()`
	rows, err := s.rawDB.QueryContext(ctx, q)
	if err != nil {
		s.logger.Warn("sla breach candidate query failed", zap.Error(err))
		return nil
	}
	defer rows.Close()
	var out []breachCandidate
	for rows.Next() {
		var c breachCandidate
		var assigneeRaw sql.NullString
		if err := rows.Scan(&c.caseID, &c.caseNumber, &c.title, &assigneeRaw); err != nil {
			continue
		}
		if assigneeRaw.Valid {
			id, err := uuid.Parse(assigneeRaw.String)
			if err == nil {
				c.assigneeID = &id
			}
		}
		out = append(out, c)
	}
	return out
}

func (s *Scheduler) sweep(_ context.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	candidates := s.fetchBreachCandidates(ctx)

	n, err := s.queries.MarkBreachedCases(ctx)
	if err != nil {
		s.logger.Error("sla breach sweep failed", zap.Error(err))
		return
	}
	if n > 0 {
		s.logger.Info("sla breach sweep", zap.Int64("newly_breached", n))
	}

	if s.notifSvc == nil || len(candidates) == 0 {
		return
	}
	for _, c := range candidates {
		if c.assigneeID == nil {
			continue
		}
		s.notifSvc.NotifyAsync(ctx, notifications.NotifyInput{
			RecipientID: *c.assigneeID,
			Type:        db.NotificationTypeCaseSlaBreached,
			Priority:    db.NotificationPriorityHigh,
			Title:       fmt.Sprintf("SLA Breached: %s", c.caseNumber),
			Message:     fmt.Sprintf(`"%s" has exceeded its SLA deadline and requires immediate attention.`, c.title),
			LinkURL:     fmt.Sprintf("/cases/%s", c.caseID),
			CaseID:      &c.caseID,
		})
	}
}
