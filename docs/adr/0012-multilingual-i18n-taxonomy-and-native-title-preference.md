# ADR-0012: Multilingual i18n Architecture, Canonical Activity Taxonomy, and Native Title Preference

- Status: Proposed; awaiting owner's final review, not implementation approval
- Date: 2026-11-11
- Scope: Multilingual representation across `{en, zh, ja}`, HealthKit / HAE activity taxonomy, observation provenance vs. canonical activity projection, authored work title preference, and edition reading language preservation
- Related: [ADR 0001](0001-provider-observations-and-canonical-records.md), [ADR 0002](0002-provider-adapter-contracts.md), [ADR 0008](0008-health-auto-export-http-intake.md), [ADR 0011](0011-provider-neutral-reading-evidence.md)
- Detailed proposal and evidence: [Multilingual i18n Architecture and Activity Taxonomy](../plans/2026-11-11-multilingual-i18n-and-activity-taxonomy.md)

## Context

Iroha aggregates heterogeneous personal data across two major domains:

1. **Health and Activities**: Apple Watch / HealthKit data ingested via native Apple Health exports (`export.xml`) and Health Auto Export (HAE) JSON v2 webhooks.
2. **Media and Reading**: Books, manga, anime, and movies ingested from WeRead, Hardcover, Bangumi, and AniList.

Both domains experienced systematic data pollution due to ad-hoc string comparisons and brittle language heuristics:

### 1. Activity Domain: Fractured Workout Records Across iOS Locales

- Apple HealthKit internally models workouts as integer enums (`HKWorkoutActivityTypeRunning`, `HKWorkoutActivityTypeWalking`).
- Native Apple Health XML exports expose the enum constant string (`HKWorkoutActivityTypeRunning`), parsed into `Title: "Running"`.
- Health Auto Export (HAE) exports `w.Name` formatted by Apple's localized string API in the device's active system locale at the moment of export (e.g., English `"Outdoor Run"`, Japanese `"屋外ラン"` / `"ラン"`, Chinese `"户外跑步"` / `"跑步"`).
- The previous implementation in `canonicalHaeWorkoutTitle` (`health_auto_export.go:802`) used an ad-hoc 10-line `switch` that only matched `"屋外ラン"` and `"屋内ラン"`.
- When iOS was set to Japanese, Apple Watch workouts labeled `"ラン"` or `"ウォーキング"` fell through to `default: return name`.
- Consequently, the personal database accumulated fractured workout records: some titled `"Outdoor Run"`, some `"Running"`, and a cluster of records titled `"ラン"` during periods when the user lived in Japanese iOS. Switching back to English restored English strings, leaving the timeline permanently fragmented.
- Furthermore, indoor vs. outdoor is **not** an activity type enum in HealthKit; it is carried by `HKMetadataKeyIndoorWorkout` (parsed into `appleWorkout.MetadataEntries` but previously ignored) and GPS route presence. Statically mapping bare `"ラン"` to `"Outdoor Run"` would fabricate location data for treadmill runs.

### 2. Media Domain: Clobbered Edition and Work Titles

- In `apps/iroha-imports/media_import.go:689-766`, title precedence was hardcoded via `titleLanguageRank` with an arbitrary script hierarchy:
  $$\text{JPN (1)} > \text{ENG (2)} > \text{CHN (3)}$$
- This heuristic was written specifically for Anime (Bangumi vs. AniList).
- When applied to **books**:
  - Chinese books read on WeRead (e.g. 《三体》) contain CJK ideographs (rank 3). When touched by an English provider (Hardcover) or Latin metadata, Latin script (rank 2) wins, overwriting the Chinese edition title with English.
  - English technical manuals (e.g. *Designing Data-Intensive Applications*) were vulnerable to being overwritten by Japanese translation titles (rank 1).
  - Edition items (`tb_media_items`) representing Chinese editions lost their authentic reading language.
  - Parent works (`tb_media_works`) had their `original_title` clobbered by translation editions.
