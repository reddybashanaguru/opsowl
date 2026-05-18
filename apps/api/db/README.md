# Database — Production-Grade Structure

```
db/
├── ddl/                          # Schema reference (CREATE TABLE, INDEX, TYPE)
│   ├── 001_extensions_and_enums.sql
│   └── 002_core_tables.sql
│
├── dml/
│   ├── queries/                  # sqlc input queries (SELECT, INSERT, UPDATE, DELETE)
│   │   ├── auth.sql
│   │   ├── cases.sql
│   │   └── ...
│   └── seeds/                   # Initial / reference data (idempotent INSERTs)
│       ├── sla_policies.sql
│       └── config_defaults.sql
│
└── migrations/
    ├── incremental/             # Forward migrations (goose Up + Down in each file)
    │   ├── 00001_init.sql
    │   ├── 00002_auth.sql
    │   └── ...
    └── rollback/                # Extracted Down-only scripts for emergency rollbacks
        ├── 00001_rollback_init.sql
        └── ...
```

## Conventions

| Folder | Purpose | Runnable? |
|--------|---------|-----------|
| `ddl/` | Schema reference — the current state of all tables/indexes | No — reference only |
| `dml/queries/` | Application queries read by sqlc | No — sqlc input only |
| `dml/seeds/` | Idempotent seed data | Yes — `psql -f <file>` after migration |
| `migrations/incremental/` | Goose migration files (Up + Down in one file) | Yes — `goose up` |
| `migrations/rollback/` | Standalone Down-only scripts for targeted rollback | Yes — emergency only |

## Running Migrations

```bash
# Apply all pending migrations
goose -dir db/migrations/incremental postgres "$DATABASE_URL" up

# Roll back one migration
goose -dir db/migrations/incremental postgres "$DATABASE_URL" down

# Emergency rollback of a specific version (standalone script)
psql "$DATABASE_URL" -f db/migrations/rollback/00012_rollback_config.sql
```

## Adding a New Migration

1. Create `db/migrations/incremental/NNNNN_description.sql` with `-- +goose Up` and `-- +goose Down` sections
2. Extract the Down section into `db/migrations/rollback/NNNNN_rollback_description.sql`
3. Update `db/ddl/` if adding new tables or indexes
4. Add application queries to `db/dml/queries/` and run `sqlc generate`
5. Add seed data to `db/dml/seeds/` if the migration requires initial data

## sqlc

The `sqlc.yaml` in the project root points to `db/dml/queries/` for query input.
Generated code lives in `internal/db/generated/` — do not edit manually.
