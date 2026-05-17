# OpsOwl AI — Claude Code Context

## Product Name
**OpsOwl AI** — Operations Intelligence Workspace

## Product in One Line
"AI-powered investigation and operations workspace for enterprise teams."

## What This Is NOT
- Not an AML engine
- Not a fraud detection engine
- Not a generic chatbot
- Not a core banking system
- Not another dashboard

## GitHub Repository
https://github.com/reddybashanaguru

---

## Tech Stack (FROZEN — do not change without explicit discussion)

### Runtime Versions (actual installed — use these)
- Node.js 23.9.0 (v23.x — next LTS cycle)
- npm 11.x
- Go 1.25.x (go.mod: 1.25.7, upgraded by pgx/v5 requirement)
- Python 3.9.6 (system — all libraries confirmed compatible)

### Frontend
- Next.js 16.2.6 LTS (App Router, Turbopack stable, React Compiler stable)
- React 19 (concurrent features, use client/server components)
- TypeScript 5.8 (strict mode, no `any`)
- Tailwind CSS v4 (CSS-first config — no tailwind.config.js)
- shadcn/ui (latest — updated for React 19 + Tailwind v4)
- TanStack Query v5 (server state)
- Zustand v5 (client state)
- Tiptap v3 (rich text editor for notes, ProseMirror-based)
- React Flow v12 (graph — Phase 2 only)
- SSE for real-time updates (NOT WebSockets for MVP)

### Backend (Go — Modular Monolith)
- Go 1.25.x
- Chi v5.2.5 router (idiomatic, lightweight)
- sqlc v1.29 (type-safe SQL — NOT an ORM)
- golang-jwt v5 (RS256 auth)
- River v0.37.1 (Go-native job queue on PostgreSQL)
- Zap v1.28 (structured logging)
- OpenTelemetry Go SDK v1.35 (tracing + metrics)

### AI Service (Python — Separate Process)
- Python 3.9.6
- FastAPI 0.115 (latest stable)
- LangChain 0.3 (latest stable)
- Anthropic SDK (latest) + OpenAI SDK (latest) — abstracted behind LLM router
- pgvector via psycopg3 (NOT psycopg2)
- PyMuPDF 1.25 (PDF text extraction)
- Tesseract OCR 5.x + pytesseract (image text extraction)
- Presidio 2.x (PII detection + masking before every LLM call)

### Database
- PostgreSQL 17 (pgvector/pgvector:pg17 Docker image — pg18 image not yet available)
- pgvector 0.8 extension (embeddings for RAG — vector(1536))
- River (background jobs on Postgres — no Redis needed for MVP)

### File Storage
- MinIO AGPL (local/on-prem dev — S3-compatible API)
- AWS S3 (cloud prod)

### Email
- Resend (latest SDK)

### Infrastructure
- Docker 27 + Docker Compose v2.35 (MVP)
- GitHub Actions (CI/CD)
- Prometheus 3.x + Grafana 11.x (observability)
- OpenTelemetry Collector (traces → Grafana Tempo)

---

## Project Structure
```
investigation-workspace/
├── apps/
│   ├── web/          ← Next.js frontend
│   ├── api/          ← Go modular monolith
│   └── ai-service/   ← Python FastAPI AI service
├── packages/
│   ├── shared-types/ ← TypeScript types shared between web and api
│   └── ui-components/← Reusable shadcn/ui-based components
├── infra/
│   ├── docker/       ← Dockerfiles + docker-compose
│   └── scripts/      ← DB migration, seed scripts
├── docs/             ← All documentation
├── CLAUDE.md         ← This file
└── README.md
```

---

## MVP Scope — 12 Features (FROZEN)

| # | Feature | Status |
|---|---------|--------|
| 1 | Authentication & RBAC | ✅ Complete |
| 2 | Case Management | ✅ Complete |
| 3 | Timeline Workspace | ✅ Complete |
| 4 | AI Copilot | 🔲 Not started |
| 5 | Document Upload & Intelligence | ✅ Complete |
| 6 | Notes & Collaboration | 🔲 Not started |
| 7 | Task Management | ✅ Complete |
| 8 | Dashboard | 🔲 Not started |
| 9 | Notifications | 🔲 Not started |
| 10 | Audit Logging | 🔲 Not started |
| 11 | Search | 🔲 Not started |
| 12 | AI Logging & Explainability | 🔲 Not started |

**Update this table as each feature completes.**

---

