# Iroha v0.4 Expense Ledger Implementation Plan

> Status: draft for review. This document turns the v0.4 roadmap item into an implementation contract; it does not authorize implementation or deployment.

## Objective

Add a private expense ledger to iroha with a deliberately explicit pipeline:

```text
collect -> canonicalize -> aggregate -> represent -> report
```

Iroha owns the durable evidence boundary, canonical records, idempotency, corrections, and read/report APIs. Suzuran remains the Telegram client. An agent CLI may inspect receipt evidence and propose
a candidate, but an agent is never the ledger of record.

The first release should make typed expenses excellent and make receipt intake safe and reviewable. It should not turn iroha into a Telegram bot, an OCR platform, an accounting system, or a
foreign-exchange service.

## User stories

### Typed Telegram expense

As a user, I can send `/expense 800 JPY food ramen` to Telegram and receive a compact confirmation containing the amount, date, category, and description.

As a user, retrying the same Telegram update does not create a second expense.

As a user, I can correct or undo an expense later, and reports reflect the correction without destroying the original history.

### Receipt and agent-assisted intake

As a user, I can send a receipt image to Telegram and see that it is awaiting review rather than receiving an unverified ledger entry.

As an agent, I can pull the original receipt, run OCR or vision locally or through an approved provider, and submit a structured candidate with confidence and field-level evidence.

As a user, I can inspect and confirm, edit, or reject that candidate before it becomes a canonical expense.

As an operator, I can retry candidate submission safely and know whether the request was accepted, duplicated, or rejected as a conflicting retry.

### Reports and representation

As a user, I can see weekly and monthly totals grouped by category and currency, with an explicit timezone boundary.

As a user, I can distinguish a ledger total from activity, sleep, and media totals in the Overview and report views.

As a privacy-conscious user, I can rely on expenses being excluded from the public export unless a separately designed sanitized projection is explicitly enabled.

## Scope and non-goals

In scope:

- Text and receipt-image intake through external clients.
- A reusable raw-intake, candidate, confirmation, and canonical-record boundary.
- Integer minor-unit money, ISO-like currency codes, date-only expense dates, and auditable corrections.
- Weekly/monthly totals by category and currency.
- A small agent CLI for pull, propose, confirm, and report workflows.
- Suzuran migration from its local `tb_expenses` table to iroha's ledger.

Explicitly out of scope for v0.4:

- Budgets, account reconciliation, bank/card synchronization, tax rules, and FX conversion.
- Receipt line-item accounting; the first candidate is receipt-total only.
- Voice intake, autonomous recurring-expense inference, and an always-on agent daemon.
- OCR or an LLM dependency inside the iroha API or job worker.
- Adding partial bearer authentication to an otherwise private API. If iroha is exposed outside the trusted network, full API authentication is a release prerequisite.

## Existing boundaries to preserve

- Suzuran is the Telegram adapter and already has an Iroha client and an allowlist. It should call iroha rather than write the expense ledger directly.
- Iroha already has `tb_intake_payloads` for raw payload metadata/content and `tb_jobs` for durable asynchronous work. Reuse them instead of creating a second raw-upload or queue abstraction.
- Iroha's current private API relies on the network boundary. v0.4 keeps that deployment assumption explicit; no misleading half-authenticated endpoint should be added.
- Public export is a separate projection. Expense tables must not be joined into it by accident.
- The Svelte Overview has multiple curated themes. Any new expense representation must be wired through every theme's contributor data, not only the default theme.

## Pipeline contract

| Stage        | Owner                                        | Durable output                             | Boundary                                                                              |
| ------------ | -------------------------------------------- | ------------------------------------------ | ------------------------------------------------------------------------------------- |
| Collect      | Suzuran, CLI, future clients                 | `tb_intake_payloads`, `tb_expense_intakes` | Preserve original bytes/text and stable source event ID.                              |
| Canonicalize | Iroha deterministic parser or external agent | `tb_expense_candidates`                    | Candidate data is untrusted until confirmed.                                          |
| Aggregate    | Iroha query/service                          | SQL result, no second ledger               | Aggregate only active canonical records; no FX conversion.                            |
| Represent    | Iroha API and Svelte UI                      | API response/view model                    | Show status and source; never imply OCR is certain.                                   |
| Report       | Iroha report API and Suzuran briefing        | Weekly/monthly response                    | Bound periods by the configured report timezone and return currency-separated totals. |

### State machines

An intake moves through:

