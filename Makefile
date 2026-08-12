# iroha task runner. Tools come from the checked-in mise configuration.

SHELL := bash
TOOL_ENV := mise exec --
SERVER_DIR := apps/iroha-server
WEB_DIR := apps/iroha-web
JOB_DIR := apps/iroha-job
PUBLIC_SITE_DIR := apps/iroha-public-site
IMAGE_NS := azusachino.icu
VERSION := $(shell tr -d '\n' < VERSION)
TAG := v$(VERSION)
OUT := ./dist/public-data
MEDIA_BRIDGE_OUT := ./dist/media-bridge

.DEFAULT_GOAL := help
.PHONY: help fmt fmt-check vet lint test contract-check test-integration scripts-test build run run-job export-public media-bridge-build web-install web-fmt web-fmt-check web-check web-test web-build web-dev web-visual-install web-visual-check public-site-install public-site-fmt-check public-site-check public-site-build public-site-dev public-site-preview fmt-docs fmt-docs-check check validate dev-up dev-watch db-up db-down db-status db-logs db-reset smoke-real-import smoke-local soak-local image-server image-job image-db-migrate image-web image-export-public images

PRETTIER := prettier
DOCS_FILES := $(shell rg --files -g '*.md' -g '*.yaml' -g '*.yml' -g '*.json' -g '!apps/iroha-web/**' -g '!apps/iroha-public-site/**' -g '!node_modules/**')

help: ## List available targets
	@grep -hE '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN{FS=":.*## "}{printf "  \033[36m%-13s\033[0m %s\n", $$1, $$2}'

## --- Go (all workspace modules; discovered from go.work by scripts/go_tasks.py) ---
# Each target enters the active tool environment once; go_tasks.py fans the
# task out over every module in go.work, so a new module needs no Makefile edit.
fmt: ## Format Go code across all modules (gofumpt + goimports)
	$(TOOL_ENV) uv run python scripts/go_tasks.py fmt

fmt-check: ## Fail if any Go file is unformatted (all modules)
	$(TOOL_ENV) uv run python scripts/go_tasks.py fmt-check

vet: ## Run go vet across all modules
	$(TOOL_ENV) uv run python scripts/go_tasks.py vet

lint: ## Run golangci-lint across all modules (Uber Go Style Guide orientation)
	$(TOOL_ENV) uv run python scripts/go_tasks.py lint

test: ## Run Go tests across all modules
	$(TOOL_ENV) uv run python scripts/go_tasks.py test

contract-check: ## Verify the registered HTTP route inventory
	$(TOOL_ENV) go -C $(SERVER_DIR) test ./pkg/httpapi -run '^TestActiveRouteInventory$$'

test-integration: db-up ## Run DB-backed Go integration tests
	$(TOOL_ENV) env DATABASE_URL=postgres://iroha:iroha_dev@127.0.0.1:5432/iroha?sslmode=disable go -C $(SERVER_DIR) test -tags=integration ./...

scripts-test: ## Run Python script unit tests
	$(TOOL_ENV) uv run python -m unittest discover -s scripts -p '*_test.py'

build: ## Build all Go modules
	$(TOOL_ENV) uv run python scripts/go_tasks.py build

run: db-up ## Run the server against the local dev stack (http://127.0.0.1:8080)
	$(TOOL_ENV) go -C $(SERVER_DIR) run ./cmd/iroha-server

run-job: db-up ## Run one iroha-job polling worker against the local dev stack
	$(TOOL_ENV) go -C $(JOB_DIR) run .

export-public: db-up ## Export sanitized public data as static JSON/GeoJSON (OUT=./dist/public-data)
	$(TOOL_ENV) go -C $(SERVER_DIR) run ./cmd/iroha-export-public --out $(abspath $(OUT))

media-bridge-build: ## Build the Bangumi->MAL->AniList bridge cache (MEDIA_BRIDGE_OUT=./dist/media-bridge)
	$(TOOL_ENV) uv run python scripts/build_media_bridge.py --out $(MEDIA_BRIDGE_OUT)

