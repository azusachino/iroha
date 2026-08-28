# Changelog

All notable changes to this project are documented in this file.

The format loosely follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). This project does not yet follow strict semantic versioning guarantees — pre-1.0 releases may change the API
contract between minor versions.

## [Unreleased]

### Changed

- Give every registered design language real ownership of the persistent app-bar/nav shell instead of leaving it fully host-owned and identical regardless of theme. `Shell.svelte`'s contract
  (`ShellThemeProps`) now accepts `brand`/`nav`/`actions` snippets from the host, and each of the six languages restyles that shared markup into its own header via scoped selectors — a dashed
  map-legend for Atlas, flat axis-tick tabs for Grapher, a dotted/wavy-underline masthead for Field Journal, a phase-dot pill for Phenology, a rack-style "mixing console" for Sound Map, and a
  card-catalog drawer strip for Archive — without duplicating the underlying interactive elements, so the existing audited focus order, tap-target sizes, and accessible names carry over unchanged.
- Give every language its own `--tile-surface`/`--tile-shadow` (every data tile app-wide reads these) and its own glyph for the shared "no records" empty-state mark, instead of both being identical
  across all six and differing only by accent color. Atlas's active nav item also gets a small triangle marker ahead of the label, borrowed from cartography's own convention for "the most important
  point." The empty-state glyphs are sourced from each theme's real-world reference rather than invented: bullet-journal notation for Field Journal, a coordinate crosshair for Atlas, a phase-wheel
  glyph for Phenology, paused-channel bars for Sound Map, and the empty-set symbol for Archive.
- Give Field Journal and Phenology's shell root the `font-family` each already declared via `--font-serif` but never actually applied outside a few hand-styled headings, so their whole page reads in
  the theme's typeface rather than falling back to the shared default sans. Give Atlas its own monospace stack instead of reusing Sound Map's byte-identical one. Add `--body-leading` and
  `--heading-tracking`, consumed by `body`/`h1`/`h2`, so line-height and letter-spacing also vary per language.
- Fix the language-switch cross-fade, which was silently inert: its wrapper was `display: contents`, a box-less display mode that opacity/transform transitions cannot render anything on. Give each
  language its own switch transition (lateral pan for Atlas, a soft settle for Field Journal, an overshoot bloom for Phenology, a quick snap for Sound Map, a stamp-like scale for Archive, plain fade
  for Grapher) and its own `--motion-micro`/`--motion-quick-state` hover/press timing.
- Give the shared sport/source badges (`SportBadge`, `SourceBadge`) each theme's own corner radius via the existing `--radius` token instead of a hardcoded pill shape — the only two shared controls
  with no per-language styling; every other shared control (`SelectControl`, `MonthNavigator`, `PeriodToolbar`/`PeriodDrill`, most chart/report components) already had it. Give `LoadingBoundary`'s
  shared progress-bar track its own shape and texture per language the same way the empty-state mark got one.

## [0.4.4] — 2026-08-24

### Added

- Add executable private-route checks for skip-link/main-landmark integrity, heading order, compact visual/focus order, 24x24px standalone controls, and semantic table drill-down controls across the
  existing language, mode, motion, and mobile viewport matrix.
- Add a seeded representative UI/UX scorecard covering Today, Overview, Patterns, Metrics, To-go, and Admin without retaining private screenshots or payloads.
- Introduce a liquid-glass material (backdrop blur + saturate + a layered inset highlight) for floating chrome only — the app bar, primary-nav pill, command palette, and the Domains/Analyze/More and
  month-picker popovers — with a `prefers-reduced-transparency` fallback to an opaque surface on every surface it touches. Data tiles and route content stay opaque.
- Add a named `--motion-*` token set (`micro`, `quick-state`, `data-update`, `language-switch`) in the shared theme package and apply it across four felt vertical slices: `LoadingBoundary`'s loading →
  ready arrival transition, the shared ring gauges' (and three per-language inline rings') value-change sweep, a cross-fade on design-language switching, and a reveal transition on the
  Domains/Analyze/More navigation popover (`@starting-style` + `transition-behavior: allow-discrete`, since it lives inside a native `<details>`).
- Animate Grapher's Today headline numbers (Move, Exercise, Steps, sleep efficiency) on change via a new shared `AnimatedNumber` primitive (`svelte/motion`'s `tweened`), jumping instead of tweening
  across `null`/absent-data transitions.
