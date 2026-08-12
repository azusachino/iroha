# Iroha v0.4 Expense Ledger Plan v4

> Status: implementation baseline. This document records the current decisions and frontend boundary for Iroha v0.4.

## Current architecture

Iroha is a deterministic canonical data service. It accepts structured expense JSON, validates it, stores it, deduplicates it, and serves deterministic expense records and monthly reports.

Iroha is the central personal-data cockpit. Expense storage and reporting belong to Iroha; no Telegram bot or companion service owns an expense domain.

The current v0.4 client is the local agent CLI:

```text
Local agent CLI -> canonical expense JSON -> Iroha API
```

Telegram is intentionally not a v0.4 expense client. A future client may use the same API, but it is outside this plan.

For a local receipt workflow:

```text
receipt.jpg -> local agent OCR/vision -> canonical JSON -> Iroha API
```

OCR, model choice, temporary image files, and any optional preview/confirmation are client concerns. Iroha has no OCR worker, agent state machine, candidate model, or confirmation endpoint.

## Frontend sharing boundary

`packages/iroha-shared/` is the source-only common library for code that is identical between `apps/iroha-web` and `apps/iroha-public-site`. It currently owns calendar helpers, the month navigator,
and the shared statistic tile. Private API clients, report/expense contracts, authentication, public-export policy, and design-language registries remain in their owning app. Move a component into the
common library only when both applications use the same behavior and accessibility contract; do not use it to blur the private cockpit/public archive boundary.

## Suzuran boundary

Suzuran's expense feature is explicitly abandoned for this initiative. It is not a migration target, not a report consumer, and not a second Iroha client for expenses.

The existing Suzuran expense code and local table are an unadopted legacy implementation. This plan does not backfill it, preserve it, dual-write it, or add new behavior around it. Any future cleanup
belongs to a separate Suzuran task. A future Telegram expense client, if desired, must be designed separately and call Iroha directly.

## Current decisions

1. Iroha accepts canonical JSON only; it does not parse natural-language expense text.
2. The local agent uses one general Iroha CLI for receipt images and other local inputs; expenses are one CLI resource, not a separate tool.
3. A future Telegram client, if approved, is a separate direct Iroha client; it is not part of Suzuran's expense feature.
4. Iroha does not require user confirmation. A client may offer a preview/confirmation mode, but the API accepts a valid canonical request directly.
5. The v0.4 storage model is one `tb_expenses` table. There are no Iroha intake, candidate, revision, mutation, or OCR tables.
6. That canonical `tb_expenses` table belongs to Iroha. Suzuran does not own, mirror, migrate, or report expenses.
7. The required canonical fields are `occurred_on`, `currency`, `amount_minor`, `category`, and `source`. `merchant`, `note`, and `items` are optional.
8. `amount_minor` is authoritative. `items` are descriptive receipt details and are not accounting line items.
9. `(source.kind, source.ref)` is immutable create identity, and `create_fingerprint` stores the normalized original request. Same identity plus the same original fingerprint returns the current
   record; a different fingerprint returns `409 Conflict`.
10. A deleted record remains addressable by its source identity and a repeated create returns `410 Gone`; it cannot be recreated. v0.4 does not provide revision history.
11. Receipt images and OCR evidence stay in the local agent environment. Iroha stores no image bytes or model prompt.
12. The monthly cross-domain report is a separate read feature. It aggregates existing personal data sections without combining incompatible units.
13. Iroha remains private-network-only under the current deployment model. Full authentication is required before external exposure.
14. Refunds and credits are out of scope for v0.4; `amount_minor` is a positive spending amount only.
15. Currency metadata is a static Go map in v0.4, initially `JPY`, `USD`, `EUR`, and `GBP`; no currency table is added.

## User stories

### Local agent entry

As a user, I can give a receipt image to a local agent and have it submit a canonical expense without filling in a table.

As a user, I can optionally review or edit the extracted merchant, note, date, or item list before submission.

As a user, I can list expenses, see monthly totals, edit an expense, or undo it through a client using the Iroha API.

### Local agent receipt entry

As a user, I can give a receipt image to a local agent. The agent extracts fields locally and submits the canonical JSON to Iroha.

As a user, I can run the agent in preview mode before submission, but this is optional and is not an Iroha protocol step.

As an operator, I can run the same extraction and submission workflow from a local CLI.

### Deterministic ledger

As any client, I can submit one canonical JSON document and receive the same expense record on a safe retry.

As a user, I can see expenses in the same monthly report surface as my activity, sleep, daily health, and media data.

As a user, I can inspect canonical expenses and delete an incorrect record in Iroha's private web cockpit. Canonical field correction is an API/CLI/agent operation, not an inline web-page workflow.

## Scope and non-goals

In scope:

- One canonical expense record with date, money, category, merchant, note, optional items, and source reference.
- Deterministic create, list, get, update, and delete APIs, with aggregation owned by the monthly report.
- Source-based idempotency for external clients.
- A general local Iroha CLI with expense commands, monthly reports, and read access to existing canonical domains.

Out of scope for v0.4:

- Budgets, bank/card synchronization, tax rules, reconciliation, and FX conversion.
- Receipt line-item accounting. The item list is descriptive; the top-level total is authoritative.
- Voice input, autonomous recurring-expense inference, and an always-on model daemon.
- OCR or an LLM dependency inside Iroha.
- Receipt bytes or OCR evidence stored in Iroha.
- Mandatory client confirmation or an Iroha confirmation workflow.

## Canonical data format

Iroha accepts structured JSON only. Natural-language text such as `800 JPY food ramen` belongs to a client parser and is never an Iroha request body.

### Create request

```http
POST /api/v1/expenses
Content-Type: application/json
```

```json
{
  "occurred_on": "2026-08-12",
  "currency": "JPY",
  "amount_minor": 1300,
  "category": "food",
  "merchant": "Ramen Shop",
  "note": "Lunch",
  "items": [
    { "name": "Ramen", "amount_minor": 800 },
    { "name": "Gyoza", "amount_minor": 500 }
  ],
  "source": {
    "kind": "local_agent",
    "ref": "receipt-2026-08-12-001"
  }
}
```

### Field rules

- `occurred_on` is required and uses `YYYY-MM-DD`. The client asks the user when the date is uncertain; Iroha does not infer it from prose.
- `currency` is required, uppercase, and three letters. v0.4 uses a static Go currency/exponent map (initially `JPY`, `USD`, `EUR`, and `GBP`) and performs no conversion or currency-table migration.
  Responses include the derived `currency_exponent` for display.
- `amount_minor` is required, positive, and an integer. JPY `1300` is `1300`; USD `$13.00` is `1300`. Floating-point amounts are rejected.
- `category` is required and must be one of `food`, `groceries`, `transport`, `shopping`, `housing`, `utilities`, `health`, `entertainment`, `subscriptions`, `work`, or `other`.
- `merchant` and `note` are optional bounded strings. Empty strings normalize consistently to null or the database default.
- `items` is optional, preserves input order, and has at most 32 entries. Each item has a required non-empty `name` of at most 200 characters and an optional non-negative `amount_minor`, which means
  the line total in the parent expense currency. Item amounts are descriptive and need not sum to the total because tax, discounts, tips, and rounding exist.
- `source` is required. `kind` identifies the client (`local_agent`, `cli`, or a separately approved future client); `ref` is an opaque stable identifier and must not contain a local filesystem path
  or receipt contents.

The top-level amount is always authoritative for reports. The item list is for display and later refinement, not double-entry accounting.

## Iroha storage and API

Add one explicit Goose migration using the next available number in `apps/iroha-server/db/migrations/` when implementation starts. Do not add workflow tables.

`tb_expenses` contains:

- UUIDv7 `id`.
- `occurred_on date not null`.
- `currency text not null`, `amount_minor bigint not null`, and a positive-value check.
- `category text not null` with application validation and a database check where practical.
- `merchant text not null default ''`, `note text not null default ''`.
- `items_json jsonb not null default '[]'`.
- `source_kind text not null`, `source_ref text not null`.
- `create_fingerprint text not null` covering the normalized original create request.
- `created_at`, `updated_at`, and nullable `deleted_at`.
- Unique `(source_kind, source_ref)` plus indexes for date/currency/category and active records.

Endpoints:

- `POST /api/v1/expenses` returns `201 Created` for a new record, `200 OK` for an identical retry, `409 Conflict` for a different original fingerprint, and `410 Gone` for a deleted source identity.
- `GET /api/v1/expenses` lists active records with `from` inclusive, `to` exclusive, category/currency filters, and cursor pagination. Order is `occurred_on DESC, id DESC`.
- `GET /api/v1/expenses/{expenseId}` returns an active record; a deleted record returns `410 Gone`.
- `PUT /api/v1/expenses/{expenseId}` replaces all editable fields (`occurred_on`, `category`, `merchant`, `note`, `items`, `amount_minor`, `currency`) and validates the complete record. Source
  identity and create fingerprint cannot change. v0.4 uses explicit last-write-wins; there is no revision or concurrency-token protocol.
- `DELETE /api/v1/expenses/{expenseId}` sets `deleted_at` and returns `204 No Content`; repeating it also returns `204`.

Create behavior is deterministic: validate, normalize, fingerprint the original request, insert, and handle the unique source conflict. On a conflict, compare the stored create fingerprint—not the
mutable current row. Successful writes invalidate any expense-list cache; monthly reports are not cached in v0.4.

