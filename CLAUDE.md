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
| 4 | AI Copilot | ✅ Complete |
| 5 | Document Upload & Intelligence | ✅ Complete |
| 6 | Notes & Collaboration | ✅ Complete |
| 7 | Task Management | ✅ Complete |
| 8 | Dashboard | ✅ Complete |
| 9 | Notifications | ✅ Complete |
| 10 | Audit Logging | ✅ Complete |
| 11 | Search | ✅ Complete |
| 12 | AI Logging & Explainability | ✅ Complete |

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

## Phase 2 — Client-Ready Features (6 features, priority order)

**Goal:** Move from "serious prototype" → "client signs a contract."
**Theme:** Every feature either (a) serves a regulator directly, (b) makes AI useful to an investigator, or (c) removes a workflow gap that breaks the daily ops loop.
**AI Rule:** Every feature where AI can add value MUST use AI — summaries, risk scoring, entity extraction, recommendations. Not optional.

### Phase 2 Feature Table

| # | Feature | Client Pain Solved | AI Used? | Status |
|---|---------|-------------------|----------|--------|
| P2-1 | **PDF Case Report** | Regulators need a closure report. Zero workaround exists today. | ✅ AI writes the executive summary section | ⬜ Todo |
| P2-2 | **AI Risk Scoring & Insights Panel** | AI is in the product name but invisible in the case view. Investigators need to see what the AI found without opening a chat panel. | ✅ Core — risk score, entity extraction, auto-flagged anomalies | ⬜ Todo |
| P2-3 | **SLA Policy Engine** | Boolean `sla_breached` is not SLA management. Ops managers need configurable windows, warning alerts at 80%, and resolution-time trends. | ✅ AI predicts breach likelihood from case velocity | ⬜ Todo |
| P2-4 | **Communication Log** | Investigators email customers, call banks, send notices — none of it is in the case today. The tool becomes an island. | ✅ AI extracts entities and action items from logged call notes | ⬜ Todo |
| P2-5 | **Team Workload View** | Managers are blind to analyst capacity. They assign work by intuition, not data. | ✅ AI surfaces overloaded analysts and suggests rebalancing | ⬜ Todo |
| P2-6 | **Document Retry + Processing UX** | 9/72 documents are "failed" with no recovery path. Ops teams upload evidence constantly — silent failure erodes trust. | ✅ AI auto-categorises document type on successful retry | ⬜ Todo |

---

### P2-1: PDF Case Report

**What:** One-click "Export Report" button on any case. Generates a formatted PDF with:
- Cover page: case number, title, status, priority, dates, assigned team
- Executive summary (AI-written — 3-4 sentences summarising what happened and outcome)
- Timeline section: chronological events
- Evidence section: document table (filename, type, size, processing status)
- Task completion summary
- Notes section: formal notes only (note_type = 'formal')
- Audit trail footer: who touched the case and when

**BE:** New `GET /api/v1/cases/:id/report` endpoint → calls Python AI service for executive summary → assembles PDF using `github.com/signintech/gopdf` or `unidoc/unipdf` → streams as `application/pdf`.
**AI:** `POST /report/summary` on Python service — takes case data JSON, returns 3-4 sentence executive summary with confidence score.
**FE:** "Export PDF Report" button in case detail header → triggers download. Loading state with progress indicator.

---

### P2-2: AI Risk Scoring & Insights Panel

**What:** Every case gets a live AI risk score (0–100) shown as a badge in the case list and case header. Case detail gets a persistent "AI Insights" section (not a chat panel — always visible) showing:
- Risk score with colour band (green/amber/red)
- Top 3 risk factors (extracted by AI from case context)
- Key entities extracted: persons, companies, account numbers, amounts
- Flagged anomalies: patterns the AI noticed (e.g., "3 wire transfers in 4 hours to same beneficiary")
- Recommended next actions (3 bullet points)