- Add `make motion-tokens-check` (wired into `make check`): fails if a `var(--motion-*)` reference has no matching definition in the shared theme package.

### Changed

- Give Grapher's frequently used Motion, Night, Patterns, and Library routes a compact utility-title scale while preserving the editorial Today and Overview hierarchy.
- Require an explicit Metrics dimension selection instead of guessing the first catalog value; preserve explicit empty selections and load catalog/series state through the shared latest-request-wins
  async resource.
- Replace mouse-only Patterns table rows in all six design languages with native period buttons that provide keyboard-equivalent evidence navigation.
- Give every design language a real base typeface instead of falling back to `system-ui` — five of six languages, and the base document text in both the private cockpit and the public site, had no
  `--font-sans` of their own.
- List a period selector's month dropdown newest-first, matching the year dropdown's own long-standing convention — every consumer (Motion, Overview, Expenses, Reports, Metrics, Night) listed January
  first regardless of which month was actually most recent.
- Narrow the global `prefers-reduced-motion` rule so it stops decorative/infinite loops without crushing every transition duration to near-zero via `!important`, which foreclosed any deliberate
  reduced-motion result a component might need.

### Fixed

- Add a first-focus skip link and one canonical focusable main landmark, align compact header DOM and visual order, raise audited standalone controls to the product's 24x24px floor, and restore Today
  and Overview's H1-first outline.
- Give empty Today a `Jump to latest recorded day` action and defer its URL synchronization until SvelteKit's router is initialized.
- Normalize the API's `null` dimension list for canonical metrics so dimensionless series such as Steps render instead of failing after catalog load.
- Restore the desktop-centered primary navigation without reopening the compact focus-order fix above — the DOM reorder that fixed 375px focus order had also pushed the nav flush right on desktop; the
  compact "own full row" treatment now targets the app-bar actions instead of the nav, so DOM order still matches visual order at every width.
- Retint dark mode's tile shadow to match the background hue (light mode already did this), and fix a popover shadow that referenced an undefined `--shadow` custom property and always silently fell
  back to a flat black default regardless of theme.
- Stop the Overview route-summaries table from blowing out of its card and off the page — a pre-existing `min-width: auto` grid-overflow bug — and relax its flat `34rem` minimum width so the actual
  content fits without forcing an unnecessary horizontal scroll in a half-width dashboard card.

## [0.4.3] — 2026-08-16

### Changed

- Move the Bangumi->MAL->AniList cross-provider media resolution bridge from two ConfigMap-mounted JSON files (rebuilt and redeployed manually, no queryability) to a `tb_media_ref_bridge` Postgres
  table (migration `00012_media_ref_bridge.sql`). `iroha-job` still loads both hops into an in-memory map at startup; refresh is now a manual, on-demand `media_bridge_refresh` job handled by the
  already-running worker (no new pod or scheduled job), triggered from a new "Media bridge" button on `/to-go` next to the existing AniList/Bangumi sync actions.
- `scripts/build_media_bridge.py` now upserts into Postgres via `psycopg` instead of writing JSON files (still available for local dev convenience).

## [0.4.2] — 2026-08-16

### Added

- Add `scripts/paypay_import.py`, a PayPay transaction-history CSV importer that creates canonical expenses idempotently (safe to re-run against overlapping exports), excluding balance top-ups,
  points, PayPay's spare-change auto-invest, received money, and refunds from spend totals.
- Complete the frontend modularization follow-up from issue #53: extract `library`/`patterns` route business logic into co-located `.svelte.ts` state modules, and reorganize
  `packages/iroha-shared/src`'s 42 flat files into domain subdirectories (`view-contracts/`, `domain/`, `format/`, `components/`, `theme/`).
- Document the month/year/lifetime scope model as a hard rule in AGENTS.md.

### Fixed

- Fix Expenses' "All months" period option, which was reachable but silently did nothing; the category/currency totals charts also needed their metric-series aggregation grain switched to match the
  selected scope width so a year view reflects the real year-wide total instead of only its first month.
- Fix `BarChart`'s horizontal-orientation grid right margin clipping longer formatted value labels (e.g. a JPY total with a thousands separator).
- Fix `ops/images/Containerfile.server`'s `export-public` stage missing `tzdata`, needed for the CronJob's Asia/Tokyo schedule check.

