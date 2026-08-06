# Public-site publishing workflow

Companion to [roadmap Milestone 7](roadmap.md#milestone-7-privacy-and-publishing). That section says the design; this document is the operational detail an operator needs day to day — the full
pipeline, the review loop, and how to undo a bad publish. Tracks [issue #41](https://github.com/azusachino/iroha/issues/41).

## Pipeline

```mermaid
flowchart LR
    subgraph priv["Private network (harus-k3s)"]
        DB[(Postgres/PostGIS)] --> CLI["iroha-export-public CLI"]
        CLI --> GATE{"publicexport.Validate<br/>schema + privacy gate"}
        GATE -- fail --> ABORT["exit non-zero<br/>no files written"]
        GATE -- pass --> FILES["summary.json / activities.json /<br/>routes.geojson / meta.json"]
        FILES --> CRON["export-public-cron.sh<br/>(iroha-export-public CronJob)"]
        CRON -- "git diff empty" --> NOOP["no push"]
        CRON -- "git diff changed" --> PUSH["commit + push to main"]
    end
    PUSH --> GH["GitHub main:<br/>apps/iroha-public-site/static/data/**"]
    GH --> WF["public-site.yml workflow"]
    WF --> BUILD["SvelteKit static build<br/>BASE_PATH=/iroha"]
    BUILD --> DEPLOY["Deploy to GitHub Pages"]
    DEPLOY --> SMOKE{"Smoke check:<br/>base path + data/*.json"}
    SMOKE -- pass --> LIVE["azusachino.github.io/iroha"]
    SMOKE -- fail --> FAILWF["workflow run marked failed<br/>previous deployment stays live"]
```

Every step left of the GitHub push happens inside the private network; nothing there is reachable from the public internet. The only thing that crosses the boundary is the sanitized snapshot, as a git
push.

### Validation gate

`publicexport.Validate` (`apps/iroha-server/pkg/publicexport/validate.go`) runs after the summary/activities/routes are built in memory and before anything is written to `--out`. It is a regression
gate, not a privacy-detection system: it catches a raw (non-`act_`-prefixed) activity ID, a negative metric, an `ended_at` before `started_at`, or an out-of-range coordinate — the shapes a future
change to `Activity`, `Summary`, or route sanitization would produce if it accidentally bypassed the sanitizer. A failure exits non-zero with no partial files on disk, so the cron script's `set -eu`
aborts before any `git add`/`commit`/`push`.

### Freshness

`meta.json` (`{"generated_at": "..."}`) is written alongside the data files. The public site reads it in `+page.ts` and shows "Data as of \<date\>" in the hero, so a visitor never mistakes the
snapshot for a live feed.

### CI smoke check

The last step of `.github/workflows/public-site.yml` curls the deployed `page_url` after `deploy-pages` and asserts:

- the `/iroha` base path root returns the expected title/heading text (confirms the base path and build landed correctly, not a 404 or a root-site default page)
- `data/meta.json`, `data/summary.json`, `data/activities.json`, `data/routes.geojson` are all fetchable under that base path

A failure here fails the workflow run but does **not** take the site down — GitHub Pages keeps serving the last successful `deploy-pages` output until a later run supersedes it.

## Operator/editor review loop

There is deliberately **no human approval gate** between a data change and publish — that would turn a personal running/media archive into an editorial workflow it doesn't need, and it would fight the
whole point of a k3s CronJob refreshing it unattended. The review loop is: automated checks first, human spot-check on suspicion, not a sign-off step per publish.

- **The validation gate** (above) is the first automated reviewer — most regressions never leave the private network.
- **The CI smoke check** is the second — most of what's left is caught right after deploy.
- **suzuran's generic CronJob monitor** (`cronjob_alerts` in `harus-suzuran`'s `src/suzuran/cluster.py`) already watches every CronJob cluster-wide for `suspend`, no recorded run, or staleness past a
  max age. Once the `iroha-export-public` CronJob exists in harus-k3s, it is covered automatically — no new suzuran wiring needed.
- **Manual spot-check**: visiting `https://azusachino.github.io/iroha/` after a schedule fires. There is no dashboard for "did today's export look right" beyond reading the numbers; that's an
  acceptable cost for a single-operator personal site.

## Rollback path

| Symptom                                                                                      | Action                                                                                                                                                                                                                                                     |
| -------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| A published data snapshot looks wrong (bad numbers, an unmasked field the gate didn't catch) | `git revert` the offending `chore: refresh public-site data export` commit on `main` and push. The path filter on `public-site.yml` re-triggers on that push and redeploys the prior snapshot.                                                             |
| The `public-site.yml` workflow itself is broken (bad build step, bad smoke-check assertion)  | Fix forward on `main`, then re-run via `workflow_dispatch` — no data change is required to redeploy.                                                                                                                                                       |
| The export CronJob is pushing bad data repeatedly                                            | `kubectl patch cronjob iroha-export-public -n harus-core -p '{"spec":{"suspend":true}}'` (same idiom as `notes-git-push` in `harus-k3s/03-core/notes/`) stops further pushes while the export logic is fixed, without touching anything already published. |

A broken deploy is never an outage: GitHub Pages only replaces the live site on a _successful_ `deploy-pages` step, so a failed build or a failed smoke check leaves the previous good snapshot serving
traffic.

## What's still missing

The `iroha-export-public` CronJob itself does not exist yet in `harus-k3s/03-core/iroha/` — only the CLI (`apps/iroha-server/cmd/iroha-export-public`), the cron script
(`ops/scripts/export-public-cron.sh`), and the Containerfile target exist on this side. Nothing has run this pipeline against real data yet; the committed `static/data/*.json` are placeholder zeros.
Standing up that CronJob (image build/import, a sealed git-push credential, the schedule) is harus-k3s's own work, following the `notes-git-push` CronJob as the closest existing pattern —
clone-run-diff-push — with one difference: this job clones a fresh disposable checkout per run instead of mounting a persistent PVC, since there is no long-lived editor process to share a working copy
with.
