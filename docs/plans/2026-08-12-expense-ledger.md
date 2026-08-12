# Iroha v0.4 Expense Ledger Plan v2

> Status: draft for review. This is the simplified implementation contract; it does not authorize implementation or deployment.

## Decision in one sentence

Local agents read receipt images and produce structured JSON; Telegram presents and confirms that draft; Iroha accepts only canonical expense JSON and deterministically validates, stores, aggregates,
and reports it.

```text
Telegram or CLI -> local agent (optional OCR/vision) -> user confirmation -> Iroha API -> reports
```

Iroha does not run OCR, invoke an LLM, store agent candidates, manage intake state, or own Telegram conversations.

## What changed from v1

The following are removed from the v0.4 design:

- `tb_expense_intakes` and `tb_expense_candidates`.
- Iroha-side intake/candidate state machines.
- An Iroha canonicalization job or OCR worker.
- Separate revision and mutation tables.
- Natural-language expense text in the Iroha API.
- Receipt upload/download through Iroha.

The v0.4 ledger has one canonical table and a small stable API. Agent drafts live in the local agent or Suzuran's short-lived conversation state and disappear after confirmation, cancellation, or
expiry.

## User stories

### Quick Telegram expense

As a user, I can type `/expense 1300 JPY`, choose a category, optionally enter a merchant or note, and confirm a compact preview without filling in a table.

As a user, I can retry the same Telegram update without creating a duplicate expense.

As a user, I can list today's or this month's expenses and correct or undo one from Telegram.

### Receipt through a local agent

As a user, I can send a receipt photo to Telegram. A local agent reads it, and Telegram shows me the extracted date, total, currency, category, merchant, and item list before saving.

As a user, I can confirm, edit, or cancel the draft. An OCR or model error never becomes a ledger entry without my confirmation.

As an operator, I can run the same extraction and submission workflow from a local CLI without Telegram.

### Deterministic ledger

As a client, I can submit one canonical JSON document and receive the same expense record on a safe retry.

As a user, I can see weekly and monthly totals grouped by category and currency, with an explicit period boundary.

As a privacy-conscious user, I can rely on expense records remaining out of the public export by default.

## Scope and non-goals

In scope:

- One canonical expense record with date, money, category, merchant, note, optional item list, and source reference.
- Deterministic create, list, get, update, delete, and aggregate APIs.
- Source-based idempotency for external clients.
- A local CLI for validating/submitting agent JSON and reading reports.
- Telegram conversational UX for quick expenses, receipt confirmation, listing, correction, and undo.
- Suzuran migration from its local `tb_expenses` table to Iroha.

Out of scope for v0.4:

- Budgets, bank/card synchronization, tax rules, reconciliation, and FX conversion.
- Receipt line-item accounting. The item list is descriptive; the top-level total is authoritative.
- Voice input, autonomous approval, recurring-expense inference, and an always-on model daemon.
- OCR or an LLM dependency inside Iroha.
- Receipt bytes or OCR evidence stored in Iroha.
- Partial bearer authentication. Iroha remains private-network-only under the current deployment model; full authentication is required before external exposure.

## Ownership boundaries

| Concern                                      | Owner                          | Required behavior                                                                                           |
| -------------------------------------------- | ------------------------------ | ----------------------------------------------------------------------------------------------------------- |
| Telegram messages, buttons, and conversation | Suzuran                        | Collect fields, show previews, handle confirmation/edit/cancel.                                             |
| Receipt OCR/vision                           | Configured local agent command | Read the image, return a draft JSON document, never call Iroha directly without confirmation.               |
| Draft lifetime                               | Suzuran/agent local state      | Store only long enough to confirm; use existing Valkey conversation state with a short TTL where available. |
| Canonical validation                         | Iroha                          | Validate every field and reject malformed or unsupported data.                                              |
| Canonical storage                            | Iroha                          | Store only confirmed client submissions.                                                                    |
| Idempotency                                  | Iroha database constraint      | Use `(source.kind, source.ref)` as the stable external identity.                                            |
| Aggregation/reporting                        | Iroha                          | Query active records deterministically; no client-side reimplementation of totals.                          |
| Image retention                              | Local deployment               | Clean temporary files after extraction or cancellation; Iroha never receives the image.                     |

## Canonical data format

