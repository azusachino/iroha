# Frontend theme architecture

Status: implementation contract for curated Iroha design languages

## Goal

An Iroha theme is a complete visual and interaction language. Switching it may change the shell, navigation, page composition, typography, chart treatment, surface vocabulary, and motion. It must not
change API contracts, data meaning, privacy boundaries, or null/error behavior.

Light and dark are contrast modes. A design language is a product-level art direction. They are independent axes:

```text
design language: Field Journal / Grapher / Atlas / Phenology / Sound Map / Archive
contrast mode:   light / dark
```

## Source of truth

The theme registry is the only place that defines the supported languages:

```text
apps/iroha-web/src/lib/themes/
├── registry.ts          metadata and component contracts
├── types.ts             shared theme/component types
├── tokens.css           semantic token defaults and language overrides
├── field-journal/
├── grapher/
├── atlas/
├── phenology/
├── sound-map/
└── archive/
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

Themes may compose shared data visualizations and primitives, but route files must not contain six-way visual conditionals. A route loads a stable view model and delegates rendering to the selected
theme component.

## Layer boundaries

### Data and view models

`src/lib/api.ts` remains the API boundary. Theme-independent adapters may turn DTOs into display view models, but they must preserve IDs, nullability, units, pagination, UTC dates, and public/private
projection boundaries.

### Shared primitives

`src/lib/components/` contains behaviorally reusable primitives:

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

## Registry contract

The registry must provide typed metadata for every language:

```ts
type ThemeDefinition = {
  id: DesignLanguage;
  label: string;
  description: string;
  shell: Component;
  routes: {
    today: Component;
    daily: Component;
    activities: Component;
    sleep: Component;
    media: Component;
    share: Component;
  };
  tokens: string;
};
```

The registry is exhaustive. Adding a language without all required production route components is a type/check failure, not a partially working option.

## Community-standard implementation rules

1. Prefer composition over inheritance and copy/paste.
2. Keep components small enough to test and review independently.
3. Keep public types explicit; do not use `any` for theme contracts.
4. Keep API loading in route/page modules and visual rendering in components.
5. Keep design tokens semantic; do not route raw hex values into data logic.
6. Keep accessibility behavior in shared primitives unless the theme has a documented interaction difference.
7. Keep each theme’s assets and styles colocated with the theme component.
8. Use the design workshop as a real implementation and review surface. Its
   layout compositions use canonical fixtures and must remain runnable; they
   are not static concept boards.
9. Prefer one complete vertical slice over six superficial variants.
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
- the seven implemented layout compositions: Editorial, Command center,
  Chronicle, Cover page, Personal OS, Field journal, and Quiet.

The layout compositions are not throwaway mockups. They are executable Svelte
compositions bound to the same canonical Today view model, with stable URL
selection, accessible controls, responsive styles, and deterministic fallback
data. They are first-class workshop implementations alongside the registered
themes, and the acceptance matrix must exercise all of them rather than
silently reducing them to screenshots.

## Migration order

1. Move the current language metadata into the typed registry.
2. Create the shared theme host and preserve the current Field Journal route as the reference implementation.
3. Build a complete Grapher vertical slice: shell, Today, Share, and chart/map/ table view primitives with provenance treatment.
4. Review the same data in the design lab at required breakpoints.
5. Migrate Daily, Activities, Sleep, and Media for the accepted languages.
6. Add Atlas, Phenology, Sound Map, and Archive only after their route contracts are complete.
7. Remove obsolete route-local visual conditionals only after the replacement has passed the visual, accessibility, and data-boundary gates.
