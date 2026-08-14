# Theme asset boundary audit

- Date: 2026-08-14
- Scope: `apps/iroha-web`, `apps/iroha-public-site`, and `packages/iroha-shared`
- Reference: `b1b6cd2`, `61ea5fe`, `793749d`, plus the current working tree

## Verdict

The boundary is partially corrected in the current working tree. The registered production route compositions and runtime registry remain web-owned migration debt, but the design-workshop compositions
and their canonical view model have now moved into `packages/iroha-shared`. They are real package-owned implementations rather than web-only variants. Cross-app reuse is still incomplete because the
public site does not yet consume the shared composition renderer or the remaining production route families.

The new hard rule is in [`AGENTS.md`](../AGENTS.md). The shared-package README was updated in the same change because it described the opposite architecture.

## Inventory

| Asset                                                                     | Current location                                                                                                                                                                                   | Finding                                                                                                                                                                                  |
| ------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Current production theme IDs, identities, route names, and registry types | [`packages/iroha-shared/src/themes.ts`](../packages/iroha-shared/src/themes.ts)                                                                                                                    | Correctly shared. This is the contract, not the implementation; adopted future designs must extend this registry rather than create a second catalog.                                    |
| Semantic theme tokens                                                     | [`packages/iroha-shared/src/themes.css`](../packages/iroha-shared/src/themes.css)                                                                                                                  | Correctly shared.                                                                                                                                                                        |
| Shared low-dependency Svelte primitives                                   | [`packages/iroha-shared/src`](../packages/iroha-shared/src)                                                                                                                                        | Correct home; the set is incomplete.                                                                                                                                                     |
| Theme registry and component map                                          | [`apps/iroha-web/src/lib/themes/registry.ts`](../apps/iroha-web/src/lib/themes/registry.ts)                                                                                                        | Partially wrong. The web app still owns the production route map; adopted workshop-composition metadata and implementations are now in `packages/iroha-shared`.                          |
| Theme context and route renderer                                          | [`apps/iroha-web/src/lib/themes/context.svelte.ts`](../apps/iroha-web/src/lib/themes/context.svelte.ts), [`ThemeRouteRenderer.svelte`](../apps/iroha-web/src/lib/themes/ThemeRouteRenderer.svelte) | Wrong as shared assets; they are generic theme-runtime infrastructure.                                                                                                                   |
| Theme provider                                                            | [`apps/iroha-web/src/lib/themes/ThemeProvider.svelte`](../apps/iroha-web/src/lib/themes/ThemeProvider.svelte)                                                                                      | Mixed responsibility. Theme context belongs in the shared package; local-storage persistence is a web adapter.                                                                           |
| Theme frame                                                               | [`apps/iroha-web/src/lib/themes/ThemeFrame.svelte`](../apps/iroha-web/src/lib/themes/ThemeFrame.svelte)                                                                                            | Mixed responsibility. The shell host belongs with the theme package; `APP_VERSION` and private fallback footer do not.                                                                   |
| Route compositions                                                        | [`apps/iroha-web/src/lib/themes/`](../apps/iroha-web/src/lib/themes/)                                                                                                                              | Partially wrong. 30 route components remain app-local (5 per registered language); Media, Expenses, Activities, Activity Detail, Reports, Sleep, and Media Detail are now package-owned. |
| Cross-theme charts and visual components                                  | [`apps/iroha-web/src/lib/components/`](../apps/iroha-web/src/lib/components/) and [`packages/iroha-shared/src/theme-ui/components/`](../packages/iroha-shared/src/theme-ui/components/)            | Mixed. BarChart, report primitives, receipts, media charts, activity trend charts, sleep charts, and LapChart are shared; maps and some detail surfaces remain web-local.                |
| Public-site theme implementation                                          | [`apps/iroha-public-site/src/`](../apps/iroha-public-site/src/)                                                                                                                                    | Incomplete. It currently imports only the shared `StatTile` wrapper and does not yet render the package-owned production or adopted compositions.                                        |

## Detailed findings

### Critical: the current production compositions are web-owned

The route folders now contain 30 Svelte files, 5 per registered production language. Media, Expenses, Activities, Activity Detail, Reports, Sleep, and Media Detail have moved to
`packages/iroha-shared/src/theme-ui/`, but the web-only registry still imports the remaining route map and constructs the only `ThemeDefinition<Component>` instances used at runtime. That remaining
map is the core asset still living in the wrong application.

This is not fixed by moving only a card or a chart. The following are all theme-owned compositions and must move together behind a shared view contract:

- shell and today
- daily and dashboard
- activities (the activity-list and Activity Detail compositions are now shared)
- sleep (the Sleep/Night composition and stage/timeline primitives are now shared)
- media detail (the Media list and detail compositions are now shared)
- expenses
- metrics

### Critical: the route components are coupled to private-app internals

The remaining route files directly import `$lib/api`, `$lib/format`, `$lib/sport`, `$lib/hero-title`, and web-local components. A blind directory move would merely relocate broken imports. The
migration needs an explicit shared view-model boundary: typed data in, callbacks/snippets out, with API fetching and route navigation kept in the application.

The most coupled areas are:

- expenses: moved. Canonical expense DTOs, view contracts, CSV serialization, ledger, and all registered production-language compositions now live in the shared package; the route retains API, period,
  filter, and selection state;
