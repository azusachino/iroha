# Setting up the Apple Health Shortcut

How to build an iOS Shortcut that sends bounded Health data to `POST /api/v1/intake/health`.

The server side of this path is implemented and fixture-verified. **No producer has been built yet** — this document is the first attempt at one, so treat it as a recipe to follow and correct, not a
description of something already running. See [provider capabilities](capabilities/providers/apple-health.md) for the acceptance boundary and [ADR-0007](adr/0007-v05-health-acceptance-boundary.md) for
what v0.5 does and does not claim.

## What this path is for

Full `export.zip` imports remain the authority for history: they are a complete snapshot and reconcile accordingly. The Shortcut is for the gap between exports — a small, bounded window of recent data
delivered automatically, so you are not exporting 900MB by hand to see last week's steps.

Because it is bounded, it never deletes anything. A window you send with no records in it asserts "nothing happened here", not "delete what you had".

## Before you start

1. **The endpoint must exist and be configured.** It ships in v0.5; the cluster currently runs v0.4.5. Set `IROHA_HEALTH_INTAKE_TOKEN` in the deployment secret environment — without it the endpoint
   returns `503 health_intake_disabled`. Keep the token out of tracked TOML and out of request logs.
2. **Your iPhone must reach the host.** `iroha.h.azusachino.com` resolves through Tailscale split-DNS, so the phone needs the Tailscale app connected and the tailnet's DNS settings applied. If the
   Shortcut can reach the URL in Safari, it can reach it from `Get Contents of URL`.
3. **Generate a token you are willing to put in a Shortcut.** It is a bearer credential stored in plain text on the device. Scope it to this one endpoint and rotate it if the phone is lost.

Prove the endpoint works with `curl` before building anything in Shortcuts — debugging a 200-step visual program against an endpoint you have not tested is the slow way round:

```bash
curl -sS -X POST https://iroha.h.azusachino.com/api/v1/intake/health \
  -H "Authorization: Bearer $IROHA_HEALTH_INTAKE_TOKEN" \
  -H 'Content-Type: application/json' \
  --data @apps/iroha-providers/parsers/testdata/apple_health_shortcut.json
```

A healthy response is `202 Accepted` with `raw_file_id`, `import_id` and `status`. Send it twice: the second call is deduplicated by raw-file hash and must not double-count anything.

## What Shortcuts can and cannot produce

This decides the shape of your payload, so read it before building.

| Envelope section  | Feasible in Shortcuts? | Notes                                                                                                            |
| ----------------- | ---------------------- | ---------------------------------------------------------------------------------------------------------------- |
| `coverage`        | Yes                    | You construct it yourself from the date range you queried. Required, and must be non-empty.                      |
| `daily_metrics`   | Yes                    | `Find Health Samples` over a quantity type, summed per day. The easiest place to start.                          |
| `activities`      | Probably               | Workouts are queryable, but you must synthesise a stable `id` yourself.                                          |
| `sleep`           | Awkward                | Stages are exposed on recent iOS, but sessionising them across midnight in Shortcuts is painful. Consider later. |
| `daily_summaries` | **No**                 | Needs Activity ring _goals_, which come from `HKActivitySummary` — not a `Find Health Samples` type.             |

Omit any section you cannot fill. Every array except `coverage` is optional; omitted is fine, and **an unknown field is fatal** — the decoder runs with `DisallowUnknownFields` and rejects the whole
payload, so do not add a `"notes"` or `"device"` key to be helpful.

## Build the minimal Shortcut: daily steps

Start here. Get one metric landing end to end, then extend.

1. **Text** — name it so you can find it later. Put your token in it, or better, read it from a password manager action. Call the variable `Token`.
2. **Find Health Samples**
   - Type: `Steps`
   - Filter: `Start Date` `is in the last` `7` `days`
   - Sort by `Start Date`, order ascending
   - **Check the input says `All Health Samples`.** If you added this action after another Health action, Shortcuts silently rewrites it to `Filter [Health Sample]` operating on the previous output,
     and it will only ever return that one sample. This is the single most common way this goes wrong.
3. **Calculate Statistics** → `Sum` over the samples, grouped per day. If grouping is awkward, run one `Find Health Samples` + `Sum` per day inside a `Repeat` over the last 7 days — slower but far
   easier to reason about.