## [0.4.1] — 2026-08-15

### Added

- Add one backend-neutral runtime cache module with Postgres, Valkey, and explicit `none` backends; production process-memory caching is not used.
- Add complete response caching for monthly and twelve-month reports under the `read_reports` namespace.
- Add bounded Postgres cache cleanup for expired and superseded-generation entries, including hourly worker maintenance.
- Add a deterministic large-fixture release performance gate for cold canonical reads, cache hits, mutation freshness, and cache retention.
- Add canonical media provider history and dated AniList activity import, keeping exact consumption events separate from provider state changes.
- Add a distinct Book media family, separate from manga and light novels, ahead of future book-provider adapters.
- Add `apps/iroha-web/src/lib/asyncResource.svelte.ts`, one shared reactive resource primitive (data/loading/error/sticky-ready) that every route's `LoadingBoundary` usage now goes through instead of
  a hand-rolled loading/error/request-counter triple per route.

### Changed

- Make cache identity include the server's effective timezone, so omitted and explicit default-timezone requests cannot collide with another interpretation.
- Make cache population generation-safe and make successful canonical mutations invalidate their dependent namespaces after commit.
- Make known invalidation failures bypass affected cache namespaces until a later invalidation succeeds.
- Keep direct expense records live and canonical while caching derived metric and report representations.
- Make media library filters, current-status totals, completion-year selection, charts, Today updates, and detail history use the same canonical scope.
- Preserve provider date precision and source basis; connector snapshots never become exact daily consumption sessions.
- Make period navigation (year/month selectors, including arrow-key shifts) clamp to the real imported data range instead of the current date.
- Consolidate each theme's page-shell width into a single `--shell-width` custom property instead of a literal duplicated across the base rule, the mobile breakpoint, and (for three themes) the footer
  padding calc.

### Fixed

- Refresh expense metric/report reads after expense create, replace, and delete.
- Refresh activity route representations after geocode changes and media/report representations after media resolution changes.
- Correct media detail evidence to distinguish exact events from dated provider updates and hide internal observation snapshots.
- Normalize Grapher media-detail titles to a regular responsive record-heading scale instead of a landing-page hero scale.
- Fix Grapher Activities/Dashboard and the shared movement-series chart replacing their whole DOM structure with a status message on every data refetch (period/filter change) instead of only the first
  load; the same non-sticky-ready bug was live on Motion, Night, Patterns, and Overview and is now structurally prevented by `LoadingBoundary` taking an async resource directly rather than
  caller-computed loading/ready booleans.
- Fix the To-go page's oversized headline crowding the actual task list on what's meant to be a short, fast, daily-use page.

## [0.4.0] — 2026-08-14

### Added

- Add the canonical private expense ledger with idempotent create, list, detail, replacement, and tombstone-delete APIs.
- Add synchronous monthly reports with typed movement, sleep, daily health, media, and expense sections.
- Add the general local CLI transport and expense/report commands, plus private `/expenses` and `/reports` cockpit pages.

### Changed

- Keep expenses and reports out of the sanitized public export; add a serialized projection regression test for that boundary.
- Adopt `requests` for the Python CLI transport and lock its dependencies with `uv`.
- Document the v0.4 local-agent boundary: OCR may happen outside Iroha, while Iroha validates and stores canonical JSON deterministically.
- Move canonical theme identities, route compositions, charts, controls, and adopted design compositions into `@iroha/shared`; both the private cockpit and public design workbench consume the same
  assets.
- Make monthly reports chart-first with twelve-month comparison, truthful empty/partial months, canonical units, provenance, and exact evidence below the visual summary.
- Add the v0.4 release-candidate contract gates: deterministic seeded integration/runtime checks, six-language light/dark browser coverage, mobile overflow/reduced-motion checks, and the shared-theme
  boundary check.
- Harden compact navigation and shared theme layouts at 320, 375, 390, and 414 CSS pixels without changing the private/public data boundary.

## [0.3.1] — 2026-08-12

### Added

- Publish the explicitly approved activity's grapher detail, including its full route, heart-rate samples, and laps, through `activity-details.json`.
- Show the public site's iroha version in its header/footer.

### Changed