- In `00002_media_schema.up.sql`, `tb_media_titles` was already designed with `(language, script, region, title_kind, provider, is_primary)`, and `tb_media_works.original_language` existed in schema, but no adapter populated `original_language`.

---

## Decision

Propose a comprehensive, unified multilingual architecture supporting the three primary languages of personal life: **English (`en`)**, **Chinese (`zh`)**, and **Japanese (`ja`)**:

### 1. Dual Domain Principles: Authored Works vs. System Taxonomies

1. **Authored Works (`tb_media_works`, `tb_media_items`) — Native First**:
   - Authored works originate in a specific linguistic and cultural context (`original_language` $\in$ `{zh, ja, en}`).
   - The native title is preferred: `primary_title = original_title` in the work's native script.
   - If `original_language` is unknown or unsupported, universal English (`en`) is the default fallback.
   - Translations from other languages never overwrite `work.original_title` or native `work.primary_title`.
2. **Consumed Editions (`tb_media_items`) — Edition Preserved**:
   - An item represents a specific edition consumed or read by the user (e.g. Chinese WeRead edition, English Hardcover edition).
   - `item.language` is explicitly recorded.
   - `item.title` stays in that edition's reading language and is **never** clobbered by foreign-language editions or script hierarchies.
3. **System Taxonomies (`tb_activities`, `tb_activity_observations`) — Universal English Canonical**:
   - System concepts (workout categories, sports, metrics, statuses) are domain taxonomy classifications, not authored artistic works.
   - Their canonical database representation (`sport_type`, system `title`) is universal English (`"Outdoor Run"`, `"Indoor Run"`, `"Walk"`).
   - Localized presentation is rendered in the UI based on user display preferences.
4. **User Custom Titles — Preserved Verbatim**:
   - If a user authors an authentic custom workout title on Apple Watch or Strava (e.g., `"Sub-40 10k Tempo"`), it does not match system activity synonyms and is preserved verbatim in `tb_activities.title`.

---

### 2. Activity Resolution & Taxonomy Engine

5. **Context-Aware Activity Resolution**:
   - The resolver signature in `apps/iroha-core/i18n` takes both raw label and workout context:

     ```go
     type ActivityContext struct {
         Provider   string
         SourceKind string
         Indoor     *bool    // from HKMetadataKeyIndoorWorkout
         HasRoute   bool     // len(RoutePoints) > 0
         DistanceM  *float64
     }
     func ResolveActivity(raw string, ctx ActivityContext) ActivityClassification
     ```

   - **Family-Level Canonicals**: Bare labels (`"ラン"`, `"Running"`, `"跑步"`) match family-level `"Run"`.
   - **Evidence-Based Promotion**:
     - Upgrade to `"Outdoor Run"` only when `ctx.Indoor == false` or `ctx.HasRoute == true`.
     - Upgrade to `"Indoor Run"` when `ctx.Indoor == true`.
     - If neither route nor indoor metadata is present, honestly remain family-level `"Run"`. No location data is fabricated.
6. **Single Owner for Sport Vocabulary**:
   - `i18n.ActivityTaxonomy` is the sole owner of canonical sport slugs (`"run"`, `"walk"`, `"ride"`, `"swim"`, `"hike"`, etc.).
   - `observations.NormalizeSport` is refactored to consult the taxonomy.
   - Exhaustive tests assert that HAE localized labels (EN/JA/ZH) and native XML enums resolve to identical sport slugs, eliminating cross-source deduplication splits on `(started_at, sport_type)`.
7. **Provenance Tracking (`title_origin`)**:
   - Add `title_origin text not null default 'unknown' check (title_origin in ('system', 'custom', 'unknown'))` to `tb_activity_observations`.
   - Native Apple Health XML exports are classified as `'system'`.
   - HAE inputs matching taxonomy synonyms are classified as `'system'`.
   - Free-form titles are classified as `'custom'`.

---

### 3. Media Works & Items Title Policy

