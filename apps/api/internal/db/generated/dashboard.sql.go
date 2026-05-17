package db

import "context"

const getDashboardStats = `
SELECT
  (SELECT COUNT(*) FROM cases WHERE status NOT IN ('closed','draft') AND deleted_at IS NULL),
  (SELECT COUNT(*) FROM tasks  WHERE status IN ('todo','in_progress')),
  (SELECT COUNT(*) FROM ai_logs WHERE created_at >= CURRENT_DATE)
`

type DashboardStats struct {
	OpenCases      int64 `json:"open_cases"`
	PendingTasks   int64 `json:"pending_tasks"`
	AIQueriesToday int64 `json:"ai_queries_today"`
}

func (q *Queries) GetDashboardStats(ctx context.Context) (DashboardStats, error) {
	row := q.db.QueryRowContext(ctx, getDashboardStats)
	var s DashboardStats
	err := row.Scan(&s.OpenCases, &s.PendingTasks, &s.AIQueriesToday)
	return s, err
}
