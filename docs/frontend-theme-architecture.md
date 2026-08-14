# Frontend theme architecture

Status: implementation contract for curated Iroha design languages

## Goal

An Iroha theme is a complete visual and interaction language. Switching it may change the shell, navigation, page composition, typography, chart treatment, surface vocabulary, and motion. It must not
change API contracts, data meaning, privacy boundaries, or null/error behavior.

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
├── themes.ts            identities, lenses, route and design contracts
├── themes.css           semantic language tokens
├── PeriodSelector.svelte
└── SelectControl.svelte
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
└── Share.svelte         public/editorial composition
```

Themes may compose shared data visualizations and primitives, but route files must not contain a visual conditional over registered languages or adopted compositions. A route loads a stable view model
and delegates rendering to the selected shared component.

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

Each registered language owns its report composition in its own `themes/<language>/Reports.svelte`. Those files choose domain order, chart orientation, hierarchy, typography, density, and the
placement of the evidence list. There is no universal five-domain report renderer. A new shared primitive may provide truthful behavior or accessibility, but it must not silently decide the visual
hierarchy for every language.

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

The registry is exhaustive for every adopted entry. Adding a language or design composition without all required production route components is a type/check failure, not a partially working option or
a design-page-only variant.

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

- the six registered design languages, selected through the theme registry;
- the seven implemented layout compositions: Editorial, Command center, Chronicle, Cover page, Personal OS, Field journal, and Quiet.

The layout compositions are not throwaway mockups. They are executable Svelte compositions bound to the same canonical Today view model, with stable URL selection, accessible controls, responsive
styles, and deterministic fallback data. They are first-class shared implementations alongside the registered themes, and the acceptance matrix must exercise all of them rather than silently reducing
them to screenshots. A design-workshop entry is not adopted until its implementation and registry identity leave the app directory.

## Migration order

1. Define the shared view models and move the current language metadata and registry into the shared theme package.
2. Promote Editorial, Command center, Chronicle, Cover page, Personal OS, Field journal, and Quiet from the web design route into shared registry entries with real implementations.
3. Build one complete shared vertical slice—Grapher media, for example—with the web and public-site adapters consuming it.
4. Review the same data in the design lab at required breakpoints.
5. Migrate Daily, Activities, Sleep, Media, Expenses, Reports, and Metrics for every accepted language and composition.
6. Remove obsolete route-local visual conditionals only after the replacement has passed visual, accessibility, and data-boundary gates.
