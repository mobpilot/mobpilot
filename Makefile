.PHONY: dev down generate lint test build help

SERVICES := identity social media notification org appstore mcp bff webhook
MODULE   := github.com/mobpilot/mobpilot

# ─── Development ────────────────────────────────────────────────────────────

dev: ## Start full local stack (Docker Compose)
	docker compose -f deploy/docker-compose.yml --profile all up --build

dev-infra: ## Start infrastructure only (postgres, redis, nats, minio, ory stack)
	docker compose -f deploy/docker-compose.yml --profile infra up -d

down: ## Stop all containers
	docker compose -f deploy/docker-compose.yml --profile all down

logs: ## Tail logs for all services
	docker compose -f deploy/docker-compose.yml --profile all logs -f

bootstrap: ## One-command local setup (starts infra + configures Ory)
	./scripts/bootstrap-dev.sh

# ─── Code generation ─────────────────────────────────────────────────────────

generate: generate-sqlc generate-buf ## Run all code generators

generate-sqlc: ## Generate type-safe DB code with sqlc
	@for svc in $(SERVICES); do \
		if [ -f services/$$svc/infrastructure/postgres/sqlc/sqlc.yaml ]; then \
			echo "→ sqlc generate services/$$svc"; \
			cd services/$$svc/infrastructure/postgres/sqlc && sqlc generate && cd -; \
		fi \
	done

generate-buf: ## Generate Connect-RPC code from .proto files
	buf generate

generate-ogen: ## Generate HTTP handlers from OpenAPI specs
	@for spec in api/openapi/*.yaml; do \
		name=$$(basename $$spec .yaml); \
		echo "→ ogen $$spec → gen/$$name"; \
		ogen --target gen/$$name --clean $$spec; \
	done

# ─── Quality ─────────────────────────────────────────────────────────────────

lint: ## Run golangci-lint
	golangci-lint run ./...

lint-fix: ## Run golangci-lint with auto-fix
	golangci-lint run --fix ./...

test: ## Run all tests
	go test -race -count=1 ./...

test-integration: ## Run integration tests (requires running infra)
	go test -race -count=1 -tags integration ./...

# ─── Build ───────────────────────────────────────────────────────────────────

build: ## Build all service binaries
	@for svc in $(SERVICES); do \
		echo "→ build $$svc"; \
		go build -trimpath -ldflags="-s -w" -o bin/$$svc ./services/$$svc/cmd/$$svc; \
	done

build-%: ## Build a specific service (e.g. make build-identity)
	go build -trimpath -ldflags="-s -w" -o bin/$* ./services/$*/cmd/$*

# ─── Database ────────────────────────────────────────────────────────────────

migrate-apply: ## Apply all migrations (atlas)
	@for svc in $(SERVICES); do \
		if [ -d migrations/$$svc ]; then \
			echo "→ atlas migrate apply $$svc"; \
			atlas migrate apply --dir "file://migrations/$$svc" --url "$$DATABASE_URL"; \
		fi \
	done

migrate-diff: ## Check for schema drift
	@for svc in $(SERVICES); do \
		if [ -f tools/atlas/$$svc/schema.hcl ]; then \
			echo "→ atlas schema diff $$svc"; \
			atlas schema diff --from "$$DATABASE_URL?search_path=$$svc" --to "file://tools/atlas/$$svc/schema.hcl"; \
		fi \
	done

# ─── Admin frontend ──────────────────────────────────────────────────────────

admin-dev: ## Start admin frontend dev server
	cd admin && npm run dev

admin-build: ## Build admin frontend
	cd admin && npm run build

admin-check: ## Lint + type-check admin frontend
	cd admin && npx biome check . && npx tsc --noEmit

# ─── Helpers ─────────────────────────────────────────────────────────────────

help: ## Show this help
	@grep -E '^[a-zA-Z_%-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
