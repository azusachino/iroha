# iroha task runner. Tools come from the Nix devShell (flake.nix).
# Outside `nix develop`, targets transparently wrap each command via NIX_DEV;
# inside the shell (IN_NIX_SHELL set) the prefix is empty and tools run directly.

SHELL := bash
NIX_DEV := $(if $(IN_NIX_SHELL),,nix develop $(CURDIR) --command )
SERVER_DIR := apps/iroha-server
WEB_DIR := apps/iroha-web

.DEFAULT_GOAL := help
.PHONY: help fmt fmt-check vet lint test test-integration build run run-job web-install web-check web-test web-build web-dev check validate db-up db-down db-status db-logs db-reset smoke-real-import

help: ## List available targets
	@grep -hE '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN{FS=":.*## "}{printf "  \033[36m%-13s\033[0m %s\n", $$1, $$2}'

## --- Go server (apps/iroha-server) ---
fmt: ## Format Go code (gofumpt)
	$(NIX_DEV)gofumpt -w $(SERVER_DIR)

fmt-check: ## Fail if any Go file is unformatted
	@files=$$($(NIX_DEV)gofumpt -l $(SERVER_DIR)); \
		if [ -n "$$files" ]; then echo "gofumpt needed:"; echo "$$files"; exit 1; fi

vet: ## Run go vet
	$(NIX_DEV)go -C $(SERVER_DIR) vet ./...

lint: ## Run golangci-lint (Uber Go Style Guide orientation)
	cd $(SERVER_DIR) && $(NIX_DEV)golangci-lint run ./...

test: ## Run Go tests
	$(NIX_DEV)go -C $(SERVER_DIR) test ./...

test-integration: db-up ## Run DB-backed Go integration tests
	$(NIX_DEV)env DATABASE_URL=postgres://iroha:iroha_dev@127.0.0.1:5432/iroha?sslmode=disable go -C $(SERVER_DIR) test -tags=integration ./...

build: ## Build the Go server
	$(NIX_DEV)go -C $(SERVER_DIR) build ./...

run: db-up ## Run the server against the local dev stack (http://127.0.0.1:8080)
	$(NIX_DEV)go -C $(SERVER_DIR) run ./cmd/iroha-server

run-job: db-up ## Run one iroha-job polling worker against the local dev stack
	$(NIX_DEV)go -C $(SERVER_DIR) run ./cmd/iroha-job

## --- Web frontend (apps/iroha-web, bun) ---
web-install: ## Install web dependencies
	cd $(WEB_DIR) && $(NIX_DEV)bun install

web-check: ## Type-check the web app (svelte-check)
	cd $(WEB_DIR) && $(NIX_DEV)bun run check

web-test: ## Run unit tests for the web app (vitest)
	cd $(WEB_DIR) && $(NIX_DEV)bun run test

web-build: ## Production build of the web app
	cd $(WEB_DIR) && $(NIX_DEV)bun run build

web-dev: ## Run the web dev server, bound to all interfaces (Tailscale/LAN)
	cd $(WEB_DIR) && $(NIX_DEV)bun run dev --host 0.0.0.0

## --- Aggregate gates ---
check: fmt-check vet lint test web-check web-test ## Pre-commit gate: fmt-check + vet + lint + test + web type-check + web tests
validate: check build web-build ## Pre-PR gate: check + full server and web builds

## --- Dev database (Postgres/PostGIS via uv scripts) ---
db-up: ## Start the local dev database stack and apply migrations
	$(NIX_DEV)uv run python scripts/dev_stack.py start

db-down: ## Stop the local dev database
	$(NIX_DEV)uv run python scripts/dev_stack.py stop

db-status: ## Show local dev database stack status
	$(NIX_DEV)uv run python scripts/dev_stack.py status

db-logs: ## Show local dev database logs
	$(NIX_DEV)uv run python scripts/dev_stack.py logs

db-reset: ## Reset the local dev database stack and apply migrations
	$(NIX_DEV)uv run python scripts/dev_stack.py reset

smoke-real-import: ## Upload/import a real local file through the HTTP API (FILE=...)
	@test -n "$(FILE)" || (echo "FILE is required, e.g. make smoke-real-import FILE=.iroha-data/imports/export.zip" >&2; exit 2)
	$(NIX_DEV)uv run python scripts/real_import_smoke.py "$(FILE)" --api-base "$(or $(API_BASE),http://127.0.0.1:8080)"
