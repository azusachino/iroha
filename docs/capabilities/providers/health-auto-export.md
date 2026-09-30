# Health Auto Export reference

Status: the daily intake was verified on the iPhone on 2026-09-28. The parser keeps its supported metric scope explicit; each HAE coverage assertion now records the names and point counts actually sent.

What [Health Auto Export](https://help.healthyapps.dev/en/health-auto-export/automations/) (HAE) is, what it sends, and how each of its settings affects Iroha's data. This page is the reference; the
step-by-step configuration is [the setup guide](../../health-auto-export-setup.md). Decisions are in [ADR-0008](../../adr/0008-health-auto-export-http-intake.md).

HAE is a third-party iOS app. It reads HealthKit on the iPhone and uploads JSON to a URL on a schedule. It is the producer for Iroha's daily health intake at `POST /api/v1/intake/health`. Full Apple
Health `export.zip` archives remain the historical backfill path; see [Apple Health](apple-health.md).

## Automation settings and their effect on Iroha

A REST API automation in HAE exports exactly one data type (Health Metrics, Workouts, Symptoms, ...). Iroha needs two automations: one for Health Metrics and one for Workouts.

| HAE setting              | Options                                                     | Effect on Iroha                                                                                                                                                                               |
| ------------------------ | ----------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Export Format            | JSON, CSV                                                   | Only JSON is parsed. CSV is sent as `multipart/form-data` and rejected with `400`.                                                                                                            |
| Export Version           | Version 1, Version 2                                        | **Only Version 2 is supported.** Intake rejects a workout without an `id` or a start time not in `yyyy-MM-dd HH:mm:ss Z` form with `400`; any other malformed timestamp fails the import job. |
| Date Range               | Default, Since Last Sync, Today, Yesterday, Previous 7 Days | Decides which days each upload restates. See [date range](#date-range).                                                                                                                       |
| Summarize Data           | ON, OFF (JSON + Health Metrics only)                        | ON sends one value per metric per day and one sleep entry per night. See [sleep](#sleep).                                                                                                     |
| Time Grouping (metrics)  | available when Summarize is ON                              | Iroha sums or averages per day regardless; day grouping keeps the payload smallest.                                                                                                           |
| Batch Requests           | ON, OFF (JSON only)                                         | ON splits one export across several uploads. See [batching](#batching).                                                                                                                       |
| Include Route Data       | ON, OFF (Workouts)                                          | ON attaches GPS points to the activity. Large; see [limits](#limits).                                                                                                                         |
| Include Workout Metrics  | ON, OFF (Workouts)                                          | ON attaches the per-workout heart-rate series as samplings.                                                                                                                                   |
| Time Grouping (workouts) | Minutes, Seconds (Version 2)                                | Seconds multiplies payload size by about 60 for no gain in Iroha's charts.                                                                                                                    |
| Headers                  | any key/value                                               | Carries `Authorization: Bearer <token>` (required). The token's credential name sets the source instance.                                                                                     |
| Sync cadence             | number + interval                                           | A request, not a guarantee; iOS decides when background work runs.                                                                                                                            |

HAE also adds its own headers to every request: `automation-name`, `automation-id`, `automation-aggregation`, `automation-period`, and `session-id`. Iroha does not read them today; they are useful
when reading server logs.

### Date range

Iroha stores one value per `(day, metric)` and **each upload replaces the stored value** for the days it contains. The date range therefore decides correctness, not just payload size:

- **Previous 7 Days** (use for Health Metrics). Every upload restates a full week of daily totals. If the phone does not sync for several days, the next successful upload repairs all of them.
  Summarized daily data for a week is a few kilobytes.
- **Default** (use for Workouts). The full previous day plus today so far. Workouts are stored by their HAE `id`, so re-sending the same workout updates it instead of duplicating it.
- **Since Last Sync** — **never use it for Health Metrics.** It sends only samples recorded since the previous run, so the evening upload would overwrite the day's step total with the afternoon's
  steps alone. It is safe for Workouts in principle, but HAE does not document whether a failed upload still advances "last sync", so a failure could lose a workout.
- **Today** / **Yesterday** — no recovery from missed syncs; not recommended.

### Sleep

Sleep arrives in one of two shapes depending on **Summarize Data**, never both:

| Summarize | Shape                                                                                                              | Iroha today                                                                                      |
| --------- | ------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------ |
| ON        | One entry per night: `sleepStart`, `sleepEnd`, `totalSleep`, `core`, `deep`, `rem`, `awake`, `inBed`, in **hours** | Parsed into a sleep session with stage durations. No stage-by-stage timeline.                    |
| OFF       | One entry per stage interval: `startDate`, `endDate`, `value` (`Core`, `Deep`, `REM`, `Awake`, ...)                | **Rejected.** The import job fails with "enable Summarize Data"; nothing from that upload lands. |

Observed in real exports by third parties (not in HAE's docs): `asleep` and `inBed` are always `0`, `totalSleep` equals `core + deep + rem`, and `awake` is not part of `totalSleep`. Iroha falls back
to `totalSleep` for time asleep and to `sleepEnd − sleepStart` for time in bed.

Stage-by-stage timelines are available from the full Apple Health export.

### Batching

HAE does not document how Batch Requests splits an export. If it ever split one metric's day across two uploads, the second upload would overwrite the day's total with a partial value. Keep Batch
Requests **OFF** for Health Metrics. For Workouts, turn it on only if an upload fails on size or time (see [limits](#limits)); if it splits by whole workout, each workout still carries its own `id`
and is stored correctly.

### Limits

- iOS allows background work about **30 seconds**. Route data and second-level grouping are the usual reasons an automation times out.
- Iroha accepts at most **10 MiB** per request; larger requests are rejected with `400 invalid_body`.
- HealthKit is unreadable while the iPhone is locked. Automations run only after the phone has been unlocked, and Low Power Mode or disabled Background App Refresh delays or stops them.
- Timing is best-effort. HAE states that automations are not guaranteed to run at the configured time.

## Payload format (Version 2)

```json
{
  "data": {
    "metrics": [{ "name": "step_count", "units": "count", "data": [{ "date": "2026-09-20 00:00:00 +0900", "qty": 8123, "source": "iPhone" }] }],
    "workouts": [
      {
        "id": "…",
        "name": "Outdoor Run",
        "start": "2026-09-20 07:00:00 +0900",
        "end": "2026-09-20 07:45:00 +0900",
        "duration": 2700,
        "distance": { "qty": 7.2, "units": "km" },
        "activeEnergyBurned": { "qty": 520, "units": "kcal" },
        "heartRateData": [{ "date": "2026-09-20 07:01:00 +0900", "Min": 120, "Avg": 135, "Max": 150, "units": "bpm" }],
        "route": [{ "latitude": 35.68, "longitude": 139.76, "altitude": 40, "speed": 3.1, "timestamp": "2026-09-20 07:00:05 +0900" }]
      }
    ]
  }
}
```

- Timestamps are always `yyyy-MM-dd HH:mm:ss Z`, carrying the **device's UTC offset at the time**. Workout `duration` is seconds; sleep values are hours.
- Units follow the phone's preferences (`km` or `mi`, `kcal` or `kJ`). Iroha converts miles and metres to kilometres and kJ to kcal.
- The top-level `data` object may also contain `stateOfMind`, `medications`, `symptoms`, `cycleTracking`, `ecg`, and `heartRateNotifications`; Iroha ignores them.

## Metric mapping

Metric names that Iroha does not recognize are **not stored** (the raw upload is kept while retention allows). The upload's `health` coverage scope records `observed_metrics` as `{ "<hae name>": <data points> }` and the unmapped subset as `unsupported_metrics` with the same shape. Any unsupported metric makes completeness `partial`; otherwise it remains `unknown`, because a bounded HAE upload does not prove complete Health coverage. Neither state claims that every Health category was collected.

The parser deliberately supports metrics with destinations in Iroha's existing daily model. Other names remain visible in `unsupported_metrics` rather than being guessed into a superficially similar field. Since `DailySummary` cannot represent a missing ring field, HAE Move/Exercise/Stand values are emitted only for days with a point for all three metrics; otherwise the observed names remain visible and no partial summary is written for that day.

| HAE metric name               | Iroha metric       | Daily reduction |
| ----------------------------- | ------------------ | --------------- |
| `step_count`                  | steps              | sum             |
| `walking_running_distance`    | distance (km)      | sum             |
| `flights_climbed`             | flights            | sum             |
| `resting_heart_rate`          | resting heart rate | average         |
| `walking_heart_rate_average`  | walking heart rate | average         |
| `heart_rate_variability_sdnn` | HRV (SDNN)         | average         |
| `vo2_max`                     | VO2 max            | average         |
| `body_mass`                   | body mass          | average         |
| `blood_oxygen_saturation`     | SpO2               | average         |
| `oxygen_saturation`           | SpO2 (alias)       | average         |
| `respiratory_rate`            | respiratory rate   | average         |
| `active_energy`               | Move (kcal)        | sum; kJ converted to kcal |
| `apple_exercise_time`         | Exercise (minutes) | sum; seconds converted to minutes |
| `apple_stand_hour`            | Stand (hours)      | sum             |
| `sleep_analysis`              | sleep session      | per night       |

## Days and time zones

- A daily value keeps its **device-local day**: steps summarized on a phone in Shanghai for 20 September belong to 20 September, even though Iroha's effective timezone is Tokyo.
- The upload's coverage window is labelled in the **effective timezone** (`IROHA_TIMEZONE`) and widened so every device-local day in the payload fits inside it whole.
- Coverage records what was observed; it never deletes data outside the payload.

See [the glossary](../../glossary.md) for _device-local day_, _coverage assertion_, and _intake credential_.

## Open questions

The owner verified the deployed HAE workflow on an iPhone on 2026-09-28 and accepted the existing source-precedence and workout-identity behavior; issues [#69](https://github.com/azusachino/iroha/issues/69) and [#70](https://github.com/azusachino/iroha/issues/70) are closed as not planned. The exact 34-name device inventory was not retained in the repository, so each subsequent upload records its observed names and counts in coverage instead of treating an undocumented inventory as complete.

- Does a failed upload advance "Since Last Sync"? The setup guide avoids this uncertainty by using **Previous 7 Days** for Health Metrics.
- How large is a real workout upload with route data? Keep Batch Requests OFF for metrics; enable it for workouts only if needed.

## Sources

- [HAE automations overview](https://help.healthyapps.dev/en/health-auto-export/automations/)
- [HAE REST API automation](https://help.healthyapps.dev/en/health-auto-export/automations/rest-api)
- [HAE JSON export format](https://help.healthyapps.dev/en/health-auto-export/export-format/)
- [HAE health metrics format](https://help.healthyapps.dev/en/health-auto-export/export-format/health-metrics/)
- [HAE supported data and metric names](https://help.healthyapps.dev/en/health-auto-export/getting-started/supported-data/)
- [HAE workouts format](https://help.healthyapps.dev/en/health-auto-export/export-format/workouts/)
- [Third-party observation of sleep fields](https://github.com/gbram895/Training-Dashboard/pull/22)

Retrieved 2026-09-28.