## EXPLICITLY OUT OF SCOPE FOR MVP
Do NOT build these regardless of how good they sound:
- AML rules engine
- Transaction monitoring
- Sanctions screening
- Graph analytics
- Advanced ML pipelines
- Kubernetes infra
- Multi-tenancy
- No-code workflow builder
- Advanced analytics / BI
- Billing engine
- External integrations marketplace
- Autonomous AI agents
- Mobile apps
- Multi-region deployments

---

## Development Workflow (Follow Every Time)

### For each feature:
1. Read the feature spec in MASTER_BLUEPRINT.html
2. Build BE (Go module) first — schema migration, service, handlers, tests
3. Build AI service endpoints if needed (Python FastAPI)
4. Build FE (Next.js) — components, API integration, UX
5. **Put yourself in the end user's shoes** — use the feature as an investigator would
6. Identify UX/logic gaps and fix them before moving on
7. Run full test suite — coverage must be >90%
8. Commit to GitHub with structured commit message
9. Update this CLAUDE.md — mark feature as ✅ complete, note any decisions made

### Commit Message Format
```
feat(module): short description

- What was built
- Key decisions made
- Known limitations

Closes #issue-number (if applicable)
```

### Testing Requirements
- Backend (Go): table-driven tests, >90% coverage per package
- Frontend (Next.js): Vitest for unit, Playwright for E2E
- AI Service (Python): pytest, mock LLM calls in tests
- Every API endpoint must have integration tests
- Every critical business logic path must have unit tests

---

## API Standards
- All endpoints versioned: `/api/v1/`
- All responses: `{ data, meta, error }` envelope
- All errors: `{ code, message, details }` structure
- Pagination: cursor-based for lists
- Dates: ISO 8601 UTC always
- IDs: UUID v4 always
- Amounts: integers in smallest unit (never floats)
- Auth: Bearer JWT in Authorization header

### Performance Targets
- GET (cached): < 50ms p99
- GET (DB): < 100ms p99
- POST/PUT: < 200ms p99
- Search: < 200ms p99
- AI operations: async, webhook/SSE on completion

---

## AI Anti-Hallucination Rules (CRITICAL)
1. Every AI response MUST cite the source document/event it used
2. AI outputs MUST include a confidence indicator
3. When context is insufficient → AI says "I don't have enough context" NOT a guess
4. All AI outputs go through a human review gate before becoming "official"
5. AI-generated content is always visually distinct (badge + border)
6. Prompt injection prevention on all user inputs
7. PII masking before every AI API call
8. Store every prompt + response + model + token count in ai_logs table

---

## Database Rules
- Every table has: id (UUID), created_at, updated_at
- Soft deletes: deleted_at nullable column (never hard delete user data)
- All timestamps: TIMESTAMPTZ (not TIMESTAMP)
- Foreign keys: always defined and enforced
- Indexes: on every foreign key + every column used in WHERE/ORDER BY
- Migrations: versioned SQL files, never modify existing migrations
- No raw string queries — use sqlc generated code only

---

## Security Rules
- Passwords: bcrypt (cost 12)
- JWT: RS256 (not HS256), 1-hour expiry, refresh token rotation
- All file uploads: virus scan + type validation before storage
- RBAC enforced at middleware level, not just UI
- Rate limiting: 100 req/min per user, 10 req/min for AI endpoints
- All AI calls: PII stripped before sending to external LLM

---

## UX Rules
- Loading states: skeleton screens, never spinners alone
- Errors: always actionable ("Try again" or "Contact support")
- Empty states: always have a helpful message + CTA
- Every destructive action: confirmation dialog
- Timeline: newest events at top, oldest at bottom
- AI outputs: always show source + confidence + "Review" button
- Mobile: responsive but not mobile-first (enterprise = desktop)

---

## After Each Feature — Update This File
Mark the feature complete in the table above.
Add a section at the bottom: `## Completed Features Log`
Include: what was built, decisions made, known tech debt.

---

## Custom Slash Commands (use these)

| Command | When to use |
|---------|-------------|
| `/infra-setup` | Run ONCE at the very start — bootstraps all 3 apps, Docker, CI/CD |
| `/feature N` | Start building feature N — follows full BE→AI→FE→test workflow |
| `/check` | Run all quality gates (Go tests, Python pytest, TS typecheck, ESLint) |
| `/status` | Show which features are done and what to build next |

---

## Feature Build Order (follow this — dependencies matter)

### Phase 0 — Infrastructure (do before any feature)
Run `/infra-setup` to complete this phase.

- [ ] Go module, Next.js 16, Python FastAPI initialized
- [ ] Docker Compose (postgres+pgvector, minio, api, web, ai-service)
- [ ] goose migration tooling
- [ ] sqlc configured
- [ ] GitHub Actions CI/CD
- [ ] Shared TypeScript types package