```text
received -> canonicalizing -> awaiting_agent -> awaiting_confirmation
                                      |                    |
                                      +-> failed           +-> confirmed
                                                           +-> rejected
```

Typed text that the deterministic parser can fully understand may go directly from `canonicalizing` to `confirmed` because the user explicitly supplied the expense command. An image or ambiguous text
must stop at `awaiting_agent` or `awaiting_confirmation`.

A candidate is `proposed`, `accepted`, `rejected`, or `superseded`. A canonical expense is active until it is deleted by an auditable tombstone; updates create revisions rather than silently replacing
history.

## Storage design

Add a new explicit Goose migration, proposed as `apps/iroha-server/db/migrations/00007_expense_ledger.sql`. The exact sequence number must be checked against the migration directory when
implementation starts.

### Reuse `tb_intake_payloads`

For each intake, create one existing `tb_intake_payloads` row with:

- `source_kind`: `telegram`, `cli`, or another registered client kind.
- `source_actor`: existing actor classification, normally `connector` or `user`.
- `source_event_id`: required for external retries; for Telegram use `telegram:<chat_id>:<message_id>`.
- `content_type`, `sha256`, `size_bytes`, and `storage_path`/`payload_json` as appropriate.

Receipt content remains private. The content-download endpoint must bypass the read cache, must not log the body, and must use the existing raw-storage policy.

### New tables

`tb_expense_intakes`:

- UUIDv7 `id`.
- Unique `intake_payload_id`.
- `status`, `error_message`, `created_at`, `updated_at`, `completed_at`.
- Index status and creation time for worker/review queues.

`tb_expense_candidates`:

- UUIDv7 `id` and `intake_id`.
- `status`, `extractor_kind`, `extractor_version`.
- `candidate_json` containing normalized proposed fields.
- `confidence_json` and `evidence_json` containing field-level confidence and short source locations, never a duplicated receipt.
- `created_at`, `updated_at`.

`tb_expenses`:

- UUIDv7 `id`, `intake_id`, and optional accepted `candidate_id`.
- `amount_minor bigint`, `currency text`, `occurred_on date`.
- `category text`, `merchant text`, and `description text`.
- `source_kind` and `source_event_id` copied for efficient display/audit.
- `created_at`, `updated_at`, and nullable `deleted_at`.
- Index `(occurred_on, currency)` and `(category, occurred_on)`.

`tb_expense_revisions`:

- UUIDv7 `id`, `expense_id`, monotonically increasing `revision_no`.
- `operation` (`create`, `update`, `delete`, `restore` if later approved).
- Complete post-operation `snapshot_json`.
- `actor_kind`, `actor_id`, optional request/correlation ID, and `created_at`.
- Append-only from the application; no destructive correction path.

`tb_expense_mutations`:

- UUIDv7 `id`, `expense_id`, `operation`, `idempotency_key`, `request_hash`, `response_json`, `created_at`.
- Unique `(expense_id, operation, idempotency_key)`.
- A reused key with a different request hash returns `409`, not a second mutation.

Money is represented canonically as integer minor units. The parser accepts human decimal input and applies the currency exponent, but the API does not accept floating-point amounts. v0.4 stores
`occurred_on` as a date; a report timezone is used only when an input omits a date and the client supplies a timestamp.

## HTTP API contract

The active contract is `docs/contracts/openapi.yaml`; route inventory and API tests must be updated in the same implementation slice.

### Collect

`POST /api/v1/expense-intakes`

Text JSON example:

```json
{
  "source_kind": "telegram",
  "source_event_id": "telegram:12345:67890",
  "content_type": "text/plain",
  "text": "800 JPY food ramen"
}
```

Receipt clients use multipart form data with the same source fields and one image file. The response is intentionally small:

```json
{
  "id": "expint_01j...",
  "status": "canonicalizing",
  "duplicate": false
}
```

The `(source_kind, source_event_id)` pair is the intake idempotency key. A retry returns the original intake and `duplicate: true`; the same key with different content returns `409`.

`GET /api/v1/expense-intakes/{intakeId}` returns current status, candidate IDs, and a safe summary. `GET /api/v1/expense-intakes/{intakeId}/content` streams the original receipt for an authorized
private client and is never cached.

### Candidate proposal and review

`POST /api/v1/expense-intakes/{intakeId}/candidates` accepts only structured proposals:

```json
{
  "extractor_kind": "agent",
  "extractor_version": "codex-2026-08-12",
  "candidate": {
    "amount_minor": 1200,
    "currency": "JPY",
    "occurred_on": "2026-08-12",
    "category": "food",
    "merchant": "7-Eleven",
    "description": ""
  },
  "confidence": {
    "amount_minor": 0.99,
    "currency": 0.99,
    "occurred_on": 0.94
  },
  "evidence": [{ "field": "amount_minor", "text": "¥1,200", "location": "receipt total" }]
}
```

`POST /api/v1/expense-candidates/{candidateId}/confirm` and `/reject` require `Idempotency-Key`. Confirmation runs in one transaction: lock/check the candidate, create the expense, append its create
revision, mark candidate/intake state, record the mutation, and invalidate affected report/briefing caches.

### Canonical ledger and reports

- `GET /api/v1/expenses` supports date range, category, currency, and active/deleted filtering.
- `GET /api/v1/expenses/{expenseId}` returns the current snapshot and revision history summary.
- `PATCH /api/v1/expenses/{expenseId}` requires an idempotency key and appends an update revision.
- `DELETE /api/v1/expenses/{expenseId}` requires an idempotency key and creates a tombstone revision.
- `GET /api/v1/expenses/aggregates?from=2026-08-01&to=2026-09-01&timezone=Asia/Tokyo` returns currency-separated total and category buckets.

Aggregate responses must include the resolved `[from, to)` boundary and timezone. Empty periods return zero buckets or an explicit empty result consistently; this is a contract test, not a UI
assumption.

## Canonicalization and jobs

Add `expense_canonicalize` to the existing runtime job kinds. The API enqueues a job with only an `intake_id`; the worker loads the private intake and writes a candidate or a terminal error.

The deterministic parser should handle the first typed format, including amount, currency, optional category, optional note, and a date when supplied. It must reject ambiguous amounts and unsupported
currencies rather than guessing.

For images, the v0.4 worker records `awaiting_agent` and does not invoke an LLM, OCR service, or remote provider. The agent CLI is the first canonicalization client. This keeps secrets, provider
choice, model versions, and receipt bytes outside the core ledger service.

No agent proposal can create a canonical expense without confirmation. The explicit Telegram command may auto-confirm its deterministic candidate because the user supplied the fields directly; this
policy must be visible in tests and API documentation.

## Agent CLI workflow

Create `scripts/expense_cli.py` using the existing Python 3.14/uv script convention and standard-library HTTP client. Default output is JSON for agent composition; `--format table` is a human
convenience. Configuration is `IROHA_API_BASE` plus an optional future `IROHA_API_TOKEN`; v0.4 private-network mode does not require the token.

The CLI is pull-based, not a daemon. A human or coding agent runs it after receiving an intake ID:

```bash
uv run python scripts/expense_cli.py intake create \
  --file ./receipt.jpg \
  --source-kind cli \
  --source-event-id codex-session-20260812-receipt-01

uv run python scripts/expense_cli.py intake content expint_01j... --output /tmp/receipt.jpg

# The agent OCRs /tmp/receipt.jpg without logging the image or receipt text,
# then writes the structured proposal to candidate.json.
uv run python scripts/expense_cli.py candidate propose expint_01j... \
  --input candidate.json \
  --extractor-kind agent \
  --extractor-version codex-2026-08-12

uv run python scripts/expense_cli.py candidate show expcan_01j...
uv run python scripts/expense_cli.py candidate confirm expcan_01j... \
  --idempotency-key confirm:codex-session-20260812-receipt-01
uv run python scripts/expense_cli.py report month 2026-08
```

The CLI must require a caller-supplied stable source event ID for intake creation. It must not silently use a content hash as the deduplication key: two identical receipts can be legitimate separate
expenses. It should print the generated intake/candidate/expense IDs and preserve nonzero exit status for `409`, failed, or awaiting-review states.

The CLI is an agent client, not an autonomous approver. Provider credentials and raw receipt data stay in the caller's environment; the CLI must not put them in logs, shell error messages, telemetry,
or the repository.

## Telegram workflow and Suzuran cutover

For `/expense`, Suzuran creates the Telegram event key, calls the intake endpoint, waits for the candidate/confirmation result, and formats the compact response. For an image, it uploads the image,
tells the user that review is pending, and later presents Confirm/Edit/Reject actions after an agent has proposed a candidate. Undo calls iroha's delete endpoint; it must no longer delete a local row.

Subscription auto-renewal events use a deterministic key such as `subscription:<subscription_id>:<renewal_date>`, so scheduler retries are safe.

The implementation must update all direct local-ledger paths in Suzuran, including:

- `src/suzuran/expenses.py` command/list/summary handlers.
- `src/suzuran/callbacks.py` undo callback.
- `src/suzuran/scheduler.py` subscription auto-renewal insertion.
- `src/suzuran/briefing.py` weekly review and dashboard queries.
- `src/suzuran/iroha.py` client methods and `src/suzuran/iroha_dialog.py`-style polling flow.

Before enabling the new write path, run a one-time backfill from Suzuran's local `tb_expenses`. Preserve the old numeric ID in the source event key, for example `legacy:<old_numeric_id>`, and record
the original source as `suzuran_legacy`. Verify counts and currency/date/amount totals, then stop local writes. Do not dual-write indefinitely; it creates two ledgers with no reliable conflict
resolution.

## Representation and reporting

Add an expense summary to the private Overview only after the API response is stable. Each theme must receive the same typed contributor data. The UI should display:

- current-period total grouped by currency;
- top categories;
- an explicit “expenses” label and link to the ledger/report view;
- pending receipt count separately from confirmed totals.

Weekly/monthly report views should use the aggregate API rather than reimplementing SQL in Svelte or Suzuran. They must show the period boundary, timezone, currency, and whether deleted records are
excluded.

Keep expenses out of public export by default and add a regression test that a populated private ledger produces no expense records in the public projection.

## Implementation slices

1. **Contract and storage:** migration, runtime models/ID prefixes, validation types, revision/idempotency tables, OpenAPI examples, and focused service tests.
2. **Typed vertical slice:** deterministic text parser, intake/confirm/list/update/delete API, cache invalidation, and aggregate queries. Prove retry and correction behavior before touching receipts.
3. **Agent receipt slice:** private content download, candidate proposal/review endpoints, CLI commands, awaiting-review state, and security tests for raw evidence handling.
4. **Suzuran adapter and backfill:** replace local reads/writes, migrate scheduler and undo, backfill legacy rows, then run Telegram smoke tests.
5. **Representation/reporting:** Overview tile, all theme wiring, report views, Suzuran briefing integration, and public-export exclusion test.
6. **Release hardening:** integration tests, migration rollback rehearsal, deployment configuration, monitoring, documentation, and a v0.4 release note.

Each slice should leave the repository buildable and should be committed separately during implementation. This plan itself is only the contract slice.

## Verification matrix

Iroha checks:

- Service/parser tests for amount parsing, currency validation, date defaults, ambiguous input, and candidate state transitions.
- Database/API tests for concurrent duplicate intake, same-key/different-content conflict, confirmation idempotency, update/delete revisions, and cache invalidation.
- Aggregate tests for timezone boundaries, multiple currencies, deleted rows, category grouping, and empty periods.
- API contract tests and OpenAPI validation for every route.
- Public-export regression test proving expenses remain private.
- CLI tests for JSON output, multipart upload, nonzero failure states, and no receipt/body logging.
- `make check`, `make scripts-test`, and `make fmt-docs-check`; use `make test-integration` when the database is available.

Suzuran checks:

- Client tests for intake retry, candidate polling, confirm/edit/reject, undo, and scheduler idempotency.
- Backfill dry-run totals and post-cutover assertion that no handler writes local `tb_expenses`.
- `make check` in Suzuran plus a Telegram integration smoke test in the private deployment.

Cross-repo acceptance:

1. Send one typed Telegram expense and retry the same update.
2. Upload one receipt, propose a candidate through the CLI, and confirm it from Telegram or the CLI.
3. Correct and undo it; verify revision history and report totals.
4. Verify an empty month, a multi-currency month, and a timezone boundary.
5. Verify public export contains no expense data.

## Approval gates

Before implementation begins, confirm these decisions:

1. **Trust boundary:** v0.4 stays private-network-only; full API authentication is a prerequisite for any external exposure.
2. **Money model:** integer minor units, no FX, and currency-separated totals.
3. **Date model:** date-only `occurred_on` for v0.4, with an explicit report timezone.
4. **Receipt scope:** receipt total only; no line items.
5. **Agent policy:** external/pull-based OCR or vision; agent proposals require human confirmation.
6. **Ledger ownership:** Suzuran's local table is backfilled and retired; iroha becomes the canonical ledger.
7. **Release shape:** typed expenses are the first shippable vertical slice; receipt-agent flow follows as a separately verifiable slice.

If the desired v0.4 is smaller, defer the receipt-agent slice and ship only typed intake, canonical records, corrections, aggregates, and reports. The storage boundary should still retain enough
provenance to add receipt candidates later without redesigning `tb_expenses`.
