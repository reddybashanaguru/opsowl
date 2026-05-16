# /status — OpsOwl AI build status

Read `CLAUDE.md` and report the current state of the project.

## Report format

### MVP Feature Status
Show the feature table from CLAUDE.md with current status icons.

### Infrastructure Status
Check if each item exists:
- [ ] `apps/api/go.mod` — Go module initialized
- [ ] `apps/web/package.json` — Next.js initialized  
- [ ] `apps/ai-service/requirements.txt` — Python dependencies
- [ ] `infra/docker/docker-compose.yml` — Docker Compose
- [ ] `apps/api/migrations/` — Migrations directory exists
- [ ] `.github/workflows/ci.yml` — CI pipeline

### What to build next
Based on the feature dependency order in CLAUDE.md, state:
1. What is the **next feature** to build (first incomplete, following dependency order)
2. What features are **blocked** by incomplete prerequisites
3. Estimated features remaining

### Quick commands
Remind the user:
- `/feature N` — start building feature N
- `/check` — run all quality gates
