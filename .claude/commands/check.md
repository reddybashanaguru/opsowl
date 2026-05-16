# /check — Run all OpsOwl AI quality gates

Run every quality check in sequence. Report PASS ✅ or FAIL ❌ for each. Do not stop on first failure — run all checks and report the full picture.

## Go Backend

```bash
cd apps/api
go build ./...
go vet ./...
go test ./... -cover -coverprofile=coverage.out
```

For each package with <90% coverage: list which lines/functions are uncovered.

## Python AI Service

```bash
cd apps/ai-service
python -m pytest --cov=app tests/ -v
```

Report coverage percentage. Flag any test that mocks incorrectly (should mock at the LLM provider boundary, not at the service boundary).

## Frontend

```bash
cd apps/web
npx tsc --noEmit
npm run lint
npm run test
```

## Database

Check that all migrations are valid:
```bash
# verify sqlc can generate from current schema
cd apps/api && sqlc generate
```

## Summary Report

After all checks, output a table:

| Check | Status | Issues |
|-------|--------|--------|
| Go build | ✅/❌ | ... |
| Go vet | ✅/❌ | ... |
| Go tests (coverage) | ✅/❌ | XX% |
| Python pytest | ✅/❌ | ... |
| Python coverage | ✅/❌ | XX% |
| TypeScript | ✅/❌ | ... |
| ESLint | ✅/❌ | ... |
| Frontend tests | ✅/❌ | ... |

If any check fails: fix the issues immediately before reporting done.
