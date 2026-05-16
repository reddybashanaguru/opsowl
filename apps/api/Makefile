.PHONY: run build test lint vet sqlc migrate migrate-down gen-keys

## Run development server (requires .env)
run:
	@source .env 2>/dev/null || true; go run ./cmd/server/...

## Build production binary
build:
	go build -ldflags="-s -w" -o bin/server ./cmd/server/...

## Run all tests with coverage
test:
	go test ./... -cover -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

## Run tests (verbose)
test-v:
	go test ./... -v -cover

## Vet
vet:
	go vet ./...

## Format
fmt:
	gofmt -w .

## Generate sqlc typed queries
sqlc:
	sqlc generate

## Run database migrations (requires DATABASE_URL env var)
migrate:
	goose -dir migrations postgres "$(DATABASE_URL)" up

## Roll back last migration
migrate-down:
	goose -dir migrations postgres "$(DATABASE_URL)" down

## Migration status
migrate-status:
	goose -dir migrations postgres "$(DATABASE_URL)" status

## Generate RS256 key pair for JWT
gen-keys:
	mkdir -p keys
	openssl genrsa -out keys/private.pem 4096
	openssl rsa -in keys/private.pem -pubout -out keys/public.pem
	@echo "Keys generated in keys/ — never commit these"

## Tidy modules
tidy:
	go mod tidy

## All checks (CI equivalent)
check: fmt vet test
	@echo "All checks passed ✅"
