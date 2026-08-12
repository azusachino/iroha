# Iroha v0.4 Expense Ledger Plan v3

> Status: draft for review. This document records current decisions and the implementation boundary; it does not authorize implementation or deployment.

## Current architecture

Iroha is a deterministic canonical data service. It accepts structured expense JSON, validates it, stores it, deduplicates it, and serves deterministic lists, aggregates, and reports.

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
9. `(source.kind, source.ref)` is the create idempotency identity. An identical retry returns the existing record; a conflicting retry returns `409 Conflict`.
10. Deletion is a `deleted_at` tombstone so a retry cannot recreate an intentionally removed source event. v0.4 does not provide revision history.
11. Receipt images and OCR evidence stay in the local agent environment. Iroha stores no image bytes or model prompt.
12. The monthly cross-domain report is a separate read feature. It aggregates existing personal data sections without combining incompatible units.
13. Iroha remains private-network-only under the current deployment model. Full authentication is required before external exposure.

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

## Scope and non-goals

In scope:

- One canonical expense record with date, money, category, merchant, note, optional items, and source reference.
- Deterministic create, list, get, update, delete, and aggregate APIs.
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
- `currency` is required, uppercase, and three letters. v0.4 supports an explicit currency metadata table for formatting and exponent validation; it performs no conversion.
- `amount_minor` is required, positive, and an integer. JPY `1300` is `1300`; USD `$13.00` is `1300`. Floating-point amounts are rejected.
- `category` is required and must be one of `food`, `groceries`, `transport`, `shopping`, `housing`, `utilities`, `health`, `entertainment`, `subscriptions`, `work`, or `other`.
- `merchant` and `note` are optional bounded strings. Empty strings normalize consistently to null or the database default.
- `items` is optional. Each item has a required non-empty `name` and an optional non-negative `amount_minor`. Item amounts are descriptive and need not sum to the total because tax, discounts, tips,
  and rounding exist.
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
- `created_at`, `updated_at`, and nullable `deleted_at`.
- Unique `(source_kind, source_ref)` plus indexes for date/currency/category and active records.

Endpoints:

- `POST /api/v1/expenses` creates or returns the record for `(source.kind, source.ref)`.
- `GET /api/v1/expenses` lists active records with date, category, currency, and pagination filters.
- `GET /api/v1/expenses/{expenseId}` returns one record; deleted records require an explicit include flag.
- `PUT /api/v1/expenses/{expenseId}` replaces editable fields after client-side editing. The full request is validated again.
- `DELETE /api/v1/expenses/{expenseId}` sets `deleted_at`; repeating it is a no-op success.
- `GET /api/v1/expenses/aggregates?from=2026-08-01&to=2026-09-01&timezone=Asia/Tokyo` returns currency-separated totals and category buckets.

Create behavior is deterministic: validate, normalize, insert, and handle the unique source conflict. On a conflict, identical normalized data returns the existing record; different data returns
`409`. Successful writes invalidate expense, report, and briefing read-cache namespaces.

The API does not accept a generic `text` field, call external services, or require an `Idempotency-Key` header in addition to the stable source identity. A client may send that header for tracing, but
source identity is the durable contract.

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

The v0.4 command surface is:

```bash
uv run python scripts/iroha_cli.py expense validate --input draft.json
uv run python scripts/iroha_cli.py expense create --input draft.json
uv run python scripts/iroha_cli.py expense list --from 2026-08-01 --to 2026-09-01
uv run python scripts/iroha_cli.py expense delete exp_01k...
uv run python scripts/iroha_cli.py report monthly --month 2026-08
uv run python scripts/iroha_cli.py activity list --from 2026-08-01 --to 2026-09-01
uv run python scripts/iroha_cli.py sleep aggregates --from 2026-08-01 --to 2026-09-01
uv run python scripts/iroha_cli.py daily aggregates --granularity month --from 2026-08-01 --to 2026-09-01
uv run python scripts/iroha_cli.py media aggregates
```

The initial supported resources are `expense` (read/write), `report` (read), `activity` (read), `sleep` (read), `daily` (read), and `media` (read). Do not add write commands for a domain until that
domain has a stable write API. The CLI uses `IROHA_API_BASE`, emits JSON by default, supports `--format table`, and never stores or prints receipt images.

For an image-aware local agent, OCR/vision and the Iroha CLI may be one command or two commands. That packaging choice is local and must not leak into the HTTP contract.

## Optional future client UX

Telegram/Suzuran is not part of the v0.4 implementation. If a separate client is approved later, it must submit canonical JSON directly to Iroha and must not introduce a second ledger.

### Example manual entry

```text
client command: expense 1300 JPY food
```

The client parses its own command, obtains the local date, and submits once required fields are present. Optional forms are:

```text
/expense 1300 JPY food Ramen Shop
/expense 1300 JPY food --date 2026-08-12 --merchant "Ramen Shop" --note "Lunch"
```

The exact parser should remain small. If a value is ambiguous, the client asks a focused follow-up question rather than sending prose to Iroha.

### Guided entry

An empty expense command may start a short conversation:

1. Ask for amount and currency.
2. Show category buttons.
3. Optionally ask for date, merchant, note, and item list; each has `Skip`.
4. Submit the canonical JSON directly when the required fields exist.

A preview and `Save/Edit/Cancel` buttons are optional UX. They are not required for correctness, and a client may provide a fast path that submits immediately.

### Listing and correction

- A client may list active records and totals by currency.
- A client may call the cross-domain monthly report API described in the monthly report plan.
- Each row may offer `Edit` and `Undo`. Edit sends `PUT`; Undo sends `DELETE` after an optional client-side confirmation.
- If Iroha returns `409`, the client displays the existing canonical record instead of creating another.
- If Iroha is unavailable, the client may retry the already-canonical JSON with the same `source.ref`; it must not write a second local ledger.

Telegram photo handling is not part of the agent workflow in v0.4. If a photo entry point is desired later, it should be designed as a separate Telegram-to-Iroha raw-evidence feature rather than
silently coupling Telegram to the local agent.

## Implementation slices

1. **Canonical contract:** migration, runtime model/ID prefix, validation, create/list/get/update/delete service, source uniqueness, OpenAPI, and API tests.
2. **Deterministic reporting:** implement the separate [monthly report plan](2026-08-12-periodic-reports.md) and expense aggregate section.
3. **General CLI:** shared transport plus expense, monthly-report, activity, sleep, daily, and media subcommands with JSON/table output tests.
4. **Optional external client:** quick entry, guided fields, optional preview, list, edit, undo, and report commands, only if separately approved.
5. **Optional external client:** implement a separate client only if approved; it must call Iroha directly and have no local expense storage.
6. **Private UI and release hardening:** Overview expense representation, all-theme wiring, migration rehearsal, monitoring, docs, and v0.4 release note.

Each slice must remain deterministic inside Iroha and must not depend on a particular client being available.

## Verification

- Validate required date/currency/amount/category/source, item shape, limits, and unsupported values.
- Test identical source retry, conflicting source retry (`409`), concurrent duplicate creates, updates, tombstones, and deleted filtering.
- Test aggregates across date boundaries, currencies, categories, deleted rows, and empty periods.
- Test OpenAPI and route inventory, public-export exclusion, cache invalidation, and CLI JSON/error behavior.
- Test the general CLI's expense payload/retry/conflict behavior, monthly report transport, and read-only domain commands.
- Test any approved external client for canonical payloads, retries, conflicts, and absence of local expense storage.
- Run `make fmt-docs-check` and `make check`; use `make test-integration` when the database is available.
