# Database — Production-Grade Structure

```
db/
├── ddl/                             # Current schema reference — always up-to-date
│   ├── 001_extensions.sql           # PostgreSQL extensions (pgvector, pg_trgm)
│   ├── 002_enums.sql                # All custom ENUM types
│   ├── 003_auth_schema.sql          # users, refresh_tokens, sessions, audit_logs, sse_tickets, totp_secrets, mfa_backup_codes
│   ├── 004_cases_schema.sql         # cases, case_assignments, case_tags, case_watchers, case_activities
│   ├── 005_timeline_schema.sql      # timeline_events
│   ├── 006_tasks_schema.sql         # tasks, task_comments
│   ├── 007_documents_schema.sql     # documents (with file_sha256), document_chunks (HNSW index)
│   ├── 008_ai_schema.sql            # ai_logs (immutable), ai_prompts, case_ai_insights
│   ├── 009_notes_schema.sql         # notes, comments, mentions
│   ├── 010_notifications_schema.sql # notifications, notification_preferences
│   ├── 011_comms_schema.sql         # communication_logs, comm_ai_extractions
│   ├── 012_sla_schema.sql           # sla_policies
│   └── 013_config_schema.sql        # system_config
│
├── dml/
│   ├── queries/                     # sqlc input — SELECT/INSERT/UPDATE/DELETE per domain
│   │   ├── auth.sql                 # users, refresh_tokens, sessions, audit_logs
│   │   ├── cases.sql                # cases, activities, assignments, tags, watchers
│   │   ├── documents.sql            # documents, document_chunks
│   │   ├── ai.sql                   # ai_logs, ai_prompts, case_ai_insights
│   │   ├── timeline.sql             # timeline_events
│   │   ├── tasks.sql                # tasks, task_comments
│   │   ├── notes.sql                # notes, comments, mentions
│   │   ├── notifications.sql        # notifications, preferences
│   │   ├── phase2.sql               # comms, SLA, workload analytics
│   │   ├── search.sql               # full-text search across all entities
│   │   ├── users.sql                # user management (admin)
│   │   └── dashboard.sql            # aggregate stats for dashboard
│   └── seeds/
│       ├── sla_policies.sql         # Default SLA thresholds per priority
│       └── config_defaults.sql      # Default system_config JSONB row
│
└── migrations/
    ├── incremental/                 # Goose forward migrations (Up + Down in each file)
    │   ├── 00001_init.sql           # Extensions
    │   ├── 00002_auth.sql           # users, refresh_tokens, sessions, audit_logs
    │   ├── 00003_cases.sql          # Cases domain
    │   ├── 00004_timeline.sql       # timeline_events
    │   ├── 00005_tasks.sql          # Tasks + set_updated_at trigger
    │   ├── 00006_documents.sql      # Documents + pgvector chunks + HNSW index
    │   ├── 00007_ai.sql             # ai_logs (immutable), ai_prompts
    │   ├── 00008_notes.sql          # Notes, comments, mentions
    │   ├── 00009_notifications.sql  # Notifications + preferences
    │   ├── 00010_search.sql         # tsvector columns + GIN indexes + triggers
    │   ├── 00011_phase2.sql         # Phase 2: comms, SLA, AI insights, analytics indexes
    │   ├── 00012_config.sql         # system_config single-row pattern
    │   ├── 00013_security_hardening.sql # file_sha256, audit immutability trigger, sse_tickets
    │   └── 00014_mfa.sql            # TOTP secrets + backup codes
    └── rollback/                    # Standalone Down-only scripts for emergency rollback
        ├── 00001_rollback_init.sql
        └── ... (one file per migration)
```

## DDL Convention
DDL files represent the **current state of the schema** after all migrations.  
When you add a migration, update the corresponding DDL file on the same PR.  
Never reference DDL files to run SQL — use `goose` for all schema changes.

## Running Migrations

```bash
# Apply all pending
goose -dir db/migrations/incremental postgres "$DATABASE_URL" up

# Roll back one step
goose -dir db/migrations/incremental postgres "$DATABASE_URL" down

# Emergency rollback of a specific migration (standalone script)
psql "$DATABASE_URL" -f db/migrations/rollback/00013_rollback_security_hardening.sql
```

## Adding a New Migration

1. Create `db/migrations/incremental/NNNNN_description.sql` with `-- +goose Up` and `-- +goose Down`
2. Extract the Down section into `db/migrations/rollback/NNNNN_rollback_description.sql`
3. **Update the relevant `db/ddl/` file** to reflect new tables/columns/indexes
4. Add or update queries in `db/dml/queries/` and run `sqlc generate`
5. Add seed data to `db/dml/seeds/` if the migration requires initial data

## sqlc

`sqlc.yaml` points to `db/dml/queries/` for input. Generated code: `internal/db/generated/` (do not edit).
