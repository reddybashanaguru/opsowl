package db

import "context"

const getDashboardStats = `
SELECT
  (SELECT COUNT(*) FROM cases WHERE status NOT IN ('closed','draft') AND deleted_at IS NULL),
  (SELECT COUNT(*) FROM tasks  WHERE status IN ('todo','in_progress')),
  (SELECT COUNT(*) FROM ai_logs WHERE created_at >= CURRENT_DATE),
  (SELECT COUNT(*) FROM cases WHERE status = 'open'           AND deleted_at IS NULL),
  (SELECT COUNT(*) FROM cases WHERE status = 'in_progress'    AND deleted_at IS NULL),
  (SELECT COUNT(*) FROM cases WHERE status = 'pending_review' AND deleted_at IS NULL),
  (SELECT COUNT(*) FROM cases WHERE status = 'escalated'      AND deleted_at IS NULL),
  (SELECT COUNT(*) FROM cases WHERE status = 'closed'         AND deleted_at IS NULL),
  (SELECT COUNT(*) FROM cases WHERE priority = 'critical' AND status != 'closed' AND deleted_at IS NULL),
  (SELECT COUNT(*) FROM cases WHERE priority = 'high'     AND status != 'closed' AND deleted_at IS NULL),
  (SELECT COUNT(*) FROM cases WHERE priority = 'medium'   AND status != 'closed' AND deleted_at IS NULL),
  (SELECT COUNT(*) FROM cases WHERE priority = 'low'      AND status != 'closed' AND deleted_at IS NULL),
  (SELECT COUNT(*) FROM cases WHERE sla_breached = true   AND status != 'closed' AND deleted_at IS NULL),
  (SELECT COUNT(*) FROM ai_logs WHERE human_review_status = 'pending')
`

type DashboardStats struct {
	OpenCases           int64 `json:"open_cases"`
	PendingTasks        int64 `json:"pending_tasks"`
	AIQueriesToday      int64 `json:"ai_queries_today"`
	StatusOpen          int64 `json:"status_open"`
	StatusInProgress    int64 `json:"status_in_progress"`
	StatusPendingReview int64 `json:"status_pending_review"`
	StatusEscalated     int64 `json:"status_escalated"`
	StatusClosed        int64 `json:"status_closed"`
	PrioCritical        int64 `json:"prio_critical"`
	PrioHigh            int64 `json:"prio_high"`
	PrioMedium          int64 `json:"prio_medium"`
	PrioLow             int64 `json:"prio_low"`
	SLABreached         int64 `json:"sla_breached"`
	PendingAIReviews    int64 `json:"pending_ai_reviews"`
}

func (q *Queries) GetDashboardStats(ctx context.Context) (DashboardStats, error) {
	row := q.db.QueryRowContext(ctx, getDashboardStats)
	var s DashboardStats
	err := row.Scan(
		&s.OpenCases, &s.PendingTasks, &s.AIQueriesToday,
		&s.StatusOpen, &s.StatusInProgress, &s.StatusPendingReview, &s.StatusEscalated, &s.StatusClosed,
		&s.PrioCritical, &s.PrioHigh, &s.PrioMedium, &s.PrioLow,
		&s.SLABreached, &s.PendingAIReviews,
	)
	return s, err
}
