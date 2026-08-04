# Iroha frontend design gap matrix

Status: historical audit baseline for `iroha:frontend-field-console:task-1`

Date: 2026-07-18

This document records the frontend redesign baseline before implementation. It uses the existing Svelte source as evidence and separates confirmed gaps from items that still require browser-level
verification.

The route inventory in this file is historical. The former `/share` page and live `/public/v1` API were removed during the static public-site split; use [frontend-request-audit.md](frontend-request-audit.md)
and the current route tree for release-candidate behavior.

## Scope and guardrails

The redesign targets `apps/iroha-web` only. The existing route tree, API consumers, imported data, and design archive remain in place during the first pass.

The first implementation boundary is additive or in-place:

- Modify shared styling and shell code only after the design contract is accepted: `src/routes/app.css`, `src/routes/+layout.svelte`, and shared components under `src/lib/components/`.
- Modify route views in place: Today, Activities, Daily, Dashboard, Sleep, Media, Share, and their detail routes.
- Add the canonical design contract under `docs/` and, if useful, semantic design tokens under `src/lib/design/`.
- Keep `src/routes/design/` and `docs/design-archive/` as review references.
- Do not delete or rename routes, API functions, design experiments, or imported data as part of this epic without a separate explicit decision.
- Do not change backend contracts unless a frontend compatibility audit proves a specific contract defect.

## Current route and data inventory

| Surface         | Route             | Primary data/API consumer             | Current role            | Intended role                |
| --------------- | ----------------- | ------------------------------------- | ----------------------- | ---------------------------- |
| Today           | `/`               | `getBriefing`, `listDaily`            | Daily command center    | Field Console / daily signal |
| Activities      | `/activities`     | `listActivities`, `getPublicSummary`  | Filtered activity cards | Movement archive             |
| Activity detail | `/activities/:id` | `getActivity`, route, samplings, laps | Detailed workout page   | Performance report           |
| Daily           | `/daily`          | `listDaily`, daily aggregates         | Trends plus table       | Pattern atlas                |
| Dashboard       | `/dashboard`      | public summary, activities, routes    | Bento overview          | Long-horizon observatory     |
| Sleep           | `/sleep`          | sleep list, aggregates, segments      | Sleep analysis          | Night report                 |
| Media           | `/media`          | media list and aggregates             | Shelves and charts      | Personal library             |
| Media detail    | `/media/:id`      | media detail                          | Item history            | Library entry                |
| Public site     | separate static site | committed sanitized summary, routes, activities | Public statistics | Editorial public report |
| Design lab      | `/design`         | briefing data plus static variants    | Design experiments      | Temporary review reference   |

The API boundary is already typed in `apps/iroha-web/src/lib/api.ts`. The redesign should preserve current `/api/v1` calls in the private cockpit; sanitized public projections belong to the separate static
site and must not be added back as a live frontend route.

## Gap matrix

