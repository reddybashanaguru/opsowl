//go:build ignore

// reset.go — wipes all case/investigation data, keeps users.
// Usage: DATABASE_URL=... go run ./cmd/seed/reset.go
package main

import (
	"context"
	"fmt"
	"os"

	pgxstdlib "github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://opsowl:opsowldev@localhost:5433/opsowl?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect error:", err)
		os.Exit(1)
	}
	defer pool.Close()
	db := pgxstdlib.OpenDBFromPool(pool)
	defer db.Close()

	stmts := []string{
		// Case-related
		"TRUNCATE finding_evidence_links, case_findings, case_narratives, case_closures, case_ai_insights, case_typologies, comm_ai_extractions, communication_logs, case_transactions, mentions, comments, notes, ai_logs, document_chunks, documents, task_comments, tasks, timeline_events, case_activities, case_watchers, case_assignments, case_subjects, str_filings CASCADE",
		"TRUNCATE cases CASCADE",
		"ALTER SEQUENCE case_number_seq RESTART WITH 1",
		// Alerts
		"TRUNCATE incoming_alerts, alert_rules, alert_sources CASCADE",
		// Subjects & entities
		"TRUNCATE entity_relationships CASCADE",
		// CSV imports
		"TRUNCATE csv_import_rows, csv_imports CASCADE",
		// Audit + AI
		"TRUNCATE audit_logs CASCADE",
		"SELECT setval('audit_log_seq', 1, false)",
		// Notifications
		"TRUNCATE notifications CASCADE",
		// Webhooks
		"TRUNCATE outbound_webhook_deliveries CASCADE",
		// Idempotency keys
		"TRUNCATE idempotency_keys CASCADE",
	}

	fmt.Println("🧹 Resetting OpsOwl to clean state (keeping users)...")
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			fmt.Fprintf(os.Stderr, "  ⚠  %s\n     error: %v\n", s[:min(60, len(s))], err)
		}
	}
	fmt.Println("✅ Done. Users preserved. Database is clean and ready.")
	fmt.Println("   Login: admin@opsowl.dev / OpsOwl@2024")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