- Shorten the README to the private/public boundary, quick start, and documentation references.

## [0.3.0] — 2026-08-11

### Fixed

- The favicon/app icon was only ever wired through a client-hydrated `<link rel="icon">` in `+layout.svelte`, so bookmarks, "Add to Home Screen", and a direct `/favicon.ico` request all missed it.
  Both `iroha-web` and `iroha-public-site` now ship a real `favicon.ico`/`favicon.svg`, `apple-touch-icon.png`, and a web manifest, linked directly in `app.html`.
- The Dashboard's route footprint required an explicit "Load route footprint" click before it would render, a 2026-08-04 change that turned out to be the wrong call. It loads automatically again on
  page load, same as the rest of the Dashboard's data.
- The Night page's daily bar chart rendered every date label as an identical, truncated `2026-0…` in the atlas, field-journal, and sound-map themes, since a full `yyyy-MM-dd` never fit a narrow
  per-bar slot. Labels now use a short `Aug 4` form; the full date remains available on hover.
- sound-map Dashboard's sport-breakdown legend truncated long category names (`FitnessGaming`, `High Intensity Interval Training`) with no way to see the full name; each band now exposes it via
  `title` on hover.

### Added

- The Dashboard's sport-breakdown chart (archive's "Sessions by sport", sound-map's "Sessions by band") is now clickable — both the bar and its legend row navigate to `/activities?sport=<key>`,
  landing on the Activities list pre-filtered to that sport.
- Today and Overview only ever reflected activity data across all six curated themes, even though sleep and media have been part of the product for a while. Overview's stats row now includes Sleep
  (recent average asleep time) and Media (completed count) alongside distance/activities/time/routes. Today now shows a Media section (mirroring the existing Activities section) — the underlying
  briefing API already returned media events, but the curated themes' Today components never received them as a prop.
- The Patterns page's Apple Health Move/Exercise/Stand rings were only ever wired into the ungated fallback theme, not any of the six curated themes — they showed a generic step chart with no
  recognizable ring visualization. Added a rings hero (reusing the same `RingGauge` component) to the top of all six themes' Patterns page.
- Replaced the hand-rolled, non-interactive SVG/div bar charts on the Patterns and Night pages (all six themes, 12 chart instances) with a new shared `BarChart` component built on the same ECharts
  stack `LineChart` already used elsewhere — every chart now has a real hover tooltip and, on Night, click-to-select. Each theme keeps its own accent color and bar orientation; move-goal-closure and
  sleep-efficiency, previously encoded only as an ambiguous bar-color gradient, are now a proper secondary line series visible in the tooltip.
- Every media item's `description` has always been empty — no AniList/Bangumi provider ever wrote to it, so every media detail page across every theme showed the same generic fallback text. AniList
  sync now fetches `description(asHtml:false)` and writes it to `tb_media_works.description` on both create and reconcile; the frontend renders it with `white-space: pre-line` so paragraph breaks
  survive instead of running into one wall of text. Bangumi's collection API doesn't include summaries at all (it needs a separate per-subject detail call, a bigger N+1-shaped change) —
  Bangumi-sourced items still show the fallback, tracked as a follow-up.

## [0.2.0] — 2026-08-05

### Added

- `publicexport.Validate` — a schema/privacy regression gate that `iroha-export-public` runs before writing any output file, catching an unprefixed activity ID, a negative total/metric, or an
  out-of-range route coordinate before it reaches the public site.
- `meta.json` in the public-site export, carrying `generated_at`; the public site now shows "Data as of \<date\>" instead of implying its data is live.
- A post-deploy smoke check in `.github/workflows/public-site.yml` that curls the deployed `/iroha` base path and its `data/*` snapshot files before the workflow is considered successful.
- `make media-bridge-build` (`scripts/build_media_bridge.py`) builds the Bangumi→MAL→AniList media resolution bridge cache from BangumiExtLinker/Fribb into the `map[string]string` shape
  `TwoHopMediaRefBridge` expects — this data was previously only ever produced ad hoc by the throwaway `media_bridge_explore.py` coverage script, so the bridge env vars
  (`IROHA_BANGUMI_BRIDGE_PATH`/`IROHA_MAL_ANILIST_BRIDGE_PATH`) had nothing real to point at. Added `TestLoadTwoHopMediaRefBridge` covering the on-disk loader, which previously had no direct test
  coverage.
