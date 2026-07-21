# iroha task runner. Tools come from the Nix devShell (flake.nix).
# Outside `nix develop`, targets transparently wrap each command via NIX_DEV;
# inside the shell (IN_NIX_SHELL set) the prefix is empty and tools run directly.

SHELL := bash
NIX_DEV := $(if $(IN_NIX_SHELL),,nix develop $(CURDIR) --command )
SERVER_DIR := apps/iroha-server
WEB_DIR := apps/iroha-web
JOB_DIR := apps/iroha-job
IMAGE_NS := azusachino.icu
TAG := v0.1.1

.DEFAULT_GOAL := help
.PHONY: help fmt fmt-check vet lint test contract-check test-integration scripts-test build run run-job web-install web-fmt web-fmt-check web-check web-test web-build web-dev fmt-docs fmt-docs-check check validate dev-up dev-watch db-up db-down db-status db-logs db-reset smoke-real-import smoke-local soak-local image-server image-job image-db-migrate image-web images

PRETTIER := prettier
DOCS_GLOB := **/*.{md,yaml,yml,json}

help: ## List available targets
	@grep -hE '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN{FS=":.*## "}{printf "  \033[36m%-13s\033[0m %s\n", $$1, $$2}'

## --- Go (all workspace modules; discovered from go.work by scripts/go_tasks.py) ---
# Each target enters the devShell once; go_tasks.py fans the task out over every
# module in go.work, so a new module is covered with no Makefile edit.
fmt: ## Format Go code across all modules (gofumpt + goimports)
	$(NIX_DEV)uv run python scripts/go_tasks.py fmt

fmt-check: ## Fail if any Go file is unformatted (all modules)
	$(NIX_DEV)uv run python scripts/go_tasks.py fmt-check

vet: ## Run go vet across all modules
	$(NIX_DEV)uv run python scripts/go_tasks.py vet

lint: ## Run golangci-lint across all modules (Uber Go Style Guide orientation)
	$(NIX_DEV)uv run python scripts/go_tasks.py lint

test: ## Run Go tests across all modules
	$(NIX_DEV)uv run python scripts/go_tasks.py test

contract-check: ## Verify the registered HTTP route inventory
	$(NIX_DEV)go -C $(SERVER_DIR) test ./pkg/httpapi -run '^TestActiveRouteInventory$$'

test-integration: db-up ## Run DB-backed Go integration tests
	$(NIX_DEV)env DATABASE_URL=postgres://iroha:iroha_dev@127.0.0.1:5432/iroha?sslmode=disable go -C $(SERVER_DIR) test -tags=integration ./...

scripts-test: ## Run Python script unit tests
	$(NIX_DEV)uv run python -m unittest discover -s scripts -p '*_test.py'

build: ## Build all Go modules
	$(NIX_DEV)uv run python scripts/go_tasks.py build

run: db-up ## Run the server against the local dev stack (http://127.0.0.1:8080)
	$(NIX_DEV)go -C $(SERVER_DIR) run ./cmd/iroha-server

run-job: db-up ## Run one iroha-job polling worker against the local dev stack
	$(NIX_DEV)go -C $(JOB_DIR) run .

## --- Web frontend (apps/iroha-web, bun) ---
web-install: ## Install web dependencies
	cd $(WEB_DIR) && $(NIX_DEV)bun install

web-fmt: ## Format the web app (prettier + prettier-plugin-svelte: spaces, double quotes)
	cd $(WEB_DIR) && $(NIX_DEV)bun run format

web-fmt-check: ## Fail if any web file is unformatted
	cd $(WEB_DIR) && $(NIX_DEV)bun run format:check

web-check: ## Type-check the web app (svelte-check)
	cd $(WEB_DIR) && $(NIX_DEV)bun run check

web-test: ## Run unit tests for the web app (vitest)
	cd $(WEB_DIR) && $(NIX_DEV)bun run test

web-build: ## Production build of the web app
	cd $(WEB_DIR) && $(NIX_DEV)bun run build

web-dev: ## Run the web dev server, bound to all interfaces (Tailscale/LAN)
	cd $(WEB_DIR) && $(NIX_DEV)bun run dev --host 0.0.0.0

## --- Docs and config formatting (prettier; Go/web/SQL out of scope) ---
fmt-docs: ## Format docs and config files (markdown wraps at 200)
	$(NIX_DEV)$(PRETTIER) --write "$(DOCS_GLOB)"

fmt-docs-check: ## Fail if any doc/config file is unformatted
	$(NIX_DEV)$(PRETTIER) --check "$(DOCS_GLOB)"

## --- Aggregate gates ---
check: fmt-check vet lint test contract-check scripts-test web-fmt-check web-check web-test ## Pre-commit gate: fmt-check + vet + lint + test + contract route check + script tests + web checks
validate: check build web-build ## Pre-PR gate: check + full server and web builds

## --- Dev stack (Podman Compose via uv scripts) ---
dev-up: ## Start the complete local stack and apply migrations
	$(NIX_DEV)uv run python scripts/dev_stack.py start

dev-watch: ## Rebuild changed Podman Compose application services
	$(NIX_DEV)uv run python scripts/dev_watch.py

db-up: ## Start only Postgres/PostGIS and Valkey, then apply migrations
	$(NIX_DEV)uv run python scripts/dev_stack.py deps

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

smoke-local: ## Run real import smoke against the Podman Compose server and worker
	@test -n "$(FILE)" || (echo "FILE is required, e.g. make smoke-local FILE=.iroha-data/imports/export.zip" >&2; exit 2)
	$(NIX_DEV)uv run python scripts/real_import_smoke.py "$(FILE)" --assert --api-base "$(or $(API_BASE),http://127.0.0.1:8080)"

soak-local: ## Run non-mutating HTTP soak checks against the Podman Compose stack
	$(NIX_DEV)uv run python scripts/local_stack_soak.py $(SOAK_ARGS)

## --- k3s local images (build with Podman, import straight into containerd; no registry) ---
image-server: ## Build iroha-server and import it into the local k3s containerd store (TAG=v0.1.1)
	podman build --target server -t $(IMAGE_NS)/iroha-server:$(TAG) -f ops/images/Containerfile.server .
	podman save $(IMAGE_NS)/iroha-server:$(TAG) | sudo k3s ctr images import -

image-job: ## Build iroha-job and import it into the local k3s containerd store (TAG=v0.1.1)
	podman build --target job -t $(IMAGE_NS)/iroha-job:$(TAG) -f ops/images/Containerfile.server .
	podman save $(IMAGE_NS)/iroha-job:$(TAG) | sudo k3s ctr images import -

image-db-migrate: ## Build iroha-db-migrate and import it into the local k3s containerd store (TAG=v0.1.1)
	podman build --target db-migrate -t $(IMAGE_NS)/iroha-db-migrate:$(TAG) -f ops/images/Containerfile.server .
	podman save $(IMAGE_NS)/iroha-db-migrate:$(TAG) | sudo k3s ctr images import -

image-web: ## Build iroha-web and import it into the local k3s containerd store (TAG=v0.1.1)
	podman build -t $(IMAGE_NS)/iroha-web:$(TAG) -f ops/images/Containerfile.web --build-arg PUBLIC_IROHA_API_BASE= .
	podman save $(IMAGE_NS)/iroha-web:$(TAG) | sudo k3s ctr images import -

images: image-server image-job image-db-migrate image-web ## Build and import all iroha images into the local k3s containerd store