## --- Web frontend (apps/iroha-web, bun) ---
web-install: ## Install web dependencies
	cd $(WEB_DIR) && $(TOOL_ENV) bun install

web-fmt: ## Format the web app (prettier + prettier-plugin-svelte: spaces, double quotes)
	cd $(WEB_DIR) && $(TOOL_ENV) bun run format

web-fmt-check: ## Fail if any web file is unformatted
	cd $(WEB_DIR) && $(TOOL_ENV) bun run format:check

web-check: ## Type-check the web app (svelte-check)
	cd $(WEB_DIR) && $(TOOL_ENV) bun run check

web-test: ## Run unit tests for the web app (vitest)
	cd $(WEB_DIR) && $(TOOL_ENV) bun run test

web-build: ## Production build of the web app
	cd $(WEB_DIR) && PUBLIC_IROHA_VERSION=$(VERSION) $(TOOL_ENV) bun run build

web-dev: ## Run the web dev server, bound to all interfaces (Tailscale/LAN)
	cd $(WEB_DIR) && PUBLIC_IROHA_VERSION=$(VERSION) $(TOOL_ENV) bun run dev --host 0.0.0.0

web-visual-install: ## One-time: install the browser binary for agent-browser visual checks
	@command -v agent-browser >/dev/null || (echo "agent-browser is required; install it before running this target" >&2; exit 1)
	agent-browser install

web-visual-check: ## Screenshot a themed route with agent-browser (THEME=field-journal, ROUTE=overview, BASE=...)
	@command -v agent-browser >/dev/null || (echo "agent-browser is required; install it before running this target" >&2; exit 1)
	cd $(WEB_DIR) && BASE="$(or $(BASE),http://127.0.0.1:5173)" THEME="$(or $(THEME),field-journal)" ROUTES="$(or $(ROUTE),overview)" OUT="$(or $(OUT),.visual-check)" bash scripts/visual-check.sh

## --- Public static site (apps/iroha-public-site, bun) ---
public-site-install: ## Install public-site dependencies
	cd $(PUBLIC_SITE_DIR) && $(TOOL_ENV) bun install

public-site-fmt-check: ## Fail if any public-site file is unformatted
	cd $(PUBLIC_SITE_DIR) && $(TOOL_ENV) bun run format:check

public-site-check: ## Type-check the public site (svelte-check)
	cd $(PUBLIC_SITE_DIR) && $(TOOL_ENV) bun run check

public-site-build: ## Production build of the public site (BASE_PATH=/iroha for GitHub Pages)
	cd $(PUBLIC_SITE_DIR) && $(TOOL_ENV) bun run build

public-site-dev: ## Run the public-site dev server, bound to all interfaces
	cd $(PUBLIC_SITE_DIR) && $(TOOL_ENV) bun run dev -- --host 0.0.0.0 --port $(or $(PORT),5174)

public-site-preview: ## Build and serve the public site locally (root path, production output)
	cd $(PUBLIC_SITE_DIR) && $(TOOL_ENV) bun run build && $(TOOL_ENV) bun run preview -- --host $(or $(HOST),127.0.0.1) --port $(or $(PORT),4173)

## --- Docs and config formatting (prettier; Go/web/SQL out of scope) ---
fmt-docs: ## Format docs and config files (markdown wraps at 200)
	$(TOOL_ENV) $(PRETTIER) --write $(DOCS_FILES)

fmt-docs-check: ## Fail if any doc/config file is unformatted
	$(TOOL_ENV) $(PRETTIER) --check $(DOCS_FILES)

## --- Aggregate gates ---
check: fmt-check vet lint test contract-check scripts-test web-fmt-check web-check web-test ## Pre-commit gate: fmt-check + vet + lint + test + contract route check + script tests + web checks
validate: check build web-build ## Pre-PR gate: check + full server and web builds