- `GET /api/v1/media/resolution-tasks` and `PATCH /api/v1/media/resolution-tasks/{taskId}` give the media cross-provider resolution inbox (`tb_media_resolution_tasks`) an API and a small `/admin`
  panel — dedupe-candidate and progress-conflict tasks the resolver already wrote were previously invisible, with no way to list or act on them. Resolving records the operator's decision in
  `resolution_json` only; it does not merge media rows or apply a progress choice — that remains a documented "later" item.

### Changed

- `ci.yml` and `public-site.yml` provision tools with `jdx/mise-action` against the checked-in `.mise.toml`; local and CI resolve the exact same pinned tool versions. The obsolete Nix flake was
  removed after the 0.3.0 release. Updated `README.md`, `CONTRIBUTING.md`, `AGENTS.md`, and `docs/dev-runtime.md` to match.

### Fixed

- Synced media never had cover art: the AniList and Bangumi connectors fetched list data but never requested/mapped either API's cover image field, so every `tb_media_items` row has always had an
  empty `cover_image_url`, regardless of how long ago it synced. AniList's query now selects `coverImage{large}` and Bangumi's `subjectRecord` now decodes `images.large`; both map into
  `observations.Media.CoverImageURL`, which the import pipeline already threaded through on both insert and reconcile-update. A resync backfills existing items, since the update path only skips a
  column when the incoming value is empty.
- Cross-provider media resolution was silently creating duplicate items instead of deduping them: `titleYearCandidates`'s `Where(...).Or(...)` chain OR'd the base scope condition into the whole clause
  instead of ANDing it against each title alternative, so it matched almost every item released in the same calendar year regardless of title (verified in prod: 1961/2000 items sat as false-positive
  `dedupe_candidate` tasks). Separately, an exact-calendar-year filter made same-work items structurally unmatchable whenever providers disagreed on release date by more than a few months, and a real
  candidate match was never actually used — the resolver logged an advisory task but still created a new item every time. `titleYearCandidates` now scopes on `media_type` + `item_role` and a ±400-day
  release-date window instead of an exact year, and `resolveMediaItem` auto-attaches to a single unambiguous candidate (logging an already-resolved audit task) while an ambiguous multi-candidate match
  still opens a task for a human, same as before.
- Even after the above, exact-string title matching still missed real cross-provider duplicates whenever the two providers rendered the same title slightly differently — a trailing bracketed reading
  gloss kept on one side and dropped on the other, the same in-title gloss in a different bracket style (fullwidth `（）` vs CJK `《》`), or a fullwidth `～` vs ASCII `~` plus incidental spacing
  differences (all verified against real production duplicates a plain lowercase/whitespace-normalized match let through). `normalizeMediaTitle` now NFKC-folds the title (which also collapses
  fullwidth punctuation to ASCII) and strips bracketed annotations before comparing; `titleYearCandidates` fetches its scoped candidate set from SQL and applies this normalization in Go on both sides,
  since neither transform has a cheap SQL equivalent.
- One more real duplicate shape survived even that: Bangumi running a title straight into its tilde-delimited subtitle with no space, AniList inserting one (`...好きすぎる～真摯...` vs
  `...好きすぎる ～真摯...`). Collapsing whitespace doesn't help when there's no redundant whitespace to collapse — the two sides simply disagree on whether a separator space exists at all.
  `normalizeMediaTitle`'s comparison key now drops whitespace entirely instead of collapsing it to single spaces.
- Blanket bracket-stripping was itself unsafe: a collision-safety unit test (`TestNormalizeMediaTitle_CanonicalKeyCollisionSafety`) caught it collapsing "薬屋のひとりごと（第二期）" (a real Season 2
  marker) onto "薬屋のひとりごと" (Season 1), and "Fullmetal Alchemist (2003)" onto "... (2009)", a different remake — bracketed content isn't always a reading gloss. `normalizeMediaTitle` now only
  strips a bracketed span when its content is entirely hiragana/katakana; kanji, digits, and Latin letters (season markers, years, initialisms) are left in place.
