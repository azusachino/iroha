# Setting up Health Auto Export HTTP Intake

How to configure [Health Auto Export](https://help.healthyapps.dev/en/health-auto-export/automations/) (HAE) on iOS/watchOS to deliver continuous, bounded health metrics, sleep sessions, and workout routes to `POST /api/v1/intake/health`.

This setup replaces the experimental, unbuilt iOS Shortcuts path and manual full-export ZIP friction with automated, native background ingestion.

---

## User Stories

- **As a daily health tracker**, I want my iPhone and Apple Watch to sync daily step counts, sleep stages, and resting vitals to Iroha automatically in the background, so my dashboard reflects current data without manual exports.
- **As a runner or cyclist**, I want completed workouts—including GPS route trackpoints and continuous heart rate samplings—to land in Iroha immediately with full route map and lap visualization.
- **As an operator of a personal data cockpit**, I want daily intake to run securely over my private Tailnet perimeter without storing plain-text bearer tokens inside third-party mobile apps, while still retaining the ability to enforce token authentication when desired.
- **As a user with years of historical data**, I want a clear two-tier ingestion model: one-time historical backfill using full `export.zip` archives, and continuous rolling daily intake using lightweight JSON syncs.

---

## Two-Tier Ingestion Architecture

| Tier | Channel | Data Source & Format | Frequency | Ingestion Mode |
| :--- | :--- | :--- | :--- | :--- |
| **Tier 1: Daily Intake** | `POST /api/v1/intake/health` | Health Auto Export Format v2 JSON | Continuous background sync (rolling 2 days) | `bounded_replacement` (updates evidence for specified window without purging historical records) |
| **Tier 2: Historical Bulk Backfill** | `POST /api/v1/raw-files` + `POST /api/v1/imports` | Apple Health `export.zip` (`export.xml` + GPX routes), FIT, TCX, GPX | On-demand / one-time | `full_snapshot` / archive reconciliation |

---

## Security Model: Tailnet Perimeter & Configurable Auth

Iroha is designed as a single-user personal data cockpit hosted on a private Tailscale network (`iroha.h.azusachino.com`).

### 1. Default: Network Perimeter Security (No Auth Header Required)
Because `iroha.h.azusachino.com` is resolvable and accessible only by devices authenticated to Haru's tailnet:
- WireGuard encryption and tailnet node identity protect all communications in transit.
- **No API token or `Authorization` header is required** in Health Auto Export by default when `IROHA_HEALTH_INTAKE_TOKEN` is unset.
- This eliminates the security risk of storing long-lived plain-text bearer tokens in mobile app configurations.

### 2. Configurable Token Authentication (Optional Defense-in-Depth)
If you wish to enforce token authentication even on the tailnet:
- Set `IROHA_HEALTH_INTAKE_TOKEN=<secret>` in `iroha-server`'s environment or deployment secret.
- When set, `POST /api/v1/intake/health` strictly enforces `Authorization: Bearer <secret>` (returning `401 Unauthorized` if omitted or invalid).
- In Health Auto Export, add an `Authorization` header with `Bearer <secret>`.

---

## Client Configuration (iOS / watchOS)

Open the **Health Auto Export** app on iOS. You only need to configure two data types: **Health Metrics** and **Workouts**. Do **not** enable Symptoms, ECG, or Medications unless specifically needed.

### 1. Health Metrics Export Settings
- **Export Format**: `JSON`
- **Export Version**: `Version 2` (Format v2)
- **Summarize Data**: `ON`
- **Group By**: `Day`
- **Date Range**: `Rolling 2 Days` (or `3 Days` to ensure complete coverage across timezones)
- **Selected Metrics**:
  - `step_count` (Steps)
  - `sleep_analysis` (In bed, asleep, core, deep, REM, awake durations and stages)
  - *(Optional)* `resting_heart_rate`, `heart_rate_variability_sdnn`, `walking_running_distance`

> [!NOTE]
> Iroha's HAE parser is sparse: unconfigured metrics are omitted safely. The parser gracefully handles missing metrics and automatically converts fractional hours (e.g. sleep duration) into integer seconds.

### 2. Workouts Export Settings
- **Export Format**: `JSON`
- **Export Version**: `Version 2`
- **Include Route Data**: `ON` (exports GPS trackpoints with latitude, longitude, and elevation)
- **Include Workout Metrics**: `ON` (exports heart rate series)
- **Time Grouping (Workout Metrics)**: `Minutes` (1-minute intervals)
- **Date Range**: `Rolling 2 Days`

### 3. Automation / Sync Settings
Under **Automations** (or **REST API Export**):
- **Export Type**: `REST API`
- **URL**: `https://iroha.h.azusachino.com/api/v1/intake/health`
- **HTTP Method**: `POST`
- **HTTP Headers**:
  - `Content-Type: application/json`
  - *(Optional)* `X-Device-Id: iphone-hae:primary`
  - *(Optional, only if `IROHA_HEALTH_INTAKE_TOKEN` is set on server)* `Authorization: Bearer <your-token>`
- **Background Sync**: `Enabled` (allows iOS background refresh to upload new metrics automatically)

---

## Verification & Probing

You can verify that the endpoint is reachable and correctly processes Format v2 payloads using `curl`:

```bash
curl -sS -X POST https://iroha.h.azusachino.com/api/v1/intake/health \
  -H "Content-Type: application/json" \
  --data @apps/iroha-providers/parsers/testdata/health_auto_export.json
```

A healthy response returns `202 Accepted`:

```json
{
  "raw_file_id": "raw_01jm8v7w...",
  "import_id": "imp_01jm8v7x...",
  "status": "queued"
}
```

The server stores the raw JSON in `tb_raw_files` with `ingestion_mode = bounded_replacement` and queues an import job in `tb_import_jobs` under parser kind `health_auto_export`. Replaying the same payload is idempotent.

---

## Retirement of iOS Shortcuts

The previous experimental iOS Shortcut receiver (`apple-health-shortcut.go` and `apple-health-shortcut-setup.md`) has been retired and removed. Because no on-device Shortcut producer could reliably build the envelope (as recorded in ADR-0007), removing this path eliminates dead code and leaves Health Auto Export as the sole, authoritative daily intake path.
