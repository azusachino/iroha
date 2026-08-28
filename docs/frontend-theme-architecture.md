# Frontend theme architecture

Status: implementation contract for registered design languages and adopted compositions

## Goal

An Iroha design language is a complete visual and interaction language. Switching it may change the shell, navigation, page composition, typography, chart treatment, surface vocabulary, and motion. An
adopted design composition is a real layout system built on the same canonical view model. Neither may change API contracts, data meaning, privacy boundaries, or null/error behavior.

Light and dark are contrast modes. A design language is a product-level art direction. They are independent axes:

```text
registered language: Field Journal / Grapher / Atlas / Phenology / Sound Map / Archive
contrast mode:   light / dark
```

## Source of truth

The shared theme package is the only place that defines supported registered languages and adopted design compositions. The current web tree is migration debt, not the target home:

```text
packages/iroha-shared/src/theme-ui/
├── registry.ts          shared component registration
├── context.svelte.ts    shared theme runtime contract
├── field-journal/
├── grapher/
├── atlas/
├── phenology/
├── sound-map/
└── archive/

packages/iroha-shared/src/
├── themes.ts            registered language identities, lenses, and route contracts
├── themes.css           semantic language tokens
├── design-compositions.ts  adopted composition identities and view contracts
├── PeriodSelector.svelte
└── SelectControl.svelte

packages/iroha-shared/src/theme-ui/compositions/
└── ...                   package-owned adopted layout implementations
```

Each theme directory owns curated visual components, not a second copy of the application data layer:

```text
theme/
├── Shell.svelte         app frame, navigation, brand, footer
├── Today.svelte         Today composition
├── Daily.svelte         pattern composition
├── Activities.svelte    archive/list composition
├── Sleep.svelte         night composition
├── Media.svelte         library composition
└── Reports.svelte       monthly analytical composition
```

Themes may compose shared data visualizations and primitives, but route files must not contain a visual conditional over registered languages or adopted compositions. A route loads a stable view model
and delegates rendering to the selected shared component.

`Shell.svelte`'s `ShellThemeProps` (`view-contracts/shell-view.ts`) includes `brand`, `nav`, and `actions` snippets alongside `children`. The web host still owns their contents — the brand link, the
primary-nav disclosure menus, the command-palette trigger, the design-language picker, and the theme toggle are host interaction/state and must not move into `packages/`. What a theme owns is
arrangement and decoration: where the header sits, how dense it is, what visual language wraps those snippets. Each of the six languages renders its own `<header>` and restyles the host's
`.main-nav`/`.appbar-actions` markup with scoped `:global()` selectors — a dashed map-legend for Atlas, flat axis-tick tabs for Grapher, a dotted/wavy-underline masthead for Field Journal, a phase-dot
pill for Phenology, a rack-style "mixing console" for Sound Map, and a card-catalog drawer strip for Archive — rather than duplicating the interactive elements. This keeps the audited focus order,
tap-target sizes, and accessible names in one place while still letting the persistent chrome look meaningfully different per theme, not just recolored. `ThemeFrame.svelte`'s own fallback branch (used
only if a theme is missing a `shell` component, which the registry tests forbid) renders the same three snippets in a plain, unstyled `<header class="appbar">` since it has no theme identity to draw
on.

`themes.css` also owns a small named `--motion-*` token vocabulary (`micro`, `quick-state`, `data-update`, `language-switch`) shared across every language, pairing a duration with an easing so a
consumer writes `transition: opacity var(--motion-quick-state)` rather than a local literal. `make motion-tokens-check` fails if a reference has no matching definition.

## Layer boundaries

### Data and view models

`src/lib/api.ts` remains the API boundary. Theme-independent adapters may turn DTOs into display view models, but they must preserve IDs, nullability, units, pagination, UTC dates, and public/private
projection boundaries.

### Shared primitives

`packages/iroha-shared/src/theme-ui/components/` contains behaviorally reusable theme primitives:

- charts, maps, gauges, timelines, shelves, tables;
- loading, empty, error, partial, and stale states;
- accessible controls, date navigation, dialogs, and filters.

A primitive owns behavior and accessibility. A theme owns composition and visual treatment. A route owns data loading and URL/API behavior.

### Curated theme components

