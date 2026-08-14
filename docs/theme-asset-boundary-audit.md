# Theme asset boundary audit

- Date: 2026-08-14
- Scope: `apps/iroha-web`, `apps/iroha-public-site`, and `packages/iroha-shared`
- Reference: `b1b6cd2`, `61ea5fe`, `793749d`, plus the current working tree

## Verdict

The registered production route compositions and their canonical view contracts now live in `packages/iroha-shared`; the web app retains only the host registry/runtime adapters. The design-workshop
compositions are real package-owned implementations rather than web-only variants. Cross-app reuse is still incomplete because the public site does not yet consume the shared composition renderer and
the host registry is not yet shared.

The new hard rule is in [`AGENTS.md`](../AGENTS.md). The shared-package README was updated in the same change because it described the opposite architecture.

## Inventory

| Asset                                                                     | Current location                                                                                                                                                                                   | Finding                                                                                                                                                |
| ------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Current production theme IDs, identities, route names, and registry types | [`packages/iroha-shared/src/themes.ts`](../packages/iroha-shared/src/themes.ts)                                                                                                                    | Correctly shared. This is the contract, not the implementation; adopted future designs must extend this registry rather than create a second catalog.  |
| Semantic theme tokens                                                     | [`packages/iroha-shared/src/themes.css`](../packages/iroha-shared/src/themes.css)                                                                                                                  | Correctly shared.                                                                                                                                      |
| Shared low-dependency Svelte primitives                                   | [`packages/iroha-shared/src`](../packages/iroha-shared/src)                                                                                                                                        | Correct home; the set is incomplete.                                                                                                                   |
| Theme registry and component map                                          | [`packages/iroha-shared/src/theme-ui/registry.ts`](../packages/iroha-shared/src/theme-ui/registry.ts)                                                                                               | Corrected. The web `registry.ts` is only a compatibility re-export; the component map and route definitions are package-owned.                                                           |
| Theme context and route renderer                                          | [`apps/iroha-web/src/lib/themes/context.svelte.ts`](../apps/iroha-web/src/lib/themes/context.svelte.ts), [`ThemeRouteRenderer.svelte`](../apps/iroha-web/src/lib/themes/ThemeRouteRenderer.svelte) | Host runtime adapter. They still depend on web persistence/context and should be extracted only after the shared component registry is defined.        |
| Theme provider                                                            | [`apps/iroha-web/src/lib/themes/ThemeProvider.svelte`](../apps/iroha-web/src/lib/themes/ThemeProvider.svelte)                                                                                      | Mixed responsibility. Theme context belongs in the shared package; local-storage persistence is a web adapter.                                         |
| Theme frame                                                               | [`apps/iroha-web/src/lib/themes/ThemeFrame.svelte`](../apps/iroha-web/src/lib/themes/ThemeFrame.svelte)                                                                                            | Host adapter. Shell compositions are package-owned; `APP_VERSION`, private fallback footer, and theme context remain web concerns.                     |
| Route compositions                                                        | [`packages/iroha-shared/src/theme-ui/`](../packages/iroha-shared/src/theme-ui/)                                                                                                                    | Corrected. All 11 registered route families, across all six production languages, are package-owned; the web tree contains no theme composition files. |
| Cross-theme charts and visual components                                  | [`apps/iroha-web/src/lib/components/`](../apps/iroha-web/src/lib/components/) and [`packages/iroha-shared/src/theme-ui/components/`](../packages/iroha-shared/src/theme-ui/components/)            | Mixed by design. Charts and visual primitives are shared; MapLibre route rendering, app navigation, and fetch-state adapters remain web-local.         |
| Public-site theme implementation                                          | [`apps/iroha-public-site/src/`](../apps/iroha-public-site/src/)                                                                                                                                    | Incomplete. It currently imports only the shared `StatTile` wrapper and does not yet render the package-owned production or adopted compositions.      |

## Detailed findings

### Critical: the current production compositions are web-owned

The web route folders no longer contain production theme compositions. Shell, Today, Daily, Dashboard, Media, Expenses, Activities, Activity Detail, Reports, Sleep, Media Detail, and Metrics now live
under `packages/iroha-shared/src/theme-ui/`, with typed data/callback contracts beside them. The web-only registry still imports those package assets and constructs the `ThemeDefinition<Component>`
instances used at runtime; that registry/runtime adapter is the remaining app-owned boundary, not the visual compositions themselves.

This is not fixed by moving only a card or a chart. The following are all theme-owned compositions and must move together behind a shared view contract:

- shell and today (now shared)
- dashboard (now shared; host-supplied map snippet remains app-owned)
- activities (the activity-list and Activity Detail compositions are now shared)
- sleep (the Sleep/Night composition and stage/timeline primitives are now shared)
- media detail (the Media list and detail compositions are now shared)
- expenses
- metrics (now shared)

### Closed for the production route-composition boundary

The production route compositions no longer import `$lib/api`, route modules, or web-local visual components. Their shared contracts accept typed data, callbacks, and host snippets; API fetching,
loading/error state, route navigation, and MapLibre remain in the application.

The most coupled areas are:

- expenses: moved. Canonical expense DTOs, view contracts, CSV serialization, ledger, and all registered production-language compositions now live in the shared package; the route retains API, period,
  filter, and selection state;
- reports: moved. Canonical report DTOs, section helpers, comparison, report cards, coverage, charts, and all registered production-language compositions now live in the shared package; the route
  retains only HTTP and period state;
- sleep detail: canonical API types, maps, charts, and source labels are now shared; the route retains API and selection state;
- media: the Media list and detail contracts, formatting, and compositions are now shared;
- daily: canonical rows/aggregates, ring and small-multiple primitives, and all six Daily compositions are now shared; the patterns route retains API, period, and drill state;
- shell: six shell compositions are now shared; version display, theme persistence, navigation, and command-palette behavior remain web adapters;
- today: six Today compositions and media-event semantics are now shared; date loading, day navigation, and media/activity callbacks remain web adapters.

### High: shared visual primitives are only partially shared

The package owns useful primitives such as `MetricPanel`, `MetricTable`, `StatTile`, period controls, source badges, `BarChart`, `LapChart`, activity trend charts, sleep charts, sleep-stage
labels/colors, report comparison/cards/coverage/facts, receipts, media charts, rings, small multiples, retry notices, and pure formatters. The web app still owns map infrastructure, route loading,
navigation, private chrome, and app-only command-palette behavior.

These should be audited one by one during migration. A component is a shared asset when its visual grammar is part of Iroha's data language and it can be fed by a typed view model. A component remains
web-local when it performs API fetching, private navigation, authentication, deployment/version display, or app-only command-palette behavior.

### Closed for the package boundary: the design workshop is now a shared implementation

[`packages/iroha-shared/src/design-compositions.ts`](../packages/iroha-shared/src/design-compositions.ts) owns the composition identities and canonical Today view model. The seven current
implementations live under [`packages/iroha-shared/src/theme-ui/compositions/`](../packages/iroha-shared/src/theme-ui/compositions/), and `/design` renders them through the shared
`DesignCompositionRenderer`. They remain a workshop/composition registry rather than a fixed theme count: new design systems can be added with another shared implementation. The remaining gaps are the
shared production component registry/runtime and a public-site fixture consumer.

### Medium: the theme runtime is still split across packages

`packages/iroha-shared/src/themes.ts` owns the registered language IDs, identities, route names, registry types, and `packages/iroha-shared/src/theme-ui/registry.ts` owns the component map. The web
`registry.ts` is only a compatibility re-export. The remaining split is the host runtime: context, persistence, route rendering, and the provider still live in `iroha-web`.

The web app should provide only data, persistence, and navigation adapters, not a second registry.

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
4. **Complete:** move all registered route families across the six production languages, including shared primitives, registry imports, and design-page specimens. Dashboard map rendering remains an
   explicit host-provided snippet because it depends on MapLibre and API-owned loading state.
5. **In progress:** move the theme context/renderer/provider contracts into the shared package while keeping local persistence and private chrome in the web host. Preserve URL, loading, persistence, and navigation behavior.
6. Add public-site fixture data and render all registered themes and adopted compositions in its design workbench. This is the cross-app proof that the boundary is real.
7. Add a CI/check target that fails if new files appear under `apps/*/src/lib/themes` or if `packages/iroha-shared` imports app aliases.

## Completion criteria

- No registered language or adopted design composition, and no theme-aware primitive, is source-of-truth code under either app.
- `iroha-web` and `iroha-public-site` render the same shared compositions from their respective data adapters.
- Shared theme code has no `$lib` or app-source imports.
- Theme IDs, route names, identities, registry entries, and tokens have one canonical definition.
- The design page exercises the same shared components used by production routes; it is not a parallel mock.
- Web and public-site checks pass, and every theme is browser-verified with representative fixture/API data.
