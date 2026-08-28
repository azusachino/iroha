# Iroha frontend design contract

Status: working contract for `iroha:frontend-field-console`

Date: 2026-07-18

This is the visual and interaction contract for the Iroha web application. It comes before the route redesign so that the redesign produces one product with different page compositions, rather than a
collection of unrelated themes.

The contract uses Hallmark as a review discipline: preserve the existing route and data boundaries, lock the visual system before broad edits, and require structural variety between pages. See the
[Hallmark repository](https://github.com/nutlope/hallmark) and its [design skill](https://github.com/Nutlope/hallmark/blob/main/skills/hallmark/SKILL.md) for the source principles.

## Product position

Iroha is a private personal data cockpit. It turns raw imported evidence into a quiet, expressive field record of movement, recovery, daily patterns, and things collected over time.

The name is part of the product story:

- `iro`: signal, sound, tone, and the changing quality of a day.
- `hana`: flower, beauty, and something that unfolds through attention.

The interface should therefore feel observant and composed, not clinical, competitive, or noisy.

### Audience and use case

- Primary user: the owner of a self-hosted private instance.
- Primary use: quickly understand today, inspect a specific event, and notice patterns across time.
- Secondary use: select a privacy-safe annual view to share.
- Trust requirement: every visible number must be traceable to imported or explicitly derived data.

## Design principles

1. **Signal before volume.** Show the few facts that orient the user before exposing the full record.
2. **Composition follows meaning.** Today, sleep, activities, media, and long-term patterns use different page structures because they answer different questions.
3. **Quiet confidence.** Use expressive color, type, and shape as emphasis; never turn every metric into an alert.
4. **Evidence stays visible.** Show source, freshness, coverage, or derived status where it affects interpretation.
5. **Private by default.** Public pages are deliberate projections, not a restyled copy of private pages.
6. **Real data, honest copy.** Do not invent goals, trends, insights, or activity when the source does not provide them.
7. **Accessible beauty.** Contrast, focus, keyboard use, reduced motion, and text alternatives are part of the visual design.

## Page archetypes

| Archetype          | Route(s)       | User question                       | Structural rule                                                                              |
| ------------------ | -------------- | ----------------------------------- | -------------------------------------------------------------------------------------------- |
| Field Console      | `/`            | What is the shape of this day?      | One daily signal, a small set of priorities, then contextual moments.                        |
| Observatory        | `/overview`    | What has accumulated over time?     | Overview wall with totals, streaks, routes, and domains; do not duplicate Today’s narrative. |
| Pattern Atlas      | `/patterns`    | What rhythms or changes can I see?  | Calendar/heatmap and trend bands lead; detailed day inspection follows.                      |
| Movement Archive   | `/motion`      | What have I done?                   | Chronological record with visual filters and compact route/context previews.                 |
| Performance Report | `/motion/:id`  | What happened in this activity?     | Route and headline facts first; charts, zones, splits, and metadata form the report.         |
| Night Report       | `/night`       | How did the night unfold?           | Circadian timeline and architecture lead; history is selectable but subordinate.             |
| Personal Library   | `/library`     | What am I collecting or continuing? | Shelves and progress lead; aggregates support the collection instead of dominating it.       |
| Library Entry      | `/library/:id` | What is the history of this item?   | Cover/title context, progress, event timeline, people, and related items.                    |
| Design Archive     | `/design`      | Which directions have we explored?  | Review-only reference; never becomes a hidden production dependency.                         |

## Visual language

### Typography

- Use a restrained sans-serif for interface text and metrics.
- Use a distinctive display face only for page-level editorial moments, not every heading.
- Do not use italic display headings as the default identity device.
- Keep numeric metrics optically prominent but subordinate to their label and unit.
- Use sentence case for controls and headings; reserve all caps for short metadata labels.

The initial implementation may use the existing system stack. Introducing a new font requires a measured loading, fallback, licensing, and performance decision; it is not a prerequisite for the
redesign.

### Semantic color roles

The implementation must expose semantic variables rather than make route components choose raw colors.

| Role                       | Meaning                            | Examples                                     |
| -------------------------- | ---------------------------------- | -------------------------------------------- |
| `--color-canvas`           | Page background                    | Body and full-bleed areas                    |
| `--color-surface`          | Primary readable surface           | Cards, panels, dialogs                       |
| `--color-surface-raised`   | Elevated surface                   | Popovers, selected panels                    |
| `--color-border`           | Structural separation              | Dividers, input borders                      |
| `--color-ink`              | Primary text                       | Headings, metric values                      |
| `--color-ink-muted`        | Secondary text                     | Units, supporting copy                       |
| `--color-signal`           | Iroha identity/focus               | Links, active controls, key emphasis         |
| `--color-signal-secondary` | Secondary identity accent          | Editorial highlights and supporting emphasis |
| `--color-positive`         | Confirmed good/complete state      | Goal completion, success                     |
| `--color-warning`          | Needs attention or incomplete data | Partial coverage, caution                    |
| `--color-danger`           | Failure or destructive state       | Request failure, invalid action              |
| `--color-focus`            | Keyboard focus indicator           | Focus ring only; never rely on color alone   |

Domain palettes must be separate semantic roles:

- sport colors: stable by sport type and readable in both themes;
- sleep-stage colors: stable by stage and distinguishable without hue alone;
- chart series colors: ordered by meaning, not by arbitrary route-local choice;
- map route and marker colors: theme-aware and distinguishable from the map;
- media status colors: status plus text, never color alone.

### Surfaces and shape

- Prefer a small number of strong surfaces over many decorative cards.
- Use the existing tile language as a base, but introduce larger editorial surfaces, rails, timelines, shelves, and report sections where the archetype needs them.
- Keep radii and shadows tokenized; avoid a separate radius language per route.
- Decoration must support hierarchy or brand recognition. Remove decoration that competes with a metric or chart.

### Spacing

Use a compact base scale and compose larger gaps from it. The exact values can remain implementation-defined until the token task, but the contract requires:

- one page gutter scale;
- one section gap scale;
- one dense data gap scale;
- consistent control heights;
- no route-local one-off spacing values without a documented reason.

## Interaction and state contract

Interactive elements must have explicit designs for:

- default;
- hover or pointer indication;
- keyboard focus;
- pressed/selected;
- disabled;
- loading;
- error/retry;
- empty/no-data;
- partial or stale data where applicable.

Controls must preserve the existing API behavior. Visual redesign is not permission to change pagination, query parameters, IDs, dates, or public/private projection boundaries.

### Day navigation

Today remains anchored to one selected UTC day. The control must provide:

- previous and next day actions;
- disabled next action at the newest available boundary;
- a calendar selection surface;
- an explicit return-to-today action;
- visible selected-day and unavailable-day states;
- keyboard navigation without stealing focus from text inputs;
- a clear no-data result that is distinct from loading and request failure.

### Data status language

Use direct language:

- “Loading …” while a request is pending;
- “Could not load …” with a retry path when a request fails;
- “No … recorded” when the source has no matching data;
- “Partial data” when a section loaded but a dependent stream is absent;
- “Imported from …” or an equivalent provenance label when source context is useful;
- “Updated …” only when the timestamp is real and available.

Do not call a metric an “insight”, “readiness score”, or “trend” unless the calculation and interpretation are defined in source code or the design document.

## Responsive and accessibility contract

Review every production route at 320, 375, 414, and 768 CSS pixels, in both themes. The following are release requirements:

- no accidental page-level horizontal scrolling;
- readable controls and labels at 320px;
- tables and dense charts have an intentional overflow treatment or a compact alternative;
- keyboard focus is visible on every actionable control;
- dialogs trap or return focus correctly;
- tabs expose selected state and usable keyboard behavior;
- charts and maps provide a meaningful text label or summary;
- color is never the sole carrier of status or category;
- `prefers-reduced-motion` disables nonessential transitions and decoration;
- images have meaningful alternative text or are explicitly decorative.

## Motion stance

Motion should explain state changes, not decorate every surface.

- Use short transitions for selection, expansion, and navigation feedback.
- Avoid perpetual animation in production pages.
- Keep decorative orbit/glow effects limited to the Today identity surface or the design archive.
- Respect `prefers-reduced-motion` globally.
- Never use motion to communicate information that has no static equivalent.

**Amendment (2026-08-28): per-language ambient backgrounds.** Five of the six registered design languages (all but Grapher, which stays undecorated) mount a low-opacity, near-imperceptible WebGL
scene (`packages/iroha-shared/src/theme-ui/ambient/`) at the root layout, behind every route rather than limited to Today/design-archive as the rule above states literally. This is a deliberate,
scoped exception, not a lapse: every data tile paints an opaque `--tile-surface`, so the scene is only ever visible through chrome and empty space and never behind real data; `prefers-reduced-motion`
is a state contract (exactly one rendered frame, `requestAnimationFrame` never scheduled), not a slower loop; and each language's scene is an explicit opt-in entry in `ambient/factories.ts`, not a
default. Approved for this personal project as a deliberate "try the fancy stuff" call rather than a response to a specific user problem. Any future perpetual-motion addition outside Today/design-
archive should get the same explicit sign-off and the same amendment treatment here, not a silent exception.

## Data and privacy boundaries

Private surfaces consume `/api/v1`. There is no in-app public surface: the sanitized public projection is built by a standalone export (`apps/iroha-server/pkg/publicexport`) and rendered by a separate
static site, not fetched live from this app.

The frontend must preserve:

- real API DTO names and nullability;
- pagination and cursor behavior;
- stable activity/media/sleep IDs;
- UTC day interpretation;
- truthful empty and partial states.

Any new derived presentation metric needs:

1. a named calculation;
2. a source reference or test;
3. a clear unit and null behavior;
4. a design decision describing how it should be interpreted.

## Quality review

Every major route is scored before acceptance from 1 to 5 on:

| Dimension   | Review question                                                                 |
| ----------- | ------------------------------------------------------------------------------- |
| Philosophy  | Does the page express Iroha’s quiet, observant identity?                        |
| Hierarchy   | Can the user identify the primary answer within seconds?                        |
| Execution   | Are spacing, type, states, and responsive behavior coherent?                    |
| Specificity | Does this feel designed for Iroha’s data rather than generic dashboard content? |
| Restraint   | Do decoration and metrics support rather than compete with meaning?             |
| Variety     | Is this page structurally appropriate and distinct from neighboring routes?     |

A score below 3 on any dimension requires revision before the route is called accepted. The review must include both themes, the target mobile widths, and loading/empty/error states.

## Implementation order

1. Consolidate semantic tokens and the application shell.
2. Rebuild Today as the reference implementation.
3. Apply distinct archetypes to Activities and Sleep.
4. Apply atlas/observatory structures to Daily and Dashboard.
5. Apply library/report structures to Media and Share.
6. Run the responsive, accessibility, theme, state, and data-boundary gate.
7. Archive the accepted direction; only then decide whether any design-lab route or experiment should be removed.

## Contract acceptance

This contract is ready for implementation when the team agrees that:

- the Field Console direction is the primary product direction;
- page archetypes are intentionally different but share semantic tokens;
- existing route/API/data boundaries are preserved;
- light/dark, responsive, accessibility, motion, and state behavior are release requirements;
- `/design` and `docs/design-archive/` remain until explicit cleanup approval.