### Phase 1 — Foundation
**Feature 1: Authentication & RBAC** ← EVERYTHING depends on this
- Depends on: Phase 0
- Run: `/feature 1`

**Audit Logging Middleware** (wire up before Feature 2)
- Build the Go middleware and River job early
- Even though the viewer UI is Feature 10, the middleware must wrap all handlers from Feature 2 onward
- All subsequent features automatically get audit trails

### Phase 2 — Core Domain
Build in this order (each depends on the previous):

1. **Feature 2: Case Management** — Depends on: Feature 1, Audit middleware
2. **Feature 3: Timeline Workspace** — Depends on: Feature 2
   - ⚠️ Build the event emitter pattern here. ALL future features use `timelineService.Emit()` to post events.
3. **Feature 7: Task Management** — Depends on: Features 2, 3

### Phase 3 — Intelligence Layer
Build Features 4 and 12 together — they are deeply coupled.

1. **Feature 5: Document Upload & Intelligence** — Depends on: Features 2, 3
   - Python AI service: text extraction, chunking, pgvector embeddings
2. **Feature 4: AI Copilot + Feature 12: AI Logging** — Depends on: Feature 5
   - Build TOGETHER in one session: AI Copilot needs ai_logs; AI Logging is the ui/api for ai_logs

### Phase 4 — Collaboration & Discovery

1. **Feature 6: Notes & Collaboration** — Depends on: Features 2, 3
2. **Feature 9: Notifications** — Depends on: All previous features (needs their events)
   - Build last in this phase — wires up all River notification jobs from other modules
3. **Feature 11: Search** — Depends on: All entities + document chunks (Feature 5)
   - tsvector indexes must exist on cases, notes, tasks, documents before building search

### Phase 5 — Analytics & Polish

1. **Feature 8: Dashboard** — Depends on: All features (aggregates everything via materialized views)
2. **Feature 10: Audit Logging UI** — Middleware built in Phase 1; this phase only adds the viewer page + CSV export
3. **E2E testing pass** — Playwright golden paths for all 12 features
4. **Performance audit** — Query EXPLAIN ANALYZE on key endpoints, check p99 targets

---

---

## Architecture Invariants (never violate these)