Expense responses include the read-only `currency_exponent` derived from the static currency map. It is not accepted in create or update requests and is never used to convert between currencies.

The API does not accept a generic `text` field, call external services, or require an `Idempotency-Key` header in addition to the stable source identity. Manual clients generate and persist a UUID
source reference before the first request. Receipt agents use a receipt SHA-256 as the default source reference and allow an explicit override for two legitimate expenses from identical images. The
CLI writes the generated source reference into the draft before sending; automatic retries reuse that draft and never generate a new reference.

## Independent local agent workflow

The local agent is a separate client, not a Telegram component. It may be a shell command, coding-agent tool, or small local program:

```text
local receipt image
  -> local OCR/vision
  -> validate draft against expense schema
  -> optional human preview
  -> POST /api/v1/expenses
```

The agent may use any local model or approved provider. Iroha does not know or care which one. Missing or uncertain values remain missing/uncertain; the agent must not silently invent them.

Create one general `scripts/iroha_cli.py`. It is a thin Iroha client with shared transport, configuration, JSON/table output, and resource subcommands. It must not become an OCR engine or reimplement
report aggregation.

The v0.4 command surface is intentionally small:

```bash
uv run python scripts/iroha_cli.py expense create --input draft.json
uv run python scripts/iroha_cli.py expense list --from 2026-08-01 --to 2026-09-01
uv run python scripts/iroha_cli.py expense get exp_01k...
uv run python scripts/iroha_cli.py expense update exp_01k... --input replacement.json
uv run python scripts/iroha_cli.py expense delete exp_01k...
uv run python scripts/iroha_cli.py report monthly --month 2026-08 --timezone Asia/Tokyo
```

The initial supported resources are `expense` (read/write) and `report monthly` (read). The CLI uses `IROHA_API_BASE`, requires `--timezone` or `IROHA_TIMEZONE` for report requests, emits JSON by
default, supports `--format table` only for the explicitly supported resources, and never stores or prints receipt images. `expense create` uses server validation; there is no duplicated client-side
`validate` command in v0.4. Read-only wrappers for activity, sleep, daily, and media are a later CLI slice after their response contracts are tightened. No domain receives write commands until it has
a stable write API.

For an image-aware local agent, OCR/vision and the Iroha CLI may be one command or two commands. That packaging choice is local and must not leak into the HTTP contract.

## Private cockpit UX

Iroha's web cockpit provides the v0.4 management surface for canonical expenses:

```text
/expenses
  -> list active records with month/category/currency filters
  -> open one record
  -> inspect the immutable canonical fields
  -> delete it with DELETE and return to the filtered list
/reports
  -> select a month
  -> open the expense section and link to the matching /expenses filter
```

The page renders server data and owns no aggregation or OCR logic. Browser mutation support requires `PUT`, `DELETE`, and `OPTIONS` in the API CORS allow-list. The web obtains an IANA timezone from
the browser for monthly report requests and sends it explicitly.

## Implementation slices

1. **Canonical contract:** migration, runtime model/ID prefix, validation, create/list/get/update/delete service, source uniqueness, OpenAPI, and API tests.
2. **Deterministic reporting:** implement the separate [monthly report plan](2026-08-12-periodic-reports.md) and expense aggregate section.
3. **General CLI foundation:** shared transport, configuration, source-reference persistence, JSON output, and error handling.
4. **General CLI v0.4 resources:** `expense create/list/get/update/delete` and `report monthly`; add read-only wrappers for activity, sleep, daily, and media only after their response contracts are
   explicit.
5. **Private cockpit:** `/expenses` visual aggregation plus read-only list/detail/delete flows with one month-by-month selector, rendered through all six design-language shells. Canonical corrections
   remain on the API/CLI/agent path. The report page links to the same canonical month and remains the sole owner of expense aggregation.
6. **Release hardening:** migration rehearsal, monitoring, docs, OpenAPI, and v0.4 release note. Future external clients are separate follow-up work.

Each slice must remain deterministic inside Iroha and must not depend on a particular client being available.

## Verification

- Validate required date/currency/amount/category/source, item shape, limits, and unsupported values.
- Test identical source retry, conflicting source retry (`409`), concurrent duplicate creates, updates, tombstones, and deleted filtering.
- Test aggregates across date boundaries, currencies, categories, deleted rows, and empty periods.
- Test OpenAPI and route inventory, public-export exclusion, cache invalidation, browser mutation CORS, and CLI JSON/error behavior.
- Test the general CLI's source-reference persistence, expense payload/retry/conflict behavior, monthly report transport, and only the read-only domain commands whose contracts are stable.
- Run `make fmt-docs-check` and `make check`; use `make test-integration` when the database is available.