Iroha accepts structured JSON only. Natural-language text such as `800 JPY food ramen` belongs to the Telegram/client layer and is never an Iroha request body.

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
    "kind": "telegram",
    "ref": "photo:12345:67890"
  }
}
```

### Field rules

- `occurred_on` is required and uses `YYYY-MM-DD`. The client asks the user when the date is uncertain; Iroha does not infer it from a free-form string.
- `currency` is required, uppercase, and three letters. v0.4 supports a small explicit currency metadata table for formatting; it performs no conversion between currencies.
- `amount_minor` is required, positive, and an integer. JPY `1300` is `1300`; USD `$13.00` is `1300`. Floating-point amounts are rejected.
- `category` is required and must be one of `food`, `groceries`, `transport`, `shopping`, `housing`, `utilities`, `health`, `entertainment`, `subscriptions`, `work`, or `other`.
- `merchant` and `note` are optional strings with length limits. Empty strings normalize to null or the database default consistently.
- `items` is optional. Each item has a required non-empty `name` and an optional non-negative `amount_minor`. Item amounts are descriptive and are not required to sum to the top-level total because
  tax, discounts, tips, and rounding exist.
- `source` is required. `kind` identifies the client (`telegram`, `local_agent`, `cli`, or `suzuran_legacy`); `ref` is an opaque stable identifier and must not contain a local filesystem path or
  receipt contents.

The top-level amount is always authoritative for reports. The item list is for display and later refinement, not double-entry accounting.

### Response

The create response is the canonical stored record:

```json
{
  "id": "exp_01k...",
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
    "kind": "telegram",
    "ref": "photo:12345:67890"
  },
  "created_at": "2026-08-12T10:30:00Z",
  "updated_at": "2026-08-12T10:30:00Z",
  "deleted_at": null
}
```

## Iroha storage

Add one explicit Goose migration, using the next available number in `apps/iroha-server/db/migrations/` when implementation starts. Do not add intake, candidate, revision, or mutation tables.

`tb_expenses` should contain:

- UUIDv7 `id`.
- `occurred_on date not null`.
- `currency text not null` and `amount_minor bigint not null` with a positive-value check.
- `category text not null` with application validation and a database check where practical.
- `merchant text not null default ''` and `note text not null default ''`.
- `items_json jsonb not null default '[]'`.
- `source_kind text not null`, `source_ref text not null`.
- `created_at`, `updated_at`, and nullable `deleted_at`.
- Unique `(source_kind, source_ref)` for safe client retries.
- Indexes for `(occurred_on, currency)`, `(category, occurred_on)`, and active records.

Deletion is a tombstone (`deleted_at`), not a hard delete, so a Telegram undo cannot make a later retry recreate the same source event. `GET` and aggregates exclude deleted records by default. There
is no revision history in v0.4; a later audit requirement can add it deliberately.

## HTTP API

The active contract is `docs/contracts/openapi.yaml`. Route inventory tests and the OpenAPI document must change with the implementation.

### Endpoints

- `POST /api/v1/expenses` creates or returns the canonical record for `(source.kind, source.ref)`.
- `GET /api/v1/expenses` lists active records with `from`, `to`, `category`, `currency`, and pagination filters.
- `GET /api/v1/expenses/{expenseId}` returns one record, including `deleted_at` when explicitly requested.
- `PUT /api/v1/expenses/{expenseId}` replaces editable canonical fields after client confirmation. The full request body is validated again; the operation is naturally retryable with the same body.
- `DELETE /api/v1/expenses/{expenseId}` sets `deleted_at`; repeating the request is a no-op success.
- `GET /api/v1/expenses/aggregates?from=2026-08-01&to=2026-09-01&timezone=Asia/Tokyo` returns totals grouped by currency and category.

Create behavior:

1. Validate JSON shape and all field rules.
2. Normalize currency/category/strings and canonicalize the item list ordering only if the client contract requires it.
3. Insert using the unique source constraint.
4. On a duplicate source, compare the normalized request with the stored record. Return the existing record for an identical retry; return `409 Conflict` if the same source reference has different
   data.
5. Invalidate expense list, aggregate, report, and briefing caches after a successful create/update/delete.

The API does not accept a generic `text` field, does not parse dates/categories/prices from prose, and does not call external services. Every client must submit the same canonical shape.

## Local agent workflow

Receipt extraction is a deployment concern. The stable boundary is the draft JSON shape above minus the server-owned `id`, timestamps, and `deleted_at`.

The configured local extractor has a narrow process contract:

```text
expense-agent extract --input /path/to/receipt.jpg --output-json
  stdout: one JSON draft
  stderr: diagnostics only; never receipt text or image bytes
  exit 0: draft returned
  exit nonzero: no draft; caller falls back to manual entry
