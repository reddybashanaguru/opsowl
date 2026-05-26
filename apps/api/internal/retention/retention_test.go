package retention_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func testDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	ctx := context.Background()

	pgc, err := postgres.Run(ctx,
		"pgvector/pgvector:pg17",
		postgres.WithDatabase("opsowl_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}

	connStr, err := pgc.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("get connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}

	sqlDB := stdlib.OpenDBFromPool(pool)

	goose.SetBaseFS(nil)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("goose dialect: %v", err)
	}
	if err := goose.Up(sqlDB, "../../db/migrations/incremental"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	return sqlDB, func() {
		pool.Close()
		_ = pgc.Terminate(ctx)
	}
}

// ── C4: PMLA retention floor tests ───────────────────────────────────────────

func TestPMLARetentionFloor_CorrectValuesAfterMigration(t *testing.T) {
	db, teardown := testDB(t)
	defer teardown()

	rows, err := db.QueryContext(context.Background(), `
		SELECT resource_type, retention_days, purge_enabled
		FROM data_retention_policies
		WHERE resource_type IN ('audit_logs','cases','documents','ai_logs','str_filings')
		ORDER BY resource_type
	`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()

	want := map[string]struct {
		days         int
		purgeEnabled bool
	}{
		"ai_logs":    {3650, false},
		"audit_logs": {3650, false},
		"cases":      {3650, false},
		"documents":  {3650, false},
		"str_filings": {3650, false},
	}

	count := 0
	for rows.Next() {
		var resourceType string
		var days int
		var purgeEnabled bool
		if err := rows.Scan(&resourceType, &days, &purgeEnabled); err != nil {
			t.Fatalf("scan: %v", err)
		}
		w, ok := want[resourceType]
		if !ok {
			t.Errorf("unexpected resource_type: %s", resourceType)
			continue
		}
		if days != w.days {
			t.Errorf("resource %s: got retention_days=%d, want %d", resourceType, days, w.days)
		}
		if purgeEnabled != w.purgeEnabled {
			t.Errorf("resource %s: got purge_enabled=%v, want %v", resourceType, purgeEnabled, w.purgeEnabled)
		}
		count++
	}
	if count != 5 {
		t.Errorf("expected 5 PMLA resource rows, got %d", count)
	}
}

func TestPMLARetentionFloor_TriggerBlocksSubFloorUpdate(t *testing.T) {
	db, teardown := testDB(t)
	defer teardown()

	// Attempting to lower audit_logs below 3650 must fail.
	_, err := db.ExecContext(context.Background(), `
		UPDATE data_retention_policies SET retention_days = 365 WHERE resource_type = 'audit_logs'
	`)
	if err == nil {
		t.Fatal("expected trigger error when lowering audit_logs below 3650, got nil")
	}
}

func TestPMLARetentionFloor_TriggerBlocksStrFilingsSubFloor(t *testing.T) {
	db, teardown := testDB(t)
	defer teardown()

	_, err := db.ExecContext(context.Background(), `
		UPDATE data_retention_policies SET retention_days = 2555 WHERE resource_type = 'str_filings'
	`)
	if err == nil {
		t.Fatal("expected trigger error when lowering str_filings below 3650, got nil")
	}
}

func TestPMLARetentionFloor_TriggerAllowsAboveFloor(t *testing.T) {
	db, teardown := testDB(t)
	defer teardown()

	// 4000 days > 3650; trigger must allow it.
	_, err := db.ExecContext(context.Background(), `
		UPDATE data_retention_policies SET retention_days = 4000 WHERE resource_type = 'audit_logs'
	`)
	if err != nil {
		t.Fatalf("unexpected error raising retention_days above floor: %v", err)
	}
}

func TestPMLARetentionFloor_TriggerAllowsNonPMLAResources(t *testing.T) {
	db, teardown := testDB(t)
	defer teardown()

	// notifications and sessions have no PMLA floor; can be set to any value.
	_, err := db.ExecContext(context.Background(), `
		UPDATE data_retention_policies SET retention_days = 1 WHERE resource_type = 'notifications'
	`)
	if err != nil {
		t.Fatalf("trigger must not block non-PMLA resource type: %v", err)
	}
}

func TestPMLARetentionFloor_TriggerAllowsExactFloor(t *testing.T) {
	db, teardown := testDB(t)
	defer teardown()

	// Exactly 3650 is the minimum; trigger must NOT reject it.
	_, err := db.ExecContext(context.Background(), `
		UPDATE data_retention_policies SET retention_days = 3650 WHERE resource_type = 'audit_logs'
	`)
	if err != nil {
		t.Fatalf("trigger must allow retention_days = 3650 (floor boundary): %v", err)
	}
}

func TestPMLARetentionFloor_TriggerBlocksInsertBelowFloor(t *testing.T) {
	db, teardown := testDB(t)
	defer teardown()

	_, err := db.ExecContext(context.Background(), `
		INSERT INTO data_retention_policies (resource_type, retention_days, purge_enabled)
		VALUES ('str_filings', 100, FALSE)
		ON CONFLICT (resource_type) DO UPDATE SET retention_days = 100
	`)
	if err == nil {
		t.Fatal("expected trigger error on INSERT/UPSERT of str_filings with retention_days < 3650")
	}
}