- reports: moved. Canonical report DTOs, section helpers, comparison, report cards, coverage, charts, and all registered production-language compositions now live in the shared package; the route
  retains only HTTP and period state;
- sleep detail: canonical API types, maps, charts, and source labels are interwoven;
- media: the Media list and detail contracts, formatting, and compositions are now shared;
- shell: private version and theme persistence are mixed into presentation.

### High: shared visual primitives are only partially shared

The package owns useful primitives such as `MetricPanel`, `MetricTable`, `StatTile`, period controls, source badges, `BarChart`, `LapChart`, activity trend charts, report
comparison/cards/coverage/facts, receipts, media charts, sleep charts, sleep-stage labels/colors, and pure formatters. The web app still owns theme-aware assets such as `ActivityDetailChart`,
`RingGauge`, maps, media detail, and the remaining route surfaces.

These should be audited one by one during migration. A component is a shared asset when its visual grammar is part of Iroha's data language and it can be fed by a typed view model. A component remains
web-local when it performs API fetching, private navigation, authentication, deployment/version display, or app-only command-palette behavior.

### Closed for the package boundary: the design workshop is now a shared implementation

[`packages/iroha-shared/src/design-compositions.ts`](../packages/iroha-shared/src/design-compositions.ts) owns the composition identities and canonical Today view model. The seven current
implementations live under [`packages/iroha-shared/src/theme-ui/compositions/`](../packages/iroha-shared/src/theme-ui/compositions/), and `/design` renders them through the shared
`DesignCompositionRenderer`. They remain a workshop/composition registry rather than a fixed theme count: new design systems can be added with another shared implementation. The remaining gap is
production route coverage and a public-site fixture consumer.

### Medium: the registry contract is split across packages

`packages/iroha-shared/src/themes.ts` owns the registered language IDs, identities, route names, and registry types, while `apps/iroha-web/src/lib/themes/registry.ts` owns the remaining production
route map. The design-composition IDs and implementations are now shared under `packages/iroha-shared`. The registries are deliberately separate: a language is a production-wide visual language, while
a composition is an adopted layout system that can be promoted across route families over time.

The canonical registry must eventually include the implementation map in the shared theme package. The web app should provide only data and navigation adapters, not a second registry.

### Medium: public-site reuse is currently nominal

Both apps alias `@iroha/shared`, but the public site only consumes a shared `StatTile` wrapper. The adopted compositions are now reusable in principle, while current production route families and
theme-aware charts are not actually consumed by both apps yet.

## Target boundary

Keep the existing `@iroha/shared` package as the source-only package for now; split a dedicated `@iroha/theme-ui` package later only if the chart/map dependency weight makes that necessary. The
important boundary is `packages/`, not the package name.

The target shape is:

```text
packages/iroha-shared/src/
  themes.ts                 # IDs, identities, route contract
  themes.css                # semantic tokens
  design-compositions.ts    # adopted composition IDs and view contracts
  theme-ui/                 # registered themes and adopted compositions
    registry.ts
    context.svelte.ts
    ThemeRouteRenderer.svelte
    atlas/ ... grapher/ ... field-journal/ ...
    components/             # charts, cards, receipts, controls
    compositions/           # adopted layout implementations
  view-models/              # canonical frontend contracts

apps/iroha-web/src/
  lib/api.ts                # HTTP and response decoding
  lib/theme-adapter/        # API -> shared view model, persistence, navigation
  routes/                   # route state, loading, errors, URL/query behavior

apps/iroha-public-site/src/
  lib/fixture-adapter/      # static examples -> shared view model
  routes/                   # public shell and static page composition
```

The shared theme code may depend on Svelte and shared frontend libraries, but must not import `$lib/api`, route modules, server packages, or another app's source. It accepts typed view data, snippets,
and callbacks. The theme context must be driven by the shared registry; local storage or URL persistence is injected by the hosting app.

## Migration order

1. Define and test shared view models for the route families, starting with media, expenses, and reports where the current regressions are visible.
2. Move the theme-aware primitive layer and its theme-specific styling into `packages/iroha-shared/src/theme-ui/components/`.
3. **Complete:** promote the design-workshop compositions into the shared design registry; Editorial, Command center, Chronicle, Cover page, Personal OS, Field Journal, and Quiet are real package
   implementations rather than route-local demos.
4. **Complete:** move the Media, Expenses, Activities, Activity Detail, Reports, Sleep, and Media Detail route families across the registered languages, including shared primitives, registry entries,
   and design-page specimens.
5. **In progress:** move the remaining route families and the shared registry. Keep web route adapters thin while preserving URL, loading, and navigation behavior.
6. Add public-site fixture data and render all registered themes and adopted compositions in its design workbench. This is the cross-app proof that the boundary is real.
7. Add a CI/check target that fails if new files appear under `apps/*/src/lib/themes` or if `packages/iroha-shared` imports app aliases.

## Completion criteria

- No registered language or adopted design composition, and no theme-aware primitive, is source-of-truth code under either app.
- `iroha-web` and `iroha-public-site` render the same shared compositions from their respective data adapters.
- Shared theme code has no `$lib` or app-source imports.
- Theme IDs, route names, identities, registry entries, and tokens have one canonical definition.
- The design page exercises the same shared components used by production routes; it is not a parallel mock.
- Web and public-site checks pass, and every theme is browser-verified with representative fixture/API data.
