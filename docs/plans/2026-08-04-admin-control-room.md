# Admin Control Room Implementation Plan

> **For Claude:** Use `${SUPERPOWERS_SKILLS_ROOT}/skills/collaboration/executing-plans/SKILL.md` to implement this plan task-by-task.

**Goal:** Deliver a private control room for personal daily tasks and allowlisted Iroha job actions, make it discoverable from the front page, and ship a visible `v0.1.3` release marker without
introducing CDN or asset infrastructure.

**Architecture:** Add a small first-class personal-task table and service, while exposing the existing durable `tb_jobs` queue through read endpoints. Browser actions call named server actions for
media syncs rather than arbitrary job kinds. The front page reads due tasks from the same task API and links to the control room.

**Tech Stack:** Go 1.26, Chi, GORM/Postgres migrations, Svelte 5, TypeScript, Lucide icons, Vitest, Go tests, Make.

---

### Task 1: Add the personal task model and service

**Files:**

- Create: `apps/iroha-server/db/migrations/00003_tasks.sql`
- Modify: `apps/iroha-runtime/models/models.go`
- Create: `apps/iroha-server/pkg/tasks/service.go`
- Create: `apps/iroha-server/pkg/tasks/service_test.go`

**Steps:**

1. Write service tests for listing open tasks, creating a task, completing a task, and preserving due-date ordering.
2. Run `go test ./apps/iroha-server/pkg/tasks` and confirm the new package fails to compile.
3. Add `tb_tasks` with UUIDv7 IDs, title, notes, status, due date, priority, source, created/updated/completed timestamps, and indexes for status/due date.
4. Add the matching runtime model and a service with DB-free input normalization helpers where practical.
5. Run the task package tests and the migration-aware integration test if available.
6. Commit as `feat: add personal task storage`.

### Task 2: Expose tasks and durable jobs through safe HTTP actions

**Files:**

- Modify: `apps/iroha-server/pkg/httpapi/server.go`
- Create: `apps/iroha-server/pkg/httpapi/tasks.go`
- Create: `apps/iroha-server/pkg/httpapi/jobs.go`
- Modify: `apps/iroha-server/pkg/httpapi/api_contract_test.go`
- Modify: `docs/contracts/openapi.yaml`
- Modify: `apps/iroha-server/cmd/iroha-server/main.go`

**Steps:**

1. Add handler tests for task list/create/complete, job list/detail, and rejected arbitrary action names.
2. Run the focused HTTP tests and confirm the new routes fail before implementation.
3. Add `GET/POST /api/v1/tasks`, `PATCH /api/v1/tasks/{taskId}`, `GET /api/v1/jobs`, `GET /api/v1/jobs/{jobId}`, and `POST /api/v1/actions/{action}`.
4. Allow only named actions such as `media-sync-anilist` and `media-sync-bangumi`; map them to existing job constants.
5. Return existing durable job timestamps, attempts, retry time, and error message without exposing lock ownership or secrets.
6. Register the routes, update the active inventory and OpenAPI contract, and run focused tests.
7. Commit as `feat: expose control room task and job APIs`.

### Task 3: Build the control room and front-page To-go card

**Files:**

- Modify: `apps/iroha-web/src/lib/api.ts`
- Create: `apps/iroha-web/src/routes/admin/+page.svelte`
- Modify: `apps/iroha-web/src/routes/+page.svelte`
- Modify: `apps/iroha-web/src/lib/components/CommandPalette.svelte`
- Modify: `apps/iroha-web/src/routes/+layout.svelte`
- Modify: `apps/iroha-web/src/lib/api.test.ts`

**Steps:**

1. Add API client tests for task CRUD, job listing, and named action POSTs.
2. Implement the API client types and methods.
3. Build `/admin` with Today tasks, add/complete controls, quick actions, active jobs, failed-job retry links, and a quiet version note.
4. Poll active jobs only while they are queued or running; stop polling for terminal states.
5. Add a front-page To-go card showing due/open tasks and a link to the control room.
6. Add an Admin/Control room command-palette entry and a small Lucide icon in navigation or the card; keep the existing flower brand mark unchanged.
7. Run `make web-test`, `make web-check`, and browser-test the page in light/dark mode at desktop and mobile widths.
8. Commit as `feat: add private control room UI`.

### Task 4: Bump and surface the release version

**Files:**

- Modify: `Makefile`
- Modify: `apps/iroha-web/package.json`
- Modify: `apps/iroha-public-site/package.json`
- Modify: `apps/iroha-web/src/lib/config.ts`
- Modify: `apps/iroha-web/src/routes/+layout.svelte`
- Modify: `README.md`

**Steps:**

1. Choose `v0.1.3` as the feature release following the deployed `v0.1.2` line.
2. Update the image tag and package versions without touching parser versions or Go module versions.
3. Expose the version in the app footer/control-room note, keeping it visually secondary.
4. Add a short README release note describing the control room and explicit CDN deferral.
5. Run formatting and version-reference searches to ensure the old release tag is not left in active build targets.
6. Commit as `chore: bump iroha release version`.

### Task 5: Verify, browser-check, and hand off

**Files:**

- Modify only files already listed above if fixes are required.

**Steps:**

1. Run `make check` and read the complete result.
2. Run `make validate` and read the complete result.
3. Run the local browser against `/`, `/admin`, and the existing `/media` route; verify task creation/completion, action enqueue, job state rendering, and responsive layout.
4. Review `git diff`, `git status`, and the branch name; preserve the pre-existing sleep-detail edits.
5. Stage only files changed for this feature, rerun the relevant checks, and commit the final feature changes on the current feature branch.