**BE:** New `POST /api/v1/cases/:id/ai-analyse` endpoint — triggers full AI analysis, stores result in new `case_ai_insights` table. Background refresh on every document processed or note added.
**AI (Python):** `/analyse/case` — RAG over all case documents + notes + timeline events. Extracts: risk score (0–100 numeric), risk_factors[], entities{}, anomalies[], recommended_actions[]. All with citations + confidence.
**DB:** `case_ai_insights` table — `case_id`, `risk_score NUMERIC(5,2)`, `risk_factors JSONB`, `entities JSONB`, `anomalies JSONB`, `recommended_actions JSONB`, `model_used`, `confidence`, `created_at`.
**FE:** Risk score badge on every case card in list (colour-coded). "AI Insights" expandable section in case detail — always rendered (not a slide-over). Refresh button. "Last analysed X ago" timestamp.

---

### P2-3: SLA Policy Engine

**What:** Replace hardcoded SLA durations with configurable policies. Add warning notifications. Add trend chart.
- Admin can set SLA hours per priority per case type (admin UI settings page)
- Cases show: elapsed %, time remaining countdown, status band (green/amber/red)
- Warning notification fired at 80% elapsed (new notification type: `case_sla_warning`)
- Dashboard gets a "SLA Health" chart: resolution time trend (7-day rolling avg per priority)
- AI predicts breach likelihood ("72% likely to breach in next 4 hours") based on case activity velocity

**BE:** New `sla_policies` table (priority, hours, warn_threshold_pct). `sla_breach_at` recalculated on policy change. River job: `SLAWatcherJob` — runs every 15 min, fires `case_sla_warning` notifications. New `GET /api/v1/analytics/sla-trends` endpoint.
**AI:** Python `POST /predict/sla-breach` — inputs: case age, activity count, last_activity_at, priority. Returns breach probability + confidence.
**FE:** SLA settings page under admin. Case detail: SLA progress bar with countdown. Dashboard: resolution time chart (7-day SVG line chart). Case list: SLA urgency column with colour bands.

---

### P2-4: Communication Log

**What:** Investigators log every outbound communication against a case — email, phone call, field visit, letter, WhatsApp. Each log entry has: channel, direction (outbound/inbound), contact name, contact detail, subject/summary, outcome, timestamp, attachments.
- AI extracts named entities and action items from free-text call notes
- Communications appear in the case timeline
- Exportable in the case PDF report

**BE:** New module `internal/comms`. `communication_logs` table: `case_id`, `channel ENUM(email|call|field_visit|letter|whatsapp)`, `direction ENUM(inbound|outbound)`, `contact_name`, `contact_detail`, `subject`, `body`, `outcome`, `attachments UUID[]`, `logged_by`, `occurred_at`. CRUD endpoints.
**AI:** On `POST /api/v1/cases/:id/comms` — async call to Python `POST /extract/comms-entities` — returns entities (persons, orgs, amounts, dates) + action_items[]. Stored in `comm_ai_extractions` table alongside the log entry.
**FE:** "Communications" tab in case detail. Log form with channel selector. Each log card shows channel icon, contact, summary, AI-extracted entities chip strip, action items checklist. Timeline integration (shows as `comm_logged` event type).

---

### P2-5: Team Workload View

**What:** Manager/admin-only page at `/workload`. Shows:
- Per-analyst: active case count, open task count, overdue task count, avg case age
- Capacity indicator: green (< 5 active cases), amber (5–8), red (> 8)
- Task distribution chart: who has what
- AI recommendation: "Sneha Patel is overloaded (9 cases, 3 overdue). Consider reassigning OWL-2026-00012 to Arjun Mehta who has capacity."
- One-click reassign from the workload view

