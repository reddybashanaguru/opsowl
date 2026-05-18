package sla

import (
	"context"
	"time"

	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"go.uber.org/zap"
)

// Scheduler periodically marks cases as SLA-breached based on sla_policies thresholds.
// It runs immediately on Start, then every interval thereafter.
type Scheduler struct {
	queries  *db.Queries
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

func (s *Scheduler) sweep(_ context.Context) {
	// Use an independent context for the DB call so cancelling the scheduler
	// lifecycle does not abort an in-progress sweep mid-query.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	n, err := s.queries.MarkBreachedCases(ctx)
	if err != nil {
		s.logger.Error("sla breach sweep failed", zap.Error(err))
		return
	}
	if n > 0 {
		s.logger.Info("sla breach sweep", zap.Int64("newly_breached", n))
	}
}