- Measured directly against production (every Bangumi manga ID actually synced vs. `bangumi_to_mal.json`'s keys): the Bangumi→AniList bridge covers 0% of manga, only anime (~66%, matching the existing
  spike number). `docs/media-sync-connectors.md` previously implied the bridge was a general cross-provider mechanism with an anime-shaped tail; corrected to state that title/date matching is the
  _sole_ cross-provider dedup path for manga, not a fallback — manga is the majority of most Bangumi+AniList libraries, so this is the more consequential path, not the less.
- One duplicate shape no normalization could fix: a provider omitting a work's entire trailing subtitle rather than reformatting it (verified real case: Bangumi's title for a manga ran straight to the
  end where AniList's had an additional "～世界最強はオレだけど、世界最カワは妹に違いない～" clause, absent, not reworded). `titlePrefixCandidates` now detects a ≥12-rune shared prefix as a
  lower-confidence signal — but deliberately only ever opens a `tb_media_resolution_task` for a human, never auto-attaches, since two different works sharing a long specific opening and diverging only
  in the subtitle is exactly the collision case `TestNormalizeMediaTitle_CanonicalKeyCollisionSafety` already guards against for bracketed content. (25 runes was the original floor; lowered after a
  second real pair, "異世界グルメで成り上がり無双", showed only a 14-rune shared prefix — since this path never auto-merges, a false positive only costs a dismiss click, so erring low was the safer
  call than erring high and staying silently invisible.)
- Every media list/detail view (all 6 UI variants) displayed a percentage next to progress even when no total was known, via `formatPercent(item.progress_percent ?? 0)` — coercing null to 0 defeated
  `formatPercent`'s own correct "—" handling and rendered as a confident, fabricated "0%" for an item with real logged progress (e.g. 46 chapters read, but no known total). Replaced with
  `formatProgressCount`, which shows a done/all count when the total is known and just the done count otherwise — never a fabricated percentage.