8. **Title Policy as a Pure Projection Over `tb_media_titles`**:
   - Do not create a parallel ranking system. Work titles are derived as a pure projection over candidate rows in `tb_media_titles`.
   - **Explicit Authority Table**:
     - Only `title_kind = 'original'` claims matching the work's declared `original_language` may set `work.primary_title` and `work.original_title`.
     - Localized translation claims (`title_kind = 'localized'`) are recorded in `tb_media_titles` but **never** overwrite work-level original titles.
     - Bangumi's legacy `is_primary = true` on `NameCN` is ignored for Japanese-original works.
9. **Populating `original_language` and `item.language`**:
   - Populate `work.original_language` from authoritative provider declarations:
     - AniList: `countryOfOrigin` (`"JP" -> "ja"`, `"CN" -> "zh"`, `"KR" -> "ko"`).
     - WeRead / Hardcover / ISBN: language metadata from catalog.
   - Fallback when unknown: keep current `primary_title`. Never guess `original_language` from title script.
10. **Elimination of `titleLanguageRank`**:
    - Remove `titleLanguageRank` entirely from `apps/iroha-imports/media_import.go`.
    - Distinct editions link to the same canonical `tb_media_works` without overwriting edition titles.

---

### 4. Database Remediation & Evidence Preservation

11. **Evidence vs. Projection Separation**:
    - `tb_activity_observations.title` represents immutable provider evidence.
    - `tb_activities.title` represents the canonical projection.
    - Reprocessing historical raw files through the updated parser (`parser_reprocess` job) heals both observations and canonical projections for all retained raw files.
12. **Migration 00027 (`00027_canonicalize_activity_titles.up.sql`)**:
    - Scope updates strictly by `source_kind in ('health_auto_export', 'apple_health_export')` and exact title equality (`WHERE title = 'ラン'`).
    - Bare `'ラン'` rewrites to family-level `'Run'`.
    - Explicit `'屋外ラン'` rewrites to `'Outdoor Run'`, and `'屋内ラン'` rewrites to `'Indoor Run'`.
    - Audit table `tb_activity_title_remediation` logs `(activity_id, old_title, new_title, migrated_at)` to guarantee auditability.
    - Also backfills empty `sport_type` on historical Chinese-locale rows.

---

## Alternatives Considered

### Retain Ad-Hoc `switch` and Patch Individual Strings

Rejected. Hardcoding individual string cases in parser files creates an unending cycle of locale bugs whenever watchOS renames a label or a user switches languages. A structured taxonomy with synonym normalization is the industry standard for wearable telemetry.

### Use Naive Unicode Script Detection (`DetectScript`) for Media Originality

Rejected. Script does not equal language. Japanese works like *SPY×FAMILY*, *BLEACH*, or romaji titles (*Kimi no Na wa*) use Latin script; Chinese and Japanese share thousands of kanji/hanzi characters. Determining originality from script would reproduce the exact defects of `titleLanguageRank`. Explicit provider declarations must govern.

### Unconditionally Map Bare `"ラン"` to `"Outdoor Run"`

Rejected. Fabricates location data. A plain `"ラン"` or `"Running"` on Apple Watch can be an indoor treadmill run. Conflating bare run with outdoor run corrupts pace and environmental analysis.

---

## Consequences

Positive:

- Historical and future workout timelines remain completely unified regardless of iOS system language switches.
- Cross-source deduplication between HAE and Apple Health native exports operates without title or sport slug collisions.
- Chinese book editions read on WeRead retain their authentic Chinese titles; English technical books retain their original English titles.
- Japanese anime and manga retain their authentic native Japanese titles without being clobbered by English translations.
- All titles remain fully searchable and discoverable in `tb_media_titles`.

Costs and Prerequisites:

- `apps/iroha-core/i18n` introduces an activity taxonomy catalog requiring ongoing maintenance for new Apple HealthKit activity types.
- Adapters must be extended to populate `original_language` and `item.language`.
- Migration 00027 requires integration testing against fixture databases with simulated English, Japanese, and Chinese workout records.