```

The extractor may use OCR, a local model, or an approved provider. Iroha does not know or care which one. The agent must map uncertain values to a user-visible draft and must not silently invent
missing values.

A thin Iroha client script, proposed as `scripts/expense_cli.py`, handles validation, preview, submission, and reports:

```bash
uv run python scripts/expense_cli.py validate --input draft.json
uv run python scripts/expense_cli.py submit --input draft.json
uv run python scripts/expense_cli.py list --from 2026-08-01 --to 2026-09-01
uv run python scripts/expense_cli.py report month 2026-08
uv run python scripts/expense_cli.py delete exp_01k...
```

It uses `IROHA_API_BASE`, emits JSON by default, supports `--format table` for humans, and never stores or prints the receipt image. The CLI does not become an autonomous daemon in v0.4.

## Telegram UX in Suzuran

Telegram is a friendly input and review surface, not a form-driven copy of the database. Suzuran owns the conversation and calls the stable Iroha API only after it has a complete confirmed payload.

### Entry points

#### Quick typed expense

Fast path:

```text
/expense 1300 JPY
```

Suzuran then presents category buttons (`Food`, `Groceries`, `Transport`, `Shopping`, `Bills`, `Health`, `Other`), asks optionally for merchant and note with a `Skip` button, and shows:

```text
Expense preview

¥1,300 · Food
Date: 2026-08-12
Merchant: —
Note: —

[Save] [Edit] [Cancel]
```

The user may also send `/expense` with no arguments. Suzuran asks for `amount currency` in one message, then uses the same buttons and preview. It must not send the intermediate prose to Iroha.

On `Save`, Suzuran submits the canonical JSON with:

```json
{ "kind": "telegram", "ref": "command:<chat_id>:<message_id>" }
```

The response message includes the Iroha expense ID, formatted amount, category, date, and an `Undo` button.

#### Receipt photo

1. User sends a photo or image document.
2. Suzuran downloads it to a private temporary file with a size/type limit.
3. Suzuran invokes the configured local `expense-agent extract` command.
4. The agent returns draft JSON; Suzuran validates the draft locally and stores only the draft plus chat/message metadata in existing Valkey conversation state with a short TTL. The image is deleted
   after extraction unless the configured agent needs a bounded retry.
5. Suzuran replies with a preview:

   ```text
   Receipt draft

   ¥1,300 · Food
   Date: 2026-08-12
   Merchant: Ramen Shop
   Items: Ramen ¥800, Gyoza ¥500

   [Confirm] [Edit] [Cancel]
   ```

6. `Confirm` submits the exact canonical JSON to Iroha using `source.ref = photo:<chat_id>:<message_id>`.
7. `Edit` asks for one field at a time (date, total/currency, category, merchant, items, note), re-renders the preview, and returns to Confirm/Edit/Cancel.
8. `Cancel`, TTL expiry, or successful submission removes the draft. A failed extractor produces a clear fallback: `I couldn't read this receipt. Send /expense to enter it manually.`

The callback data must contain only a short draft token, not the JSON payload. Callback handlers verify chat ownership before reading Valkey state. A bot restart may lose an unconfirmed draft; the
user can resend the photo, which is acceptable for v0.4 because no canonical record existed.

#### Listing and correction

- `/expenses` shows today's records and a total by currency, with buttons for `This week` and `This month`.
- `/expenses 2026-08` shows the monthly list and aggregate from Iroha.
- Each row has `Edit` and `Undo`. `Edit` runs the same field-by-field conversation and sends `PUT`; `Undo` confirms once and sends `DELETE`.
- Reports are fetched from Iroha's aggregate endpoint; Suzuran does not sum local rows.

### Telegram failure behavior

- If Iroha is unavailable, keep the confirmed payload in a bounded retry queue owned by Suzuran and show `Saved locally, waiting to sync`; retry with the same source reference.
- If the response is `409`, show the existing canonical record and do not create another one.
- If the agent is unavailable or returns invalid JSON, offer manual entry immediately.
- Never echo raw model diagnostics, receipt paths, provider errors, or image contents into Telegram logs.

## Suzuran cutover

Replace every direct local `tb_expenses` read/write path:

- `src/suzuran/expenses.py`: commands call Iroha list/create/aggregate APIs.
- `src/suzuran/callbacks.py`: undo calls Iroha delete.
- `src/suzuran/scheduler.py`: subscription renewal creates an Iroha expense with a deterministic source reference such as `subscription:<subscription_id>:<renewal_date>`.
- `src/suzuran/briefing.py`: weekly review and dashboard use Iroha reports.
- `src/suzuran/iroha.py`: add the typed expense client methods and error/idempotency handling.
- Add the local extractor command configuration and Valkey draft helpers in Suzuran; keep them out of Iroha.