## --- Dev stack (Podman Compose via uv scripts) ---
dev-up: ## Start the complete local stack and apply migrations
	$(TOOL_ENV) uv run python scripts/dev_stack.py start

dev-watch: ## Rebuild changed Podman Compose application services
	$(TOOL_ENV) uv run python scripts/dev_watch.py

db-up: ## Start only Postgres/PostGIS and Valkey, then apply migrations
	$(TOOL_ENV) uv run python scripts/dev_stack.py deps

db-down: ## Stop the local dev database
	$(TOOL_ENV) uv run python scripts/dev_stack.py stop

db-status: ## Show local dev database stack status
	$(TOOL_ENV) uv run python scripts/dev_stack.py status

db-logs: ## Show local dev database logs
	$(TOOL_ENV) uv run python scripts/dev_stack.py logs

db-reset: ## Reset the local dev database stack and apply migrations
	$(TOOL_ENV) uv run python scripts/dev_stack.py reset

smoke-real-import: ## Upload/import a real local file through the HTTP API (FILE=...)
	@test -n "$(FILE)" || (echo "FILE is required, e.g. make smoke-real-import FILE=.iroha-data/imports/export.zip" >&2; exit 2)
	$(TOOL_ENV) uv run python scripts/real_import_smoke.py "$(FILE)" --api-base "$(or $(API_BASE),http://127.0.0.1:8080)"

smoke-local: ## Run real import smoke against the Podman Compose server and worker
	@test -n "$(FILE)" || (echo "FILE is required, e.g. make smoke-local FILE=.iroha-data/imports/export.zip" >&2; exit 2)
	$(TOOL_ENV) uv run python scripts/real_import_smoke.py "$(FILE)" --assert --api-base "$(or $(API_BASE),http://127.0.0.1:8080)"

soak-local: ## Run non-mutating HTTP soak checks against the Podman Compose stack
	$(TOOL_ENV) uv run python scripts/local_stack_soak.py $(SOAK_ARGS)

## --- k3s local images (build with Podman, import straight into containerd; no registry) ---
image-server: ## Build iroha-server and import it into the local k3s containerd store (TAG=$(TAG))
	podman build --target server -t $(IMAGE_NS)/iroha-server:$(TAG) -f ops/images/Containerfile.server .
	podman save $(IMAGE_NS)/iroha-server:$(TAG) | sudo k3s ctr images import -

image-job: ## Build iroha-job and import it into the local k3s containerd store (TAG=$(TAG))
	podman build --target job -t $(IMAGE_NS)/iroha-job:$(TAG) -f ops/images/Containerfile.server .
	podman save $(IMAGE_NS)/iroha-job:$(TAG) | sudo k3s ctr images import -

image-db-migrate: ## Build iroha-db-migrate and import it into the local k3s containerd store (TAG=$(TAG))
	podman build --target db-migrate -t $(IMAGE_NS)/iroha-db-migrate:$(TAG) -f ops/images/Containerfile.server .
	podman save $(IMAGE_NS)/iroha-db-migrate:$(TAG) | sudo k3s ctr images import -

image-web: ## Build iroha-web and import it into the local k3s containerd store (TAG=$(TAG))
	podman build -t $(IMAGE_NS)/iroha-web:$(TAG) -f ops/images/Containerfile.web --build-arg PUBLIC_IROHA_API_BASE= --build-arg PUBLIC_IROHA_VERSION=$(VERSION) .
	podman save $(IMAGE_NS)/iroha-web:$(TAG) | sudo k3s ctr images import -

image-export-public: ## Build iroha-export-public and import it into the local k3s containerd store (TAG=$(TAG))
	podman build --target export-public -t $(IMAGE_NS)/iroha-export-public:$(TAG) -f ops/images/Containerfile.server .
	podman save $(IMAGE_NS)/iroha-export-public:$(TAG) | sudo k3s ctr images import -

images: image-server image-job image-db-migrate image-web image-export-public ## Build and import all iroha images into the local k3s containerd store
