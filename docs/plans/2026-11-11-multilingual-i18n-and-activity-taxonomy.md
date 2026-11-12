# Multilingual i18n Architecture and Activity Taxonomy

Last updated: 2026-11-11
Status: Draft / Under Review
ADR Reference: [ADR-0012](../adr/0012-multilingual-i18n-taxonomy-and-native-title-preference.md)

## Overview

This plan establishes a comprehensive i18n architecture across Iroha supporting **English (`en`)**, **Chinese (`zh`)**, and **Japanese (`ja`)**. It addresses two critical data integrity problems caused by ad-hoc string matching and brittle language heuristics:

1. **Activity / Workout Records**: Eliminates fractured records where switching iOS between English and Japanese generated duplicate or divergent activity names (e.g. `"ラン"` vs. `"Outdoor Run"`). Introduces a canonical Activity Taxonomy with context-aware disambiguation.
2. **Media Works and Editions**: Eliminates destructive `titleLanguageRank` clobbering where Chinese WeRead edition titles were overwritten by English metadata or Japanese translations. Enforces **Native First** for authored works and **Edition Preserved** for consumed items.

---

## 1. Supported Language Set: { EN, ZH, JA }

| Domain | Principle | Behavior |
| :--- | :--- | :--- |
| **Authored Works** (`tb_media_works`) | **Native First** | `primary_title = original_title` in work's native script (`ja`, `zh`, `en`). Universal English is fallback if native is unknown. |
| **Consumed Editions** (`tb_media_items`) | **Edition Preserved** | `tb_media_items.title` stays in `item.language`. Never overwritten by other editions. |
| **System Taxonomies** (`tb_activities`) | **Universal English Canonical** | Technical domain classifications are stored in universal English (`"Outdoor Run"`, `"Indoor Run"`, `"Walk"`). Localized display is rendered in UI. |
| **Custom Workout Titles** | **Preserved Verbatim** | User-authored titles (e.g. "Tokyo Marathon 2026") are never normalized or overwritten. |

---

## 2. Package Architecture: `apps/iroha-core/i18n`

A pure domain package with zero DB dependencies:

- `lang.go`: `LangEN`, `LangZH`, `LangJA`, `DetectScript(string) Language`.
- `activity_taxonomy.go`: Canonical activity taxonomy cataloging standard Apple Health / Strava activities across EN, JA, ZH:
  - Context-aware resolution signature:

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

  - **Family-Level Canonicals**: Bare labels (`"ラン"`, `"Running"`, `"跑步"`) map to family-level `"Run"`.
  - **Context Promotion**:
    - Promotes to `"Outdoor Run"` if `ctx.Indoor == false` or `ctx.HasRoute == true`.
    - Promotes to `"Indoor Run"` if `ctx.Indoor == true`.
    - Otherwise honestly remains family-level `"Run"`. No location data is fabricated.
  - Similar comprehensive mappings for walking, cycling, swimming, hiking, strength training, HIIT, yoga, and fitness gaming.
  - **Single Owner for Sport Vocabulary**: Unifies sport slugs with `observations.NormalizeSport` and guarantees HAE label ↔ native XML enum slug equality.
- `media_titles.go`: Pure projection over `tb_media_titles`:
  - Only `title_kind = 'original'` claims matching declared `original_language` may set `work.primary_title` and `work.original_title`.
  - Translations never overwrite `work.original_title`.
  - Populates `work.original_language` (AniList `countryOfOrigin`, ISBN metadata).

---

## 3. Provider and Import Refactoring

- **`apps/iroha-providers/parsers/health_auto_export.go`**:
  - Replace brittle 10-line `switch` in `canonicalHaeWorkoutTitle` with `i18n.ResolveActivity`.
  - Construct `ActivityContext` with route presence and parsed distance.
- **`apps/iroha-providers/parsers/apple_health.go`**:
  - Read `HKMetadataKeyIndoorWorkout` from parsed `appleWorkout.MetadataEntries`.
  - Map `HKWorkoutActivityType...` via `i18n.ResolveActivity`.
- **`apps/iroha-imports/media_import.go`**:
  - Remove `titleLanguageRank` clobbering completely from `refreshMediaItemFields`.
  - Maintain `item.Language` and keep `item.Title` in that edition's language.
  - Enforce decision table where translations cannot overwrite `original_title`.

---

## 4. Database Remediation: Migration 00027

Add `00027_canonicalize_activity_titles.up.sql`:

1. Create audit table `tb_activity_title_remediation`.
2. Add `title_origin text not null default 'unknown' check (title_origin in ('system', 'custom', 'unknown'))` to `tb_activity_observations`.
3. Scope updates strictly by `source_kind in ('health_auto_export', 'apple_health_export')` and exact title equality:
   - Bare `'ラン'` rewrites to family-level `'Run'`.
   - Explicit `'屋外ラン'` / `'屋外 ラン'` rewrites to `'Outdoor Run'`.
   - Explicit `'屋内ラン'` / `'屋内 ラン'` rewrites to `'Indoor Run'`.
   - Explicit `'ウォーキング'` / `'屋外ウォーキング'` rewrites to `'Outdoor Walk'`.
   - Explicit `'サイクリング'` / `'屋外サイクリング'` rewrites to `'Outdoor Cycling'`.
4. Backfill empty `sport_type` on historical Chinese-locale rows (`'户外跑步'` $\to$ `'run'`).

---

## 5. Verification Plan

- `apps/iroha-core/i18n`: Unit tests for script detection, taxonomy lookup, and context-aware promotion.
- `apps/iroha-providers/parsers`: Unit tests for HAE and Apple Health workout parsing with multilingual inputs.
- `apps/iroha-imports`: Integration test confirming Chinese WeRead book titles are not clobbered by English or Japanese items.
- Migration tests asserting exact equality, collision fixtures (`朝のウォーキング` untouched), and audit table population.
- Run `make test` and `make check`.
