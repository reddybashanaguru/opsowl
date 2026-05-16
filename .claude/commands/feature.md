# /feature — Build a specific OpsOwl AI feature

Build Feature $ARGUMENTS following the exact OpsOwl AI development workflow.

## Step 1 — Read Spec and Context

First, read these files:
1. The feature spec: `docs/0$ARGUMENTS_*.html` (find the file matching feature number $ARGUMENTS)
2. `CLAUDE.md` — review all rules (API standards, DB rules, security rules, UX rules)

Report back: what this feature does, the key DB tables needed, all API endpoints, and whether Python AI service is involved.

## Step 2 — Plan (before writing any code)

State your build plan:
- New DB tables + migrations (with soft delete, TIMESTAMPTZ, indexes on every FK)
- Go module structure (`internal/<module>/`)
- API endpoints (method, path, request shape, response shape)
- Python AI service endpoints (if needed)
- Next.js pages and components
- River jobs (if async work is needed)
- SSE events emitted (if real-time is involved)
- Audit events emitted (what goes to audit_logs)
- Timeline events emitted (what goes to timeline_events)

Wait for acknowledgement before writing code.

## Step 3 — Backend (Go)

Build in this order:
1. **Migration SQL** → `apps/api/migrations/NNNN_feature_name.sql`
   - Every table: id UUID PK, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ
   - Soft deletes: deleted_at TIMESTAMPTZ nullable (NEVER omit this)
   - All timestamps: TIMESTAMPTZ (not TIMESTAMP)
   - FK constraints enforced
   - Indexes: on every FK column + every column used in WHERE/ORDER BY
   - Run: `sqlc generate` after migration

2. **Go module** → `apps/api/internal/<module>/`
   - `repository.go` — sqlc queries, no raw SQL
   - `service.go` — business logic, calls repository
   - `handler.go` — HTTP handlers, validation, response envelope `{data, meta, error}`
   - `routes.go` — Chi router setup

3. **Go tests** → `apps/api/internal/<module>/<file>_test.go`
   - Table-driven tests
   - Coverage must be >90% per package
   - Run: `go test ./...`
   - Run: `go vet ./...`

4. **Emit audit events** — every state change calls `auditService.Log(...)`
5. **Emit timeline events** — every case-related action calls `timelineService.Emit(...)`
6. **Emit notifications** — assign River notification jobs where spec says

## Step 4 — AI Service (Python) — skip if not needed for this feature

Build in `apps/ai-service/app/`:
1. Router: `routers/<feature>.py`
2. Service: `services/<feature>_service.py`
3. Tests: `tests/test_<feature>.py` — mock all LLM calls, >90% coverage
4. Run: `python -m pytest tests/`

## Step 5 — Frontend (Next.js)

Build in `apps/web/app/`:
1. Page(s): `(dashboard)/<feature>/page.tsx`
2. Components: `components/<feature>/`
3. API hooks: `hooks/use-<feature>.ts` (TanStack Query)
4. Types: shared from `packages/shared-types/`

UX rules to follow:
- Loading: skeleton screens, never spinners alone
- Errors: actionable ("Try again" or "Contact support")
- Empty states: helpful message + CTA
- Destructive actions: confirmation dialog
- AI outputs: source + confidence + "Review" button

Run: `npm run typecheck` and `npm run lint`

## Step 6 — Quality Gate

Run `/check` and fix every error before proceeding.

## Step 7 — User Perspective Test

Put yourself in the shoes of an **investigator** using this feature:
- Walk through the golden path (create, view, edit, delete if applicable)
- Test edge cases (empty state, validation errors, unauthorized access)
- Check that audit trail shows correct events
- Check that timeline shows correct events
- Verify notifications fire as expected

Describe exactly what you tested and what you found.

## Step 8 — Commit

Use this format:
```
feat(<module>): <short description>

- What was built
- Key decisions made  
- Known limitations (if any)

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
```

## Step 9 — Update CLAUDE.md

Change the feature status in the MVP table from `🔲 Not started` to `✅ Complete`.
Add an entry to `## Completed Features Log` at the bottom.