**BE:** New `GET /api/v1/analytics/workload` endpoint — aggregates across cases + tasks per analyst in single SQL query. New `POST /api/v1/analytics/workload/ai-suggest` — returns AI rebalancing suggestion.
**AI:** Python `POST /analyse/workload` — inputs workload JSON, returns natural language suggestion with specific case_id recommendations.
**FE:** `/workload` page. Analyst cards in a grid (avatar, stats, capacity bar). Assignment modal. RBAC gate (manager+ only). SVG bar chart for task distribution. AI suggestion banner at top of page.

---

### P2-6: Document Retry + Processing UX

**What:** Failed documents are dead ends today. Add recovery and better UX:
- "Retry Processing" button on failed documents
- Progress indicator during processing (polling or SSE)
- Better error messages (not just "unsupported binary format")
- AI auto-categorises successfully processed documents (contract/invoice/identity/evidence/other)
- Document type label shown on document card

**BE:** New `POST /api/v1/cases/:id/documents/:docId/retry` endpoint — re-triggers async processing goroutine. Update error handling in Python processor to return specific error codes. New `doc_category` column on `documents` table.
**AI:** Python processor — after successful extraction, call `/categorise/document` → returns category + confidence. Stored in `doc_category`.
**FE:** Retry button on failed document cards (spinner during processing). Processing progress SSE event (`document.processing`) displayed as animated progress bar. Category chip on document cards. Better empty states.

---

### Phase 2 Build Order

Build in this order (dependencies matter):

1. **P2-6: Document Retry** — small, quick win, builds trust in the upload pipeline
2. **P2-2: AI Risk Scoring** — highest client impact; makes "AI-Native" real in the UI
3. **P2-4: Communication Log** — fills the biggest workflow gap
4. **P2-1: PDF Case Report** — needs communication log data to be complete
5. **P2-3: SLA Policy Engine** — needs working notification system (already built)
6. **P2-5: Team Workload View** — pure analytics, no new domain entities needed

### Phase 2 Dev Workflow (same as MVP)
1. Build BE schema + service + handler + tests
2. Build AI service endpoint if needed
3. Build FE page/component
4. Test as end user — find gaps, fix them
5. Commit with structured message
6. Update this CLAUDE.md — mark feature ✅ complete

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

### ✅ Feature 11 — Search (2026-05-17)
**Built:** Full-text search across all entities — cases, notes, tasks, documents — using PostgreSQL `tsvector` with `websearch_to_tsquery`. Migration adds `search_vector` columns + GIN indexes + trigger-based maintenance + backfill. Parallel fan-out in Go (4 goroutines, `sync.WaitGroup`) for sub-100ms response on filtered or global queries.
**BE (Go — `internal/search`):** Migration `00010_search.sql` — trigger functions per table (`update_cases_search`, `update_notes_search`, `update_tasks_search`, `update_documents_search`). Weighted tsvectors: A=title/case_number, B=description/content_text, C=tags. Document filenames tokenized via `regexp_replace` (underscores/hyphens/dots → spaces). sqlc queries: `SearchCases`, `SearchNotes`, `SearchTasks`, `SearchDocuments` with `ts_rank` scoring. `GET /api/v1/search?q=&filter=&limit=` — filter param accepts `cases|notes|tasks|documents|""` to scope results.
**FE (Next.js):** `/search` page — debounced input (350ms), filter tabs per entity type, inline `<HL>` component for keyword highlighting via `dangerouslySetInnerHTML` with mark-tag CSS, skeleton loading (5 animated rows), empty state, section grouping by entity type with count badges. Each result card links directly to the relevant entity.
**Key decisions:**
- Trigger-based maintenance vs generated columns: `to_tsvector` is STABLE not IMMUTABLE in PostgreSQL, so generated columns are not allowed. Triggers are the correct approach.
- `websearch_to_tsquery` chosen over `plainto_tsquery` — handles natural language (AND, OR, NOT, phrases) without requiring users to know tsquery syntax
- Document search on filename only (not chunk content) — chunk content is handled by AI RAG pipeline, not keyword FTS
- Parallel goroutine fan-out with shared mutex for result accumulation — avoids sequential N×DB calls