- `titlePrefixCandidates`' rune-count floor turned out insufficient on its own: verified in prod that "My Hero Academia" vs "My Hero Academia Season 2" (and equivalents — `Komi Can't Communicate` vs
  `... Part 2`, `進撃の巨人 第三季` vs `... Part.2`) both clear a 12-rune shared prefix, but a missing season/part marker means a different installment of the same franchise, not a duplicate. No
  rune-count threshold can distinguish that from a genuinely omitted subtitle — the trailing content itself has to be inspected. `titlePrefixMatch` now rejects a match when the remainder after the
  shared prefix looks like a season/part/cour marker (English `Season N`/`Part N`/`Cour N`/`OVA`/`Movie`, Japanese `第N期`/`最終季`/`劇場版`, Chinese equivalents).
- That keyword list is necessarily incomplete — a franchise doesn't have to spell "Season 2" to mean it. Gintama's real sequels are the base title plus a single mark (`銀魂` → `銀魂゜` → `銀魂°`),
  which matches no keyword pattern. Measured every case found in prod: every false-positive remainder (season/part markers) was 1–7 runes; every genuine omitted subtitle was 25+ runes. Added
  `titleRemainderMinRunes = 10` as a general-purpose backstop in that gap — rejects any prefix match whose remainder is too short to plausibly be a subtitle clause, regardless of whether it matches a
  known keyword, catching symbolic sequel marks no enumerable list could cover.
- The media detail page's title heading used a viewport-only `clamp()` with no regard for title length, so a 100+ character title (not uncommon for these titles) rendered as many lines of oversized
  text. `heroTitleFontSize` (`$lib/hero-title.ts`) scales each theme's own clamp down as title length grows; wired into the default page and all five theme `MediaDetail.svelte` components.
- `formatProgressCount`'s done/all count still looked wrong for a real case: Bangumi reports `status: completed, position: 10` for a finished anime season but never a numeric `total`, so "10 episodes"
  sat next to a "completed" label with no way to tell it was actually finished, and the progress bar/ring next to it rendered empty (`progressValue()`/`boundPercent(item.progress_percent)` had no
  total to derive a fill from). A completed item's position _is_ its total by definition. `effectiveTotal` (internal to `$lib/format.ts`) infers this when `status === "completed"` and no total was
  recorded; `formatProgressCount` and the new `progressPercent` (replacing ad hoc `boundPercent(item.progress_percent)` bar/ring math across all 6 list views and the shared detail `progressValue()`)
  both use it, so a completed item now reads "10/10 episodes" with a full bar instead of a bare, ambiguous count next to an empty one.
- `titleSeasonMarkerPattern` anchored the keyword at the very start of the remainder, so a season marker introduced with a leading separator slipped through: "Ore dake Level Up na Ken: Season 2 -
  Arise from the Shadow" vs "Ore dake Level Up na Ken" put a colon at remainder position 0, one character before "season" ever appears. Titles routinely introduce a subtitle/season marker this way
  ("Title: Season 2", "Title - Part 2"); the pattern now allows one optional leading separator before the keyword.

## [0.1.4] — 2026-08-04

### Added

- **Private control room** — `/admin` and the front-page Daily to-go lane provide personal task tracking, recent durable-job visibility, and named AniList/Bangumi sync triggers.
- **Release identity** — the root `VERSION` file drives image tags and the frontend's subtle version note; sleep sessions now have a detail endpoint/page as well.
- **Frontend request audit** — persisted route-by-route traffic findings, a scoped control-room job feed, and lazy Daily history loading with browser regression coverage.

### Removed

- **`/public/v1`.** The sanitized public API surface, and the in-app `/share` page that rendered it, were never actually exposed to the internet — dead weight of an open-CORS route, a second
  rate-limit budget, and six themed frontend variants serving a page nobody could reach. A separate static site, deployable to GitHub Pages and kept fresh by a k3s CronJob (not a self-hosted GitHub
  Actions runner — see [roadmap Milestone 7](docs/roadmap.md#milestone-7-privacy-and-publishing)), takes over that role instead.

### Added

- `GET /api/v1/activities/summary` and `GET /api/v1/activities/routes` — private equivalents of the removed public endpoints. The dashboard and activities pages depended on the public routes directly
  for their own totals/routes-map widgets, not only the removed share page.
- `apps/iroha-server/pkg/publicexport` — the sanitized activity DTO and query logic, extracted out of the HTTP layer so both the new private routes and the static-export CLI below reuse it.
- `apps/iroha-server/cmd/iroha-export-public` — writes a static `summary.json`/`activities.json`/`routes.geojson` snapshot for the public site (`make export-public`, `OUT=...`).
- `apps/iroha-public-site` — a new SvelteKit app (adapter-static, fully prerendered) rendering that snapshot, deployable to GitHub Pages as a project page (`make public-site-build`,
  `BASE_PATH=/iroha`).
- `ops/scripts/export-public-cron.sh` and the `export-public` target in `ops/images/Containerfile.server` — the k3s CronJob container that regenerates and pushes the snapshot from inside the private
  network (the CronJob resource itself lives in harus-k3s). `.github/workflows/public-site.yml` builds and deploys the site on an ordinary GitHub-hosted runner whenever that push lands.
- `make image-server` / `image-job` / `image-db-migrate` / `image-web` / `image-export-public` / `images` — build with Podman and import straight into the local k3s node's containerd store, for the
  `azusachino.icu/iroha-*` local image naming already used elsewhere in the homelab.

### Changed

- **Read response cache** — successful JSON reads for briefing, activities, sleep, daily, and media are cached by canonical method/path/query keys. Import completion advances all five read namespaces,
  and Valkey keys use the `iroha:cache:v1:` application prefix; tasks, jobs, and mutations remain uncached.
- GORM query logs are routed through the app's `*slog.Logger` instead of GORM's own ANSI-colored default logger; `iroha-server`/`iroha-job` now emit uniform JSON log lines throughout.
- `ops/local-dev/` split into `ops/local-dev/` (Podman Compose orchestration: compose files, initdb, README) and `ops/images/` (environment-agnostic build definitions: both Containerfiles, Caddyfile,
  migrate-entrypoint.sh) — these were always consumed by both local dev and manual production builds, but living under a folder named "local-dev" obscured that.
- `ops/images/Containerfile.web`'s `PUBLIC_IROHA_API_BASE` build-arg now defaults to empty (same-origin) instead of the local-dev convenience value `http://127.0.0.1:8080`, so images built for k3s no
  longer bake in a value that only makes sense on a developer's machine.
- The `migrate` Containerfile target/image is renamed `db-migrate` for clearer naming in the shared homelab image list.

### Fixed

