# /infra-setup — Bootstrap OpsOwl AI project infrastructure

Initialize the complete project infrastructure (Phase 0). Run this once before building any feature.

## What this sets up

1. Go module + dependencies
2. Next.js 16 app
3. Python FastAPI service
4. Docker Compose (all services)
5. Database migration tooling
6. CI/CD GitHub Actions pipeline
7. Shared types package

---

## Step 1 — Go Module (apps/api)

```bash
cd apps/api
go mod init github.com/reddybashanaguru/opsowl/api
```

Create `apps/api/go.sum` and install core dependencies:
```bash
go get github.com/go-chi/chi/v5@latest
go get github.com/golang-jwt/jwt/v5@latest
go get github.com/riverqueue/river@latest
go get github.com/jackc/pgx/v5@latest
go get go.uber.org/zap@latest
go get go.opentelemetry.io/otel@latest
go get golang.org/x/crypto@latest
go get github.com/google/uuid@latest
go get github.com/pressly/goose/v3@latest
go mod tidy
```

Create directory structure:
```
apps/api/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   ├── db/
│   └── server/
├── migrations/
├── sqlc.yaml
└── go.mod
```

Write `cmd/server/main.go` — basic Chi server that starts on port 8080.
Write `sqlc.yaml` — sqlc config pointing to migrations/ and internal/db/.
Write `internal/config/config.go` — env-based config (DATABASE_URL, JWT_SECRET_PATH, etc.)

## Step 2 — Next.js App (apps/web)

```bash
cd apps/web
npx create-next-app@16 . --typescript --tailwind --eslint --app --src-dir --import-alias "@/*"
```

Then install additional deps:
```bash
npm install @tanstack/react-query@5 zustand@5 @tiptap/react@3 @tiptap/starter-kit@3
npm install -D @playwright/test vitest @vitejs/plugin-react
npx shadcn@latest init
```

Configure:
- `tailwind.config.ts` → Tailwind v4 CSS-first config
- `next.config.ts` → enable Turbopack, strict mode
- `src/lib/api.ts` → base fetch wrapper with auth header + `{data, meta, error}` envelope handling
- `src/lib/query-client.ts` → TanStack Query client config

## Step 3 — Python AI Service (apps/ai-service)

Create `apps/ai-service/requirements.txt`:
```
fastapi==0.115.*
uvicorn[standard]
langchain==0.3.*
anthropic
openai
pgvector
psycopg[binary]
presidio-analyzer
presidio-anonymizer
pymupdf
pytesseract
python-multipart
pydantic-settings
pytest
pytest-asyncio
pytest-cov
httpx
```

Create structure:
```
apps/ai-service/
├── app/
│   ├── main.py          # FastAPI app, routers, lifespan
│   ├── config.py        # Pydantic settings
│   ├── db.py            # pgvector connection
│   ├── routers/
│   └── services/
│       ├── llm_router.py
│       └── pii_masker.py
├── tests/
│   └── conftest.py
├── requirements.txt
└── Dockerfile
```

Write `app/main.py` — FastAPI app with health endpoint at GET /health.
Write `app/config.py` — env-based config.
Write `app/services/llm_router.py` — provider abstraction (Anthropic/OpenAI).

## Step 4 — Docker Compose (infra/docker/)

Write `infra/docker/docker-compose.yml`:

```yaml
services:
  postgres:
    image: pgvector/pgvector:pg18
    ports: ["5432:5432"]
    environment:
      POSTGRES_DB: opsowl
      POSTGRES_USER: opsowl
      POSTGRES_PASSWORD: opsowldev
    volumes:
      - postgres_data:/var/lib/postgresql/data

  minio:
    image: minio/minio:latest
    ports: ["9000:9000", "9001:9001"]
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin123
    command: server /data --console-address ":9001"
    volumes:
      - minio_data:/data

  api:
    build: ../../apps/api
    ports: ["8080:8080"]
    depends_on: [postgres, minio]
    environment:
      DATABASE_URL: postgres://opsowl:opsowldev@postgres:5432/opsowl?sslmode=disable

  web:
    build: ../../apps/web
    ports: ["3000:3000"]
    depends_on: [api]

  ai-service:
    build: ../../apps/ai-service
    ports: ["8000:8000"]
    depends_on: [postgres]
    environment:
      DATABASE_URL: postgres://opsowl:opsowldev@postgres:5432/opsowl?sslmode=disable

volumes:
  postgres_data:
  minio_data:
```

Also write individual `Dockerfile` for each app.

## Step 5 — Database Migration Tooling

Use **goose** for migrations:
```bash
cd apps/api
go install github.com/pressly/goose/v3/cmd/goose@latest
mkdir -p migrations
```

Write first migration `migrations/00001_init.sql`:
```sql
-- Enable pgvector
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Base schema
CREATE SCHEMA IF NOT EXISTS public;
```

Write `infra/scripts/migrate.sh` — runs goose up.

## Step 6 — CI/CD (GitHub Actions)

Write `.github/workflows/ci.yml`:
- Trigger: push to main, PRs to main
- Jobs:
  - `go-test`: go test ./... -cover, go vet, go build
  - `python-test`: pytest --cov=app
  - `frontend-check`: npm run typecheck, npm run lint, npm run test
  - `docker-build`: docker compose build (validates Dockerfiles)
- All jobs run in parallel except docker-build (runs after all pass)

## Step 7 — Shared Types (packages/shared-types/)

Write `packages/shared-types/src/index.ts`:
```typescript
// API envelope
export interface APIResponse<T> {
  data: T;
  meta?: { cursor?: string; total?: number; };
  error?: { code: string; message: string; details?: unknown; };
}

// Pagination
export interface CursorPage<T> {
  items: T[];
  next_cursor: string | null;
  has_more: boolean;
}
```

## Step 8 — Verify

Run:
```bash
cd apps/api && go build ./...
cd apps/web && npm run typecheck
cd infra/docker && docker compose build
```

All three must succeed before proceeding to Feature 1.

## Step 9 — First Commit

```
chore: initialize OpsOwl AI project infrastructure

- Go 1.26 modular monolith (Chi, sqlc, River, pgx)
- Next.js 16 with TypeScript strict, Tailwind v4, shadcn/ui
- Python 3.14 FastAPI AI service
- Docker Compose with PostgreSQL 18 + pgvector, MinIO
- goose database migration tooling
- GitHub Actions CI/CD pipeline
- Shared TypeScript types package

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
```
