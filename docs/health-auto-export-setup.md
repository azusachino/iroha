# Setting up Health Auto Export HTTP intake

How to make an iPhone send daily health data and workouts to Iroha automatically, using [Health Auto Export](https://help.healthyapps.dev/en/health-auto-export/automations/) (HAE).

When you finish, the phone uploads the last 7 days of daily metrics and recent workouts to `POST /api/v1/intake/health` in the background. Every setting below is chosen for a reason explained in
[the HAE reference](capabilities/providers/health-auto-export.md); change one only after reading why it is set that way.

Historical data does not come through this path. Backfill years of history once with a full Apple Health `export.zip` through `POST /api/v1/raw-files` and `POST /api/v1/imports`; see
[Apple Health](capabilities/providers/apple-health.md).

## Before you start

You need:

- Iroha running with database migrations applied through `00021_health_intake_credential`.
- An HTTPS URL for Iroha that the iPhone can reach. Below it is written `https://<iroha-host>`. A public intake hostname means the phone needs no VPN; the deployment must route only
  `POST /api/v1/intake/health` there (see [public exposure](iroha-server.md#auth)).
- A shell where you can run the `iroha-server` binary against Iroha's database (inside the server container, or on the host running it).
- Health Auto Export installed on the iPhone, with a subscription that includes REST API automations.

## Step 1: Issue the intake token

Iroha refuses all intake (`503 intake_not_provisioned`) until a token exists. Log in to Iroha, open **Admin**, and under **Intake tokens** keep the device name `primary` and choose **Issue token**.
Without the web app, an operator can run `iroha-admin intake-token issue` in the server container instead.

Iroha shows the token **once**. Iroha stores only its SHA-256 hash, so the token cannot be shown again. Copy it straight into HAE in step 3; do not save it in a note, a tracked file, or a chat.

The token is named `primary` and uploads are recorded under source instance `iphone-hae:primary`. A second device needs its own token under its own name, such as `ipad`.

## Step 2: Prepare the iPhone

1. **Settings → General → Background App Refresh**: on, and on for Health Auto Export.
2. In Health Auto Export, grant Health read access for every metric you select in step 3 and for Workouts.
3. Expect uploads only after the phone has been unlocked. iOS blocks Health data while locked, and Low Power Mode delays background work. Charging overnight while unlocked recently is the most
   reliable time.

## Step 3: Create the Health Metrics automation

In Health Auto Export, open **Automations**, add a new automation, and choose **REST API**. Set every field exactly as listed:

| Field           | Value                                                           |
| --------------- | --------------------------------------------------------------- |
| Automation Name | `iroha metrics`                                                 |
| URL             | `https://<iroha-host>/api/v1/intake/health`                     |
| Headers         | `Authorization` = `Bearer <token from step 1>`                  |
| Data Type       | Health Metrics                                                  |
| Health Metrics  | `Step Count`, `Sleep Analysis`, plus any optional metrics below |
| Export Format   | JSON                                                            |
| Export Version  | Version 2                                                       |
| Summarize Data  | ON                                                              |
| Time Grouping   | Day                                                             |
| Date Range      | **Previous 7 Days**                                             |
| Batch Requests  | OFF                                                             |
| Sync Cadence    | every 1 hour (a request to iOS, not a guarantee)                |

Optional metrics Iroha understands: Walking + Running Distance, Flights Climbed, Resting Heart Rate, Walking Heart Rate Average, Heart Rate Variability, VO2 Max, Body Mass (weight), Blood Oxygen
Saturation, Respiratory Rate, Active Energy, Exercise Time, and Stand Hours. Select Active Energy, Exercise Time, and Stand Hours together if you want the daily Move/Exercise/Stand summary. Other metrics stay out of normalized observations and are listed in the upload's `unsupported_metrics` coverage scope; check that scope before assuming the Health import is complete.

> [!WARNING] **Never use "Since Last Sync" for Health Metrics.** Iroha replaces each day's stored value with the one in the latest upload. "Since Last Sync" sends only the samples since the previous
> run, so the day's total would be overwritten by a fragment. "Previous 7 Days" also repairs any days the phone missed.

> [!WARNING] **Keep Summarize Data ON.** With Summarize OFF, HAE sends sleep as stage intervals without a nightly entry, and Iroha records **no sleep at all**.

## Step 4: Create the Workouts automation

Add a second REST API automation:

| Field                           | Value                                          |
| ------------------------------- | ---------------------------------------------- |
| Automation Name                 | `iroha workouts`                               |
| URL                             | `https://<iroha-host>/api/v1/intake/health`    |
| Headers                         | `Authorization` = `Bearer <token from step 1>` |
| Data Type                       | Workouts                                       |
| Export Format                   | JSON                                           |
| Export Version                  | Version 2                                      |
| Include Route Data              | ON                                             |
| Include Workout Metrics         | ON                                             |
| Time Grouping (Workout Metrics) | Minutes                                        |
| Date Range                      | Default                                        |
| Batch Requests                  | OFF (see troubleshooting if uploads time out)  |
| Sync Cadence                    | every 1 hour                                   |

The token identifies the device, so no device header is needed. Issuing the same device a token under a different name later splits its provenance history.

## Step 5: Test each automation

1. In each automation, run a manual export (pick a date range that contains data) and wait for it to finish.
2. Open the automation's **Activity Logs**. A successful run shows HTTP **202**.
3. In Iroha, the daily steps, last night's sleep, and any recent workout appear once the background import job has run.

You can also probe the server directly. This checks the URL and token, not the phone:

```bash
curl -sS -X POST https://<iroha-host>/api/v1/intake/health \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  --data @apps/iroha-providers/parsers/testdata/health_auto_export.json
```

A healthy response is `202 Accepted`:

```json
{ "raw_file_id": "raw_…", "import_id": "imp_…", "status": "queued" }
```

The probe stores the test fixture as real evidence. Use it against a development instance, not production.

## Verify with a real device

Before relying on the intake, confirm the facts HAE's documentation leaves open (tracked in [the HAE reference](capabilities/providers/health-auto-export.md#open-questions)):

1. Capture one real upload from each automation (for example with a request bin on your own network, or from Iroha's stored raw file).
2. Send the same workout upload twice. Iroha must still show **one** activity. If it shows two, the workout `id` is not stable.
3. Note the time and UTC offset in a summarized metric's `date`, and the size of a workout upload with a route.
4. Check every optional metric you selected actually appears in Iroha. A missing one means HAE spells its name differently from Iroha's parser; the upload's coverage scope lists it under `unsupported_metrics`.
5. Replace `apps/iroha-providers/parsers/testdata/health_auto_export.json` with a trimmed, anonymized real payload (move GPS points away from home, remove device names).

## Rotate the token

1. On **Admin → Intake tokens**, issue a new token with the same device name and copy it. The old token keeps working.
2. Paste it into the `Authorization` header of **both** automations.
3. Run each automation manually and confirm **202** in Activity Logs.
4. Revoke the old token in the same list.

If a token leaks, revoke it first; the phone then fails with `401` until step 2, and "Previous 7 Days" re-sends the missed data.

## Troubleshooting

| Activity Logs shows             | Cause                                                                                       | Fix                                                                                                                         |
| ------------------------------- | ------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------- |
| `503 intake_not_provisioned`    | No token has been issued on this Iroha instance.                                            | Step 1.                                                                                                                     |
| `401 unauthorized`              | Header missing, not `Bearer <token>`, or the token was rotated.                             | Re-check the header in both automations; rotate if the token is lost.                                                       |
| `400 invalid_health_payload`    | Not Version 2 JSON (workout without `id`, timestamp without UTC offset) or CSV selected.    | Export Format JSON, Export Version 2.                                                                                       |
| `400 invalid_body`              | Upload over 10 MiB.                                                                         | Workouts: turn Batch Requests ON. Metrics: remove metrics you do not need.                                                  |
| Timeout / no request            | iOS stopped the background task (about 30 s), phone locked, or Low Power Mode.              | Unlock and charge the phone; for workouts, turn Batch Requests ON or keep grouping at Minutes.                              |
| `202` but the import job failed | Summarize Data OFF ("enable Summarize Data"), or a malformed field; the job error names it. | Summarize Data ON; otherwise capture the upload and report the error.                                                       |
| `202` but a metric missing      | HAE's metric name is not one Iroha recognizes.                                              | Check the name in the raw upload against [the metric mapping](capabilities/providers/health-auto-export.md#metric-mapping). |
| Day totals too low              | Date Range is "Since Last Sync", or Batch Requests is ON for metrics.                       | Date Range "Previous 7 Days", Batch Requests OFF.                                                                           |

HAE's own diagnostics: **Activity Logs** in each automation, and its [App Event Logs](https://help.healthyapps.dev/en/health-auto-export/troubleshooting/app-event-logs).
