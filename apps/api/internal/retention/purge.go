// Package retention implements the DPDP Act 2023-compliant data purge scheduler.
// It runs once per day, reads retention_policies from the DB, and deletes
// records older than the configured retention_days for each enabled resource type.
package retention

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.uber.org/zap"

	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

// Scheduler runs daily retention purges.
type Scheduler struct {
	sqlDB  *sql.DB
	logger *zap.Logger
}

func NewScheduler(sqlDB *sql.DB, logger *zap.Logger) *Scheduler {
	return &Scheduler{sqlDB: sqlDB, logger: logger}
}

// Start begins the daily purge loop. Call in a goroutine.
func (s *Scheduler) Start(ctx context.Context) {
	// Run once on startup (catches any missed purges), then every 24h.
	s.runPurge(ctx)
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runPurge(ctx)
		}
	}
}

func (s *Scheduler) runPurge(ctx context.Context) {
	policies, err := s.fetchPolicies(ctx)
	if err != nil {
		s.logger.Error("retention: could not load policies", zap.Error(err))
		return
	}

	for _, p := range policies {
		if !p.PurgeEnabled {
			continue
		}
		cutoff := time.Now().AddDate(0, 0, -p.RetentionDays)
		deleted, err := s.purge(ctx, p.ResourceType, cutoff)
		if err != nil {
			s.logger.Error("retention purge failed", zap.String("resource", p.ResourceType), zap.Error(err))
			continue
		}
		if deleted > 0 {
			s.logger.Info("retention purge complete",
				zap.String("resource", p.ResourceType),
				zap.Int64("deleted", deleted),
				zap.Time("cutoff", cutoff))
		}
		_ = s.updateLastPurged(ctx, p.ResourceType)
	}
}

type retentionPolicy struct {
	ResourceType  string
	RetentionDays int
	PurgeEnabled  bool
}

func (s *Scheduler) fetchPolicies(ctx context.Context) ([]retentionPolicy, error) {
	rows, err := s.sqlDB.QueryContext(ctx, `
		SELECT resource_type, retention_days, purge_enabled
		FROM data_retention_policies
		WHERE purge_enabled = TRUE
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []retentionPolicy
	for rows.Next() {
		var p retentionPolicy
		if err := rows.Scan(&p.ResourceType, &p.RetentionDays, &p.PurgeEnabled); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// purge deletes records older than cutoff for the given resource type.
// Table mapping is explicit to prevent SQL injection via resource_type values.
func (s *Scheduler) purge(ctx context.Context, resourceType string, cutoff time.Time) (int64, error) {
	tableMap := map[string]string{
		"notifications": "DELETE FROM notifications WHERE created_at < $1",
		"sessions":      "DELETE FROM sessions WHERE last_active_at < $1",
		// audit_logs and cases are intentionally excluded from automated purge by default
	}
	query, ok := tableMap[resourceType]
	if !ok {
		return 0, fmt.Errorf("unknown resource type: %s", resourceType)
	}
	result, err := s.sqlDB.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *Scheduler) updateLastPurged(ctx context.Context, resourceType string) error {
	_, err := s.sqlDB.ExecContext(ctx,
		`UPDATE data_retention_policies SET last_purged_at = NOW() WHERE resource_type = $1`,
		resourceType)
	return err
}

// PolicyResponse is returned by the admin API.
type PolicyResponse struct {
	ResourceType  string     `json:"resource_type"`
	RetentionDays int        `json:"retention_days"`
	PurgeEnabled  bool       `json:"purge_enabled"`
	LastPurgedAt  *time.Time `json:"last_purged_at,omitempty"`
}

// ListPolicies returns all retention policies.
func ListPolicies(ctx context.Context, sqlDB *sql.DB) ([]PolicyResponse, error) {
	rows, err := sqlDB.QueryContext(ctx, `
		SELECT resource_type, retention_days, purge_enabled, last_purged_at
		FROM data_retention_policies
		ORDER BY resource_type
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PolicyResponse
	for rows.Next() {
		var p PolicyResponse
		var lastPurged sql.NullTime
		if err := rows.Scan(&p.ResourceType, &p.RetentionDays, &p.PurgeEnabled, &lastPurged); err != nil {
			return nil, err
		}
		if lastPurged.Valid {
			p.LastPurgedAt = &lastPurged.Time
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Ensure db import is used (it's used by callers).
var _ = db.Queries{}
