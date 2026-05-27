package scheduled_reports

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// Scheduler ticks every minute, finds due report schedules, and dispatches emails.
type Scheduler struct {
	svc    *Service
	logger *zap.Logger
}

func NewScheduler(svc *Service, logger *zap.Logger) *Scheduler {
	return &Scheduler{svc: svc, logger: logger}
}

func (sc *Scheduler) Start(ctx context.Context) {
	sc.sweep(ctx)
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			sc.sweep(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (sc *Scheduler) sweep(ctx context.Context) {
	sweepCtx, cancel := context.WithTimeout(context.Background(), 55*time.Second)
	defer cancel()

	due, err := sc.svc.q.GetDueSchedules(sweepCtx)
	if err != nil {
		sc.logger.Error("report scheduler: failed to fetch due schedules", zap.Error(err))
		return
	}

	for _, row := range due {
		row := row
		go func() {
			if err := sc.svc.sendReport(sweepCtx, row); err != nil {
				sc.logger.Error("report scheduler: failed to send report",
					zap.String("schedule_id", row.ID.String()),
					zap.String("name", row.Name),
					zap.Error(err),
				)
			} else {
				sc.logger.Info("report scheduler: sent",
					zap.String("name", row.Name),
					zap.Strings("recipients", row.Recipients),
				)
			}

			// Advance next_run_at regardless of send success (avoid spam on error).
			nextRun := CalcNextRun(row.Frequency, row.SendHour, func() *int32 {
				if row.DayOfWeek.Valid {
					v := row.DayOfWeek.Int32
					return &v
				}
				return nil
			}())
			sc.svc.q.MarkScheduleSent(sweepCtx, row.ID, nextRun) //nolint
		}()
	}
}
