.PHONY: help up infra down logs build test check migrate gen-keys setup

## Show available commands
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

## Start only infrastructure (postgres + minio) — use this for local dev
infra: ## Start only postgres + minio (run app services locally)
	docker compose -f infra/docker/docker-compose.yml up -d postgres minio

## Start all services in Docker
up: ## Start all Docker services (postgres, minio, api, web, ai-service)
	docker compose -f infra/docker/docker-compose.yml up -d

## Stop all services
down: ## Stop all Docker services
	docker compose -f infra/docker/docker-compose.yml down

## View logs (all services or specific: make logs s=api)
logs: ## View logs (make logs s=api)
	docker compose -f infra/docker/docker-compose.yml logs -f $(s)

## Rebuild Docker images
build: ## Rebuild all Docker images
	docker compose -f infra/docker/docker-compose.yml build

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
check: ## Run all linters and type checkers
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

## First-time setup
setup: ## Install all dependencies (run once after clone)
	@echo "▶ Installing Go deps..."
	cd apps/api && go mod download
	@echo "▶ Installing Node deps..."
	cd apps/web && npm install
	@echo "▶ Installing Python deps..."
	cd apps/ai-service && pip install -r requirements.txt
	@echo "▶ Generating JWT keys..."
	cd apps/api && make gen-keys
	@echo "✅ Setup complete — run 'make up' to start services"
