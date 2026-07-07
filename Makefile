# iroha task runner. Tools come from the Nix devShell (flake.nix).
# Outside `nix develop`, targets transparently wrap each command via NIX_DEV;
# inside the shell (IN_NIX_SHELL set) the prefix is empty and tools run directly.

SHELL := bash
NIX_DEV := $(if $(IN_NIX_SHELL),,nix develop $(CURDIR) --command )
SERVER_DIR := apps/iroha-server
WEB_DIR := apps/iroha-web

.DEFAULT_GOAL := help
.PHONY: help fmt fmt-check vet test build web-install web-check web-test web-build check validate db-up db-down db-reset

help: ## List available targets
	@grep -hE '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN{FS=":.*## "}{printf "  \033[36m%-13s\033[0m %s\n", $$1, $$2}'

## --- Go server (apps/iroha-server) ---
fmt: ## Format Go code
	$(NIX_DEV)gofmt -w $(SERVER_DIR)

fmt-check: ## Fail if any Go file is unformatted
	@files=$$($(NIX_DEV)gofmt -l $(SERVER_DIR)); \
		if [ -n "$$files" ]; then echo "gofmt needed:"; echo "$$files"; exit 1; fi

vet: ## Run go vet
	$(NIX_DEV)go -C $(SERVER_DIR) vet ./...

test: ## Run Go tests
	$(NIX_DEV)go -C $(SERVER_DIR) test ./...

build: ## Build the Go server
	$(NIX_DEV)go -C $(SERVER_DIR) build ./...

## --- Web frontend (apps/iroha-web, bun) ---
web-install: ## Install web dependencies
	cd $(WEB_DIR) && $(NIX_DEV)bun install

web-check: ## Type-check the web app (svelte-check)
	cd $(WEB_DIR) && $(NIX_DEV)bun run check

web-test: ## Run unit tests for the web app (vitest)
	cd $(WEB_DIR) && $(NIX_DEV)bun run test

web-build: ## Production build of the web app
	cd $(WEB_DIR) && $(NIX_DEV)bun run build

## --- Aggregate gates ---
check: fmt-check vet test web-check web-test ## Pre-commit gate: fmt-check + vet + test + web type-check + web tests
validate: check build web-build ## Pre-PR gate: check + full server and web builds

## --- Dev database (Postgres/PostGIS via uv scripts) ---
db-up: ## Start the local dev database
	$(NIX_DEV)uv run python scripts/dev_db.py start

db-down: ## Stop the local dev database
	$(NIX_DEV)uv run python scripts/dev_db.py stop

db-reset: ## Reset the local dev database
	$(NIX_DEV)uv run python scripts/dev_db.py reset