- A media-import race: two `iroha-job` workers resolving the same `(provider, external_id)` at once could both pass a lookup-then-insert "not found" check and collide on `tb_media_external_refs`'s
  unique constraint, aborting the job transaction. Replaced with `INSERT ... ON CONFLICT (provider, external_id) DO NOTHING` plus a re-fetch.
- The `goose` binary bundled into the `db-migrate` image was built with cgo enabled in the glibc build stage and copied into the alpine/musl final stage, so it couldn't run (`goose: not found`,
  despite the file existing). Built with `CGO_ENABLED=0` instead.

## [0.1.1] — 2026-07-20

### Removed

- **JWT authentication.** `/api/v1` is now unauthenticated, matching the actual deployment model: iroha is a single-user personal project running on a private NAS, and only the (not yet exposed)
  `/public/v1` share surface is ever meant to be reachable from outside that network. The JWT layer was always a self-signed static credential standing in for network-level access control — it added a
  secret-provisioning step (`IROHA_JWT_SECRET`, token minting) without changing who could actually reach the API. Removed `golang-jwt/jwt/v5`, the `AuthConfig`/`IROHA_LOCAL_NO_AUTH`/`IROHA_JWT_*`
  config surface, and the web client's `PUBLIC_IROHA_API_TOKEN` build argument. Rate limiting is unchanged and still guards both `/api/v1` and `/public/v1`.

## [0.1.0] — 2026-07-20

First tagged release. Iroha owns personal running/fitness, sleep, daily activity, and media-consumption history end to end: raw exports in, canonical Postgres/PostGIS facts out, private and
sanitized-public read surfaces on top.

### Added

- **Import core** — raw-file archive with content-hash dedupe, a durable `tb_import_jobs` lifecycle, and a Postgres-backed `tb_jobs` queue (`FOR UPDATE SKIP LOCKED`) consumed by the standalone
  `iroha-job` worker.
- **Apple Health domains** — activities (workouts, routes, laps, per-sample streams), sleep sessions and stage segments, and daily summaries/metrics (Move/Exercise/Stand rings, steps, distance,
  flights, body vitals), all reconciled from a full-export snapshot rather than blindly appended.
- **Media sync** — AniList and Bangumi connectors (`POST /api/v1/media/sync/{connectorId}`) with canonical media items/events, aggregates, and a MAL↔AniList/Bangumi bridge cache.
- **Postgres/PostGIS cache backend** — durable, invalidation-aware cache (`tb_cache_entries`) with Valkey kept as an optional compatibility backend; durable geocode reverse-lookup with retry/backoff
  against Nominatim.
- **Private API v1 contract** — JWT authentication, per-route rate limiting, an OpenAPI v1 spec (`docs/contracts/openapi.yaml`), and a route-inventory test that fails the build if a registered route
  drifts from the contract.
- **Themed Svelte cockpit** — a switchable multi-design frontend (grapher, field journal, atlas, phenology, sound-map, archive, and default themes) sharing one typed theme registry and route renderer.
- **Sanitized public views** — `/public/v1` activity/route/summary projections, always derived, never the canonical store.
- **Local dev stack** — Podman Compose profile for Postgres/PostGIS, server, worker, and web behind a Caddy edge container, with `scripts/dev_stack.py` owning lifecycle and migrations.
- **k3s deploy readiness** — a standalone `migrate` container image (goose CLI + bundled migrations) for running as a Job/initContainer, non-root `server`/`job`/`migrate` containers, bounded-retry
  Postgres connect on startup, and graceful shutdown (SIGTERM-aware drain) in `iroha-server`.

### Changed

- Cache backend defaults to Postgres; Valkey is opt-in via `IROHA_CACHE_BACKEND`.
- `rawfiles` moved from `iroha-server` into the shared `iroha-runtime` module so both the API and the worker own the same file-storage boundary.

### Fixed

- Geocode retry storms now back off instead of hammering Nominatim on rate-limit responses.
- Local stack startup sequencing (dependencies before app containers, migrations before server).

[0.3.0]: https://github.com/azusachino/iroha/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/azusachino/iroha/compare/v0.1.4...v0.2.0
[0.1.4]: https://github.com/azusachino/iroha/compare/v0.1.1...v0.1.4
[0.1.1]: https://github.com/azusachino/iroha/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/azusachino/iroha/releases/tag/v0.1.0
