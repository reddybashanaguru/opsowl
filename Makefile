.PHONY: help up infra down logs build test check migrate gen-keys setup gen-certs backup

COMPOSE := docker compose -f infra/docker/docker-compose.yml --env-file .env

## Show available commands
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

## Start only infrastructure (postgres + minio) — for local dev
infra: ## Start only postgres + minio
	$(COMPOSE) up -d postgres minio

## Start all services
up: ## Start all Docker services
	@test -f .env || (echo "❌ .env not found — copy .env.example to .env and fill in values"; exit 1)
	$(COMPOSE) up -d

## Stop all services
down: ## Stop all Docker services
	$(COMPOSE) down

## View logs (all services or specific: make logs s=api)
logs: ## View logs (make logs s=api)
	$(COMPOSE) logs -f $(s)

## Rebuild Docker images
build: ## Rebuild all Docker images
	$(COMPOSE) build

## Run all tests (Go + Python + Frontend)
test: ## Run full test suite
	@echo "▶ Go tests..."
	cd apps/api && go test ./... -cover
	@echo "▶ Python tests..."
	cd apps/ai-service && python -m pytest
	@echo "▶ Frontend tests..."
	cd apps/web && npm run test
	@echo "✅ All tests passed"

## Run quality checks
check: ## Run linters and type checkers
	cd apps/api && go vet ./... && go build ./...
	cd apps/web && npx tsc --noEmit && npm run lint
	@echo "✅ All checks passed"

## Run database migrations
migrate: ## Run pending database migrations
	cd apps/api && goose -dir migrations postgres "$(DATABASE_URL)" up

## Generate sqlc typed queries
sqlc: ## Generate type-safe SQL queries
	cd apps/api && sqlc generate

## Generate JWT RS256 key pair
gen-keys: ## Generate RSA key pair for JWT signing
	cd apps/api && make gen-keys

## Generate self-signed TLS certificate for development
gen-certs: ## Generate dev TLS cert (replace with CA-signed cert for production)
	bash infra/scripts/gen-dev-certs.sh

## Run a manual backup
backup: ## Backup database + documents to ./backups/
	bash infra/scripts/backup.sh

## First-time setup
setup: ## Install all dependencies and generate keys/certs
	@echo "▶ Installing Go deps..."
	cd apps/api && go mod download
	@echo "▶ Installing Node deps..."
	cd apps/web && npm install
	@echo "▶ Installing Python deps..."
	cd apps/ai-service && pip install -r requirements.txt
	@echo "▶ Generating JWT keys..."
	cd apps/api && make gen-keys
	@echo "▶ Generating TLS certificate..."
	$(MAKE) gen-certs
	@echo "▶ Copying .env.example → .env (edit before running 'make up')"
	@test -f .env || cp .env.example .env
	@echo "✅ Setup complete — edit .env, then run 'make up'"