1. **Every API response** uses `{data, meta, error}` envelope — no exceptions
2. **Every table** has id UUID, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ, deleted_at TIMESTAMPTZ
3. **No raw SQL** — only sqlc generated functions
4. **No hardcoded prompts** — prompts stored in ai_prompts table
5. **No AI output** enters case record without human_review_status = 'approved'|'edited'
6. **No PII** reaches LLM — Presidio masking always runs first
7. **Every mutating request** emits an audit_log entry (via middleware — handlers don't call this directly)
8. **Every case action** emits a timeline_event (handlers call timelineService.Emit explicitly)
9. **Amounts** always in smallest unit (paise), never floats
10. **Dates** always ISO 8601 UTC in API responses

---

## Completed Features Log

<!-- Append here after each feature completes -->

### ✅ Feature 5 — Document Upload & Intelligence (2026-05-17)
**Built:** Full document lifecycle — multipart upload to MinIO, async processing via Python AI service goroutine, cursor-paginated list with status filter, soft delete with RBAC. Storage abstraction (`storage.Client` interface) with MinIO implementation. MIME validation via magic bytes (`net/http.DetectContentType`) + extension fallback for `application/octet-stream`. Python AI service: PyMuPDF + Tesseract OCR extraction, boundary-aware chunking (~2000 chars / 200-char overlap), OpenAI `text-embedding-3-small` embeddings stored in `document_chunks` via pgvector/psycopg3.
**Key decisions:**
- Goroutine for async AI processing (no River job for MVP — avoids River table/worker complexity)
- No ClamAV antivirus scanning — stubbed; add in Phase 2 before production
- No PDF.js inline preview — 1-hour presigned MinIO URL for download; viewer in Phase 2
- `prefixReader` re-injects first 512B (consumed by MIME detection) so MinIO receives the full stream
- Explicit UUID pre-generated before DB insert so storage key (`case/{caseId}/doc/{docId}/{filename}`) is known before upload
- `resolveByExtension()` fallback for OOXML types (`docx`, `xlsx`, `doc`, `xls`) which are ZIP files that `DetectContentType` returns as `application/zip`
**BE tests:** 91% coverage, 40+ tests (shared testcontainers-go `TestMain` pattern to avoid per-test Docker container overhead)
**FE tests:** 30 component tests (DocumentCard 15, DocumentUploadZone 8, DocumentProcessingBadge 7)
**Frontend:** `/cases/[id]/documents` page with infinite scroll, status filter dropdown, drag-and-drop upload zone, document cards with processing status badge. "Documents →" nav added to case detail page.
**Known limitation:** Processor goroutine has no retry on AI service transient failure — marks document `failed` on first error. Add retry with exponential backoff in Phase 2.

### ✅ Feature 7 — Task Management (2026-05-16)
**Built:** Full task CRUD with state machine (todo→in_progress→done/cancelled), cursor pagination, overdue detection (synchronous on create/update — no River job for MVP), task comments (CRUD + pagination), `My Tasks` cross-case view grouped by urgency. Timeline events emitted on `task_created` and `task_completed`. RBAC: analysts can create/update own tasks, manager+ can update any.
**Key decisions:**
- `is_overdue` computed synchronously at create/update time (due_date < now), cleared immediately on complete — no River job needed for MVP correctness
- Cursor pagination uses `id < cursor_id` (UUID comparison) — works for display purposes but not perfectly ordered with `sort_order ASC`; acceptable for MVP, fix with UUIDv7 in Phase 2
- `claims.FullName` doesn't exist on `auth.Claims` — creator name fetched via `GetByID` join in response builders
- `nullStr` helper was dead code — removed to maintain clean codebase
**BE tests:** 92.5% coverage (39 tests: 16 service + 23 handler)
**FE tests:** 27 tests (TaskCard 16, TaskStatusBadge 5, TaskPriorityBadge 6)
**Frontend:** Case tasks page at `/cases/[id]/tasks`, task detail with comments at `/cases/[id]/tasks/[taskId]`, My Tasks cross-case page at `/tasks` grouped (Overdue→Today→This Week→Later→Done)
**Known limitation:** Cursor pagination for tasks uses UUID byte comparison vs sort_order ordering — page boundaries aren't perfectly stable. Fix in Phase 2 with UUIDv7 or composite cursor.

### ✅ Feature 3 — Timeline Workspace (2026-05-16)
**Built:** SSE real-time timeline hub with auth via `?token=` query param (EventSource can't set headers), base64 JSON cursor pagination (occurred_at + id), 20+ event types with color-coded icons, backdated flag (|occurred_at - created_at| > 1 hour), filter by source/event_type. Timeline emitted from cases (create/status-change/assign/tag) and tasks (create/complete).
**Key decisions:**
- `db.NullTime` doesn't exist — use `sql.NullTime` from standard library
- `jwtMgr.Verify()` is the correct method name (not Validate)
- SSE hub uses buffered channels (size 64) with non-blocking publish to drop slow consumers
**BE tests:** 14 tests (service + handler), all pass
**FE tests:** 13 tests for TimelineEventCard
**Known limitation:** SSE reconnect doesn't replay missed events between disconnect and reconnect — track `lastEventId` but server-side replay not implemented for MVP.

### ✅ Feature 2 — Case Management (2026-05-16)
**Built:** Full case CRUD with status state machine (draft→open→in_progress→pending_review→closed/escalated), cursor pagination, SLA breach tracking, watchers, tags, assignee, activity log. Role-based access: viewers can read, analysts can create/update own cases, managers can manage all.
**Key decisions:**
- `organization_id` is hardcoded to `00000000-0000-0000-0000-000000000001` for MVP (no multi-tenancy)
- SLA deadline computed at create time based on priority: critical=4h, high=24h, medium=72h, low=168h
- Watchers stored as UUID array in postgres (not a join table) for simplicity
**FE:** Dark navy/cyan design system matching spec. Cases list with filter/sort, case detail with status transitions, activity feed, SLA countdown.

### ✅ Feature 1 — Authentication & RBAC (2026-05-16)
**Built:** JWT RS256 login/refresh/logout/me/change-password, refresh token rotation with SHA-256 hash storage, family revocation on reuse detection, 5-role RBAC (super_admin → viewer), user CRUD, bcrypt cost 12, rate limiting (100 req/min token bucket), async audit log writes.
**Key decisions:**
- JWT/RBAC middleware lives in `internal/auth/` (not `internal/middleware/`) to avoid circular imports
- pgxpool wrapped via `stdlib.OpenDBFromPool` for sqlc database/sql compatibility
- Access token in memory (Zustand), refresh token as HttpOnly cookie + body
- `users/service.go` calls `auth.HashPassword()` — no bcrypt duplication
**Frontend:** Dark navy/cyan theme matching spec docs. Login page, protected dashboard layout with sidebar, user management table, RBACGate component.
**Known limitation:** `ChangePassword` stores new hash in memory but lacks a dedicated sqlc query to write it — add `UpdateUserPassword` query before shipping (non-blocking for development).