4. **Dictionary** per day, with exactly these keys: `day` (`YYYY-MM-DD`), `metric` (`steps`), `value` (number), `unit` (`count`). Append each to a list variable `Metrics`.
5. **Dictionary** for the envelope:
   - `schema` → `iroha.health.shortcut.v1`
   - `source_instance_key` → `iphone-health:primary` (any stable string; it is how iroha groups this device's evidence over time, so never change it casually)
   - `captured_at` → current date, ISO 8601
   - `coverage` → a one-element list, see below
   - `daily_metrics` → `Metrics`
6. **Get Contents of URL**
   - URL: `https://iroha.h.azusachino.com/api/v1/intake/health`
   - Method: `POST`
   - Headers: `Authorization` → `Bearer ` + `Token`
   - Request Body: `JSON`, and pass the envelope dictionary
7. **Show Result** while developing, so you can see the `202` or the error body.

The coverage element for the above:

```json
{
  "category": "daily_metrics",
  "scope": {},
  "from": "2026-09-05T00:00:00+09:00",
  "to": "2026-09-12T00:00:00+09:00",
  "timezone": "Asia/Tokyo",
  "completeness": "unknown"
}
```

`from`/`to` are half-open and `from` must be strictly before `to`. `timezone` must be an IANA name. `scope` must be a JSON object — `{}` is fine, and omitting it entirely is also fine.

### Which `completeness` to send

This is the field that carries the honesty of the whole payload, so pick deliberately:

- `unknown` — you cannot prove you read the whole window. **This is the right default for a Shortcut**: if the phone was locked for part of the run, or a permission was silently denied, you got
  partial data and no error.
- `partial` — you know the window is incomplete.
- `covered` — you are asserting the window is complete. Only send this if you genuinely know it.
- `covered_empty` — the window is complete and genuinely contains nothing.

Completeness is recorded as evidence, not acted on: `covered` does not currently cause iroha to delete records inside the window. Sending it wrongly does not destroy data today, but it does write a
false statement into the evidence chain.

## Automating it

Shortcuts → Automation → Personal Automation → Time of Day, then Run Immediately with "Notify When Run" off once it works.

**Do not expect this to be reliable, and do not design around it being reliable.** Apps cannot read Health data while the iPhone is locked — this is a HealthKit restriction, not a Shortcuts quirk, and
it affects every tool in this space including the commercial ones. A scheduled 03:00 run against a locked phone returns nothing, and nothing is indistinguishable from a genuinely empty day. That is
precisely why `completeness: unknown` and a 7-day overlapping window matter: the next successful run re-covers the days the failed one missed.

Practical mitigations, in order of how well they work:

1. Overlap generously. A 7-day window run daily tolerates six consecutive failures.
2. Trigger on something that implies an unlocked phone — an Automation on "When I open Health", or a Back Tap — rather than a fixed clock time.
3. Accept that some days need a manual tap.

## If it does not work

| Response                       | Meaning                                                                                                                                     |
| ------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------- |
| `503 health_intake_disabled`   | `IROHA_HEALTH_INTAKE_TOKEN` is unset on the server.                                                                                         |
| `401 unauthorized`             | Missing or wrong `Authorization: Bearer <token>` header.                                                                                    |
| `400 invalid_health_payload`   | The envelope failed validation. Almost always an unknown field, a missing/empty `coverage`, a bad timezone name, or `from` not before `to`. |
| `400 invalid_body`             | Body over 2 MiB. Narrow the window or send fewer categories.                                                                                |
| `202` but nothing appears      | The import job runs in the background; check the raw file and import status via the API using the returned ids.                             |
| Request never leaves the phone | Tailscale is not connected, or split-DNS is not applied.                                                                                    |

A `202` means the evidence was stored and the work was queued — not that the import succeeded. The raw payload is retained either way, so a parsing problem can always be replayed later rather than
re-collected from the device.

## If Shortcuts turns out to be too fragile

The realistic alternative is [Health Auto Export](https://help.healthyapps.dev/en/health-auto-export/automations/), which already POSTs JSON to a custom REST endpoint and covers more types than
Shortcuts exposes. It emits its own payload shape, so it would need its own adapter and envelope version (`iroha.health.hae.v1`) rather than this one — the server's coverage and reconciliation
contract would not change. It is subject to the same locked-phone restriction, and has no missed-day backfill of its own.