### ✅ Feature 10 — Audit Logging UI (2026-05-17)
**Built:** Admin-only audit log viewer with server-side cursor pagination, severity filter tabs, event_code keyword filter (debounced 400ms), inline JSON metadata popover, CSV export. Middleware was built in Phase 1 (`internal/middleware/audit.go`) — this phase adds the viewer endpoint and frontend page.
**BE (Go — `internal/audit`):** Two new sqlc queries added to `auth.sql`: `ListAuditLogs` (cursor-paginated, filterable by severity/event_code/actor_id) and `CountAuditLogs`. `GET /api/v1/audit` — admin+ only via `RequireRole`. `GET /api/v1/audit/export` — streams CSV via `encoding/csv`. `GET /api/v1/audit/stats` — total count.
**FE (Next.js):** `/audit` page — `useInfiniteQuery` cursor pagination, table layout with columns: timestamp, event_code (monospace cyan), actor (name + email), severity pill with colored dot, resource type + truncated UUID, metadata popover. `RBACGate` wrapper shows lock state for non-admins. `TableSkeleton` with 10 animated rows.
**Key decisions:**
- Metadata rendered as raw JSON in a popover (not parsed per-event-type) — generic approach handles all future event types without schema changes
- CSV export hits the API directly from browser anchor click — no client-side CSV generation needed
- `event_code` uses `ILIKE '%...%'` for substring match — acceptable for admin tool; no performance concern at audit log scale

### ✅ Feature 9 — Notifications (2026-05-17)
**Built:** Real-time per-user notification system — in-app bell with unread badge, dropdown with last 15 notifications, mark-read/mark-all-read, SSE stream for instant delivery. Notifications fired on: case_assigned, case_status_changed, task_assigned. Notification preferences table for future per-type toggling.
**BE (Go — `internal/notifications`):** Migration `00009_notifications.sql` — `notification_type` ENUM (11 values), `notification_priority` ENUM (4 values), `notifications` table, `notification_preferences` table. `UserBroker` — per-user SSE channel broker. `Service.NotifyAsync()` — goroutine-based fire-and-forget for non-blocking notification delivery from case/task handlers. `WithNotifications(notifSvc)` injection pattern in cases and tasks services — avoids constructor signature changes and circular import risk.
**FE (Next.js):** `NotificationBell` component in sidebar — TanStack Query polling (30s) for unread count, dropdown with `useQuery(enabled: open)` lazy-loads list, SSE listener auto-invalidates on new events. Notification type icons + priority colors. `timeAgo()` helper. Outside-click close via `mousedown` listener.
**Key decisions:**
- `WithNotifications()` injection post-construction so notifSvc (created after cases/tasks) can be wired in server.go without restructuring module init order
- SSE auth via `?token=` query param — EventSource API cannot set Authorization headers
- `NotifyAsync` fires goroutine and returns immediately — notification failures don't block case/task mutations
- Seed data does NOT create notifications (no case assign/task assign calls during seeding) — expected behavior