| Area                       | Evidence in current source                                                                                                                                                                                  | Status            | Consequence                                                                                                                                    | Planned resolution                                                                                                                              |
| -------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| Canonical design contract  | No `design.md` or equivalent canonical frontend design contract exists in the repository.                                                                                                                   | MISSING           | New pages can drift in typography, spacing, color, and interaction behavior.                                                                   | Add a design contract before broad route edits; make it the source of truth for tokens and page archetypes.                                     |
| Styling ownership          | `app.css` defines Tailwind theme tokens and plain CSS variables; route/component files contain local `<style>` blocks; the separate public site owns its own static presentation.                         | PARTIAL           | The same concept can be styled through multiple systems and become difficult to refactor consistently.                                         | Decide which concerns are semantic tokens, shared primitives, or route-local composition; progressively reduce accidental duplication.          |
| Semantic tokens            | Base and light-theme tokens exist in `app.css`, but domain/chart colors are also hardcoded in `RoutesMap`, `RouteMap`, `SleepTimelineChart`, `FusedActivityChart`, activity detail, and sleep route styles. | PARTIAL           | Light/dark contrast and future palette changes cannot be reasoned about centrally.                                                             | Move visual roles—not raw values only—into semantic tokens, including map, chart, sport, sleep-stage, status, and focus colors.                 |
| Page differentiation       | Today, Dashboard, Daily, and the design lab all use variations of tile/bento/dashboard compositions.                                                                                                        | PARTIAL           | The application can feel like one repeated dashboard even when the information purpose changes.                                                | Assign distinct archetypes: command center, observatory, atlas, archive, report, library, and editorial share page.                             |
| Shared component language  | There are reusable charts, gauges, maps, stat tiles, `RouteIntro`, `DayPicker`, and `ThemeToggle`, but no documented component/state contract.                                                              | PARTIAL           | Reuse exists, but visual and accessibility behavior may diverge between routes.                                                                | Define shared primitives and required states before extracting more abstractions.                                                               |
| Loading/error/empty states | Most routes implement loading and error branches; Today, Activities, Media, and Dashboard include empty or partial sections.                                                                                | PRESENT / PARTIAL | Coverage exists, but copy, layout, and visual hierarchy are inconsistent; stale-data behavior is not a named shared state.                     | Standardize state primitives and add partial, stale, retry, and no-permission variants where applicable.                                        |
| Theme completeness         | `app.css` has dark/light root overrides and `ThemeToggle`; several charts/maps and sleep surfaces retain fixed colors or dark-oriented shadows/backgrounds.                                                 | PARTIAL           | A page can technically switch theme while a chart, map, icon, or overlay remains visually wrong.                                               | Verify every surface in both themes and replace fixed presentation colors with semantic roles where appropriate.                                |
| Responsive system          | Breakpoints vary across files (`520`, `560`, `640`, `720`, `760`, `800`, `820`, `900`); heatmap, tables, and share content intentionally use horizontal overflow.                                           | PARTIAL           | Narrow layouts may be usable but are not governed by one predictable system.                                                                   | Define the 320/375/414/768 review matrix, then retain overflow only where it is an intentional data interaction with an accessible alternative. |
| Motion policy              | No repository-wide `prefers-reduced-motion` rule or documented motion stance was found.                                                                                                                     | MISSING           | New transitions and decorative effects may violate user preferences or become inconsistent.                                                    | Add a reduced-motion contract and ensure decorative motion never carries data meaning.                                                          |
| Accessibility contract     | Charts and key controls have useful ARIA labels; command palette has dialog/listbox semantics; tab-like controls exist on Daily and Media.                                                                  | PARTIAL           | Existing semantics are promising, but keyboard order, focus visibility, tab behavior, and chart fallback content are not verified as a system. | Run a keyboard/semantic audit and define accessible patterns for tabs, filters, calendars, maps, charts, and dialogs.                           |
| Visual regression          | `package.json` provides build, check, format, and unit-test scripts, but no browser screenshot or visual regression command is present.                                                                     | MISSING           | Theme and breakpoint regressions can pass static checks unnoticed.                                                                             | Add a lightweight local review/screenshot workflow or documented manual matrix before final acceptance.                                         |
| Data provenance/freshness  | The UI renders real API data, but freshness/source language is not a shared component or page-wide convention.                                                                                              | MISSING / PARTIAL | Viewers may not know whether a metric is imported, derived, incomplete, or stale.                                                              | Add quiet provenance/freshness treatment to data-heavy surfaces without inventing metrics.                                                      |
| Public privacy boundary    | The private cockpit uses `/api/v1`; the static public site consumes sanitized export artifacts.                                                                                                           | PRESENT           | A future editorial redesign could accidentally expose or imply private-only information.                                                       | Keep the boundary documented and test export DTOs for leakage.                                                                                  |
| Design archive             | `/design` contains multiple Today variants and `docs/design-archive/` contains archive notes.                                                                                                               | PRESENT / PARTIAL | Useful visual history exists, but accepted direction and rejected experiments are not yet clearly labeled.                                     | Keep the archive during migration; record the chosen direction and acceptance evidence before any cleanup.                                      |

## Confirmed gaps versus verification gaps

Confirmed from source inspection:

- No canonical design contract.
- Mixed styling ownership.
- Hardcoded visual colors outside the token system.
- No documented reduced-motion policy.
- No repository-level visual regression command.
- No shared provenance/freshness convention.
- Page archetypes are not formally defined.

Requires browser or runtime verification:

- Whether light mode fails on specific charts, maps, icons, or overlays.
- Whether 320px and 375px layouts overflow or lose important controls.
- Whether tab, calendar, command-palette, and map interactions are keyboard-complete.
- Whether chart fallback text is understandable without visual inspection.
- Whether private/public navigation creates misleading expectations.
- Whether the current design lab variants remain useful on the local imported dataset.

## Acceptance criteria for the audit task

Task 1 is complete when:

1. Every current route and detail route has a named role and data boundary.
2. Confirmed source gaps are documented separately from browser-only risks.
3. The file-level edit boundary is explicit and prohibits unapproved deletion.
4. The later implementation order is clear: design contract, shared foundation, Today, domain archetypes, then quality verification.
5. This document passes repository formatting and `git diff --check`.

## Next dispatch

Dispatch `iroha:frontend-field-console:task-2` only after reviewing this matrix. Task 2 should create the canonical design contract; it should not yet rebuild all routes.