Theme components own:

- page layout and hierarchy;
- navigation emphasis and shell structure;
- type scale and editorial voice;
- selection of appropriate visualization primitives;
- decoration, texture, motion, and density;
- responsive composition for that language.

They must not invent metrics or bypass shared state contracts.

### Report composition contract

Reports are a deliberate test of the boundary. The route owns the selected month, the server-generated monthly envelope, the twelve-month series, and URL/error/loading state. The shared primitives own
chart/table parity, canonical units, coverage, provenance, and export behavior.

Each registered language owns its report composition in its own `packages/iroha-shared/src/theme-ui/<language>/Reports.svelte`. Those files choose domain order, chart orientation, hierarchy,
typography, density, and the placement of the evidence list. There is no universal five-domain report renderer. A new shared primitive may provide truthful behavior or accessibility, but it must not
silently decide the visual hierarchy for every language.

## Registry contract

The shared registry must provide typed metadata and actual Svelte components for every registered language and adopted composition. Identity and page-lens metadata lives in the shared manifest; the
web app supplies only data and navigation adapters:

```ts
type ThemeDefinition = {
  identity: ThemeIdentity;
  implementation: ThemeImplementationStatus;
  primitives: ThemePrimitives;
  components: Record<ThemeRoute, Component>;
};
```

The language registry is exhaustive for every production route. The composition registry is open-ended but equally concrete: adding a composition requires a shared implementation, a canonical
view-model contract, a runnable design-workshop specimen, and an explicit adoption status. A URL tab or CSS-only variant is not an implementation.

## Community-standard implementation rules

1. Prefer composition over inheritance and copy/paste.
2. Keep components small enough to test and review independently.
3. Keep public types explicit; do not use `any` for theme contracts.
4. Keep API loading in route/page modules and visual rendering in components.
5. Keep design tokens semantic; do not route raw hex values into data logic.
6. Keep accessibility behavior in shared primitives unless the theme has a documented interaction difference.
7. Keep each theme’s assets and styles colocated with the shared theme component under `packages/`.
8. Use the design workshop as a real implementation and review surface. Its layout compositions use canonical fixtures and must remain runnable; they are not static concept boards.
9. Prefer one complete vertical slice over superficial variants, while keeping every adopted composition real.
10. Keep the existing route tree and API paths stable while migrating.

## Definition of a complete theme

A theme is not complete when its colors change. It is complete when a viewer can identify it with the language selector hidden because these differ from another theme:

- shell and navigation;
- page-level layout;
- primary visualization choice;
- typography and density;
- interaction and selection treatment;
- loading/empty/error/partial states;
- responsive behavior at 320, 414, 768, and desktop widths.

The same imported day must be rendered through every accepted theme in the design lab before a theme is promoted to production.

## Design workshop compositions

The `/design` route contains two real, complementary implementation axes:

- registered design languages, selected through the theme registry;
- package-owned layout compositions selected through the design-composition registry. The current set is Editorial, Command center, Chronicle, Cover page, Personal OS, Field journal, and Quiet; this
  list is extensible and is not a ceiling on future Iroha design systems.

The layout compositions are not throwaway mockups. They are executable Svelte compositions bound to the same canonical Today view model, with stable URL selection, accessible controls, responsive
styles, and deterministic fallback data. They are first-class shared implementations alongside the registered languages, and the acceptance matrix must exercise all of them rather than silently
reducing them to screenshots. The public-site fixture workbench is the cross-app consumer for these compositions; its 84-case mobile smoke covers every registered language, adopted composition, and
light/dark mode.

## Migration order

1. **Complete:** define the shared view models and move language metadata and the production registry into the shared theme package.
2. **Complete:** promote the workshop compositions into the shared composition registry with real implementations.
3. **Complete:** build shared route-family slices with web and public-site adapters consuming the package-owned assets.
4. **Complete for v0.4:** review the same fixture/API data in the design lab at the required breakpoints and contrast modes.
5. **Complete for v0.4:** migrate Daily, Activities, Sleep, Media, Expenses, Reports, and Metrics for every registered language; adopted compositions are exercised by the shared design workbench.
6. **Complete for v0.4:** remove obsolete route-local visual conditionals after the visual, accessibility, and data-boundary gates passed.