Before cutover, backfill Suzuran's existing local rows into Iroha with `source.kind = suzuran_legacy` and `source.ref = legacy:<old_numeric_id>`. Verify row counts and totals by currency/date. Stop
local writes after the cutover; do not dual-write indefinitely.

## Representation and reporting

Add private expense representation only after the API is stable:

- Overview expense tile with current-period totals by currency.
- Category breakdown and recent expenses.
- Pending drafts are a Telegram/Suzuran concern and must not appear in Iroha totals.
- All existing Svelte themes receive the same typed expense contributor data.
- Public export remains expense-free by default, with a regression test covering a populated private ledger.

Aggregate responses include the resolved `[from, to)` range and timezone. There is no FX conversion; JPY, USD, and other currencies remain separate buckets.

## Implementation slices

1. **Canonical contract:** migration, runtime model/ID prefix, validation, create/list/get/update/delete service, source uniqueness, OpenAPI, and API tests.
2. **Deterministic reporting:** aggregate queries, cache invalidation, timezone/range tests, and public-export exclusion.
3. **Thin CLI:** validate/submit/list/report/delete commands and JSON/table output tests.
4. **Telegram typed UX:** quick command, conversational fields, preview, confirm/save, edit, undo, list, and report.
5. **Telegram receipt UX:** temporary image handling, configured extractor subprocess, Valkey draft TTL, preview/edit/confirm/cancel, and manual fallback.
6. **Suzuran cutover:** client migration, subscription/briefing paths, legacy backfill, no-local-write assertion, and Telegram smoke tests.
7. **Private UI and release hardening:** Overview/report views, all-theme wiring, migration rehearsal, monitoring, docs, and v0.4 release note.

Each slice should be independently testable. The Iroha slices must remain deterministic and must not depend on the local extractor or Telegram being available.

## Verification matrix

Iroha:

- Validation tests for required date/currency/amount/category/source, item shape, limits, and unsupported values.
- Create tests for identical source retry, same source with different payload (`409`), concurrent duplicate creates, and deleted-source behavior.
- Update/delete tests for full validation, tombstone filtering, and naturally retryable requests.
- Aggregate tests for date boundaries, timezone parameter, multiple currencies, categories, deleted rows, and empty periods.
- API route inventory and OpenAPI contract tests.
- Public-export regression test proving expenses remain private.
- `scripts/expense_cli.py` tests for validation, JSON output, HTTP errors, and no receipt logging.
- `make fmt-docs-check` and `make check`; use `make test-integration` when the database is available.

Suzuran:

- Conversation tests for quick entry, category buttons, optional fields, preview, save, edit, cancel, undo, and monthly report.
- Fake extractor tests for valid draft, malformed output, timeout, missing date, and missing total.
- Valkey draft ownership/TTL tests and callback-token tests.
- Iroha client tests for retry, `409`, unavailable-server queueing, and subscription source references.
- Backfill dry-run totals and post-cutover assertion that handlers no longer write local `tb_expenses`.
- `make check` and a private Telegram integration smoke test.

Acceptance flow:

1. Send `/expense 1300 JPY`, choose Food, save, and retry the same update.
2. Send a receipt photo, inspect the local-agent draft in Telegram, edit one field, and confirm it.
3. List the expense, edit it, undo it, and verify it disappears from reports.
4. Verify a month containing JPY and USD produces separate totals.
5. Verify public export contains no expense data.

## Approval gates

1. Iroha receives canonical JSON only; OCR/vision is outside Iroha.
2. The stable required fields are `occurred_on`, `currency`, `amount_minor`, `category`, `source`, with optional `merchant`, `note`, and `items`.
3. The top-level amount is authoritative; item lists are descriptive and optional.
4. Source identity `(source.kind, source.ref)` is the only v0.4 create-idempotency mechanism; no mutation table is needed.
5. Telegram owns preview and confirmation; an unconfirmed agent draft is never sent to Iroha.
6. Receipt images stay local and temporary; Iroha stores no raw image or OCR evidence.
7. Suzuran's local ledger is backfilled and retired; Iroha becomes the canonical ledger.
8. Typed Telegram entry is the first shippable vertical slice; receipt-agent UX follows without changing the Iroha API.