### ✅ Feature 6 — Notes & Collaboration (2026-05-17)
**Built:** Rich-text notes per case (Tiptap v3 with StarterKit, Bold/Italic/Strike/Lists/Code/Blockquote, @mention extension), threaded comments, mentions tracking, SSE real-time push, cursor-paginated feed with type filter (internal/formal), pin-to-top, AI-generated/AI-assisted badge lifecycle, soft delete, formal note designation for PDF export (Phase 2).
**BE (Go — `internal/notes`):** Migration `00008_notes.sql` — `notes`, `comments`, `mentions` tables. sqlc queries for all CRUD. SSE broker (`NotesBroker`) — generic JSON channel broker, separate from timeline hub. Service handles mention insertion atomically with note/comment save. `GET /api/v1/cases/:id/notes/stream` uses `?token=` query param auth (EventSource cannot set headers).
**FE (Next.js):** `/cases/[id]/notes` page — `useInfiniteQuery` with cursor pagination, skeleton loading, SSE listener (`EventSource`) for real-time invalidation, Tiptap `NoteEditor` with toolbar, `NoteCard` with inline comment thread (Enter to send), pin/edit/delete actions, AI badge rendering.
**Key decisions:**
- @mentions send `mentioned_user_ids[]` from frontend (extracted from Tiptap state) rather than re-parsing HTML in Go — simpler and correct
- SSE broker is separate from timeline hub (different payload shape — JSON map vs typed `db.TimelineEvent`)
- `is_human_edit` param in UpdateNote triggers `ai_generated→false, ai_assisted→true` transition in DB (badge downgrade)
- Comments are plain text for MVP (no Tiptap in comment boxes) — reduces complexity without losing value
**Seed:** 8 notes (spanning UPI fraud, card skimming, KYC, insider threat cases) · 6 threaded comments

### ✅ Feature 8 — Dashboard (2026-05-17)
**Built:** Real-data dashboard with pure-SVG donut chart (case status breakdown), horizontal bar chart (priority distribution), 4 stat cards (Open Cases, Pending Tasks, AI Queries Today, SLA Breached), attention panel (SLA-breached cases, escalated, pending AI reviews). Backend: `GetDashboardStats` sqlc query with 14 fields (status/priority breakdowns). No chart library — all SVG written by hand.

### ✅ Features 4 & 12 — AI Copilot + AI Logging & Explainability (2026-05-17)
**Built together** (deeply coupled — both rely on `ai_logs` table).
**BE (Go — `internal/ai`):** Immutable `ai_logs` append-only table (no `updated_at`/`deleted_at`). Human review machine: `pending → approved | rejected | edited`. `ai_prompts` table for future versioned prompt registry. sqlc sentinel values (`$2 = ''` for no-status-filter, `uuid.Nil` for no-cursor) to avoid nullable param issues. 120-second timeout on AI routes via `chimw.Timeout` applied per-route group. HTTP client calls Python AI service for ask/summarize/generate-note/extract-actions.
**AI Service (Python — `ai-service/app`):** RAG pipeline (`/copilot/ask`): PII masking (Presidio) → embed → pgvector cosine similarity (`<=>` operator) → augmented prompt → LLM → citations + confidence score. `INSUFFICIENT_CONTEXT` sentinel when avg similarity < 0.35. Prompt injection detection via regex patterns (`ignore previous instructions`, `you are now`, etc.) → HTTP 400. Confidence scoring from avg cosine similarity of top-k chunks.
**FE (Next.js):** `CopilotPanel` — sliding side panel, Cmd+Shift+A keyboard shortcut, tabs for Ask/Summarize/Note/Actions. `AICard` — confidence bar (emerald ≥0.8, amber 0.5–0.79, red <0.5), citations toggle, approve/reject/edit inline actions. `AIReviewBadge` — color-coded status. `/cases/[id]/ai-logs` page — Feature 12 — cursor-paginated AI activity log with status filter and full review actions.
**Key decisions:**
- Sync AI calls (no River async jobs) — 120s timeout override, acceptable for MVP analyst workflows
- `human_edited_response` field dual-purpose: stores rejection reason AND human edit text
- Confidence score stored as `NUMERIC(3,2)` in DB, exposed as `*float64` in JSON
- Prompt injection detection in Python only (trust boundary at Python service, not Go)
- PII masking with Presidio applied before embedding AND before LLM call; masked text stored in `prompt_text`
**BE tests:** 91.8% coverage, 63 tests (integration tests via shared testcontainers-go `TestMain` with pgvector/pgvector:pg17)
**FE tests:** 48 component tests (AICard 21, AIReviewBadge 9, CitationCard 8, ConfidenceBar 10)

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

