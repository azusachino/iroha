# Dependency and loading cleanup

Owner-approved scope: remove unused/unnecessary dependencies, incorporate coverage PR #91, and investigate heavy sparsely-used dependencies with packaging/loading UX priority. No deployment, release, Node/TypeScript major upgrade, quality-floor changes or live-data probes.

Baseline: main `6612293c6f70f3b6d76c4506406fde328eb1ece7`. Branch: `chore/dependency-cleanup-table`. Workstation tier 1; independent verification pending.

## Decisions and acceptance

- A1: Remove unused `three` from private/shared, `@types/three` from shared, and `uplot` from private. Regenerate locks without unrelated version updates. No source imports exist for those packages.
- A2: Owner explicitly chose removing TanStack rather than upgrading to stable 9.2.4. Preserve header labels, initial descending date order, first-row-type toggle defaults, Shift state, leading-key-only row ordering, year/sport filters, progressive rows and detail links. Correct stale header arrows and expose the actual leading sort with `aria-sort`.
- A3: Incorporate coverage 7.16.2. Owner explicitly approved this version before the seven-day age window elapsed; preserve general `P7D` lock metadata and leave no lasting package exemption. Coverage floors stay unchanged. `uv lock --check` must pass.
- A4: Defer the public map engine until a map mounts. No-route public landing must not request it. Delayed or failed library loading must leave charts, activity records and route summaries usable, with explicit loading/error feedback.
- A5: Package MapLibre's worker explicitly for both hosts. A canvas alone is insufficient: the public map's `aria-busy` clears only when its GeoJSON source reports loaded. Verify actual worker/source loading, deep links, back navigation and successful recovery after an explicit page reload.
- A6: Preserve sanitized public-only requests, existing map styling/attribution/controls, shared chart runtime deduplication and existing test assertions/thresholds. No replacement map/chart dependency is introduced.
- A7: Run owning validation, full browser suite, integration, production-focused browser checks, and fresh independent verification. Report actual bundle changes and limitations, not speculative transfer/latency gains.

## Findings

Read-only advisor `iroha-dep-advisor`, Sonnet 5.5 / medium, session `d992c052-1b6e-4772-a4d7-485c3a145f23`, independently traced source imports. Lead compared emitted artifacts and browser behavior.

| Dependency | Decision | Evidence / boundary |
| --- | --- | --- |
| Three.js / its types | Remove | No runtime/type imports in private/public/shared source; leftover declarations. Removal cleans installs, not emitted JS. |
| uPlot | Remove | No source imports; README's line-chart claim was obsolete. |
| TanStack Table | Replace with typed Svelte state | Public page already sorted and paged arrays manually; library only supplied header/toggle state. The retained baseline test exposed stale arrows after actual keyboard sorting. |
| ECharts | Retain | Shared bar/line/pie/fused charts, tooltip/legend/theme/motion/data contracts; existing modular imports. A home-made replacement would recreate substantial verified behavior. |
| MapLibre | Retain, defer public loading, repair worker packaging | Raster basemap, GeoJSON routes, bounds, zoom/pan, markers/playback and attribution. Heavy engine cost warrants a bounded alternative prototype. |
| Lucide | Retain | Named imports across 87 source sites; upstream `sideEffects: false`. No measured reason to maintain copied SVGs. |
| Playwright / test package / coverage / formatters / Svelte / Tailwind / Vite | Retain | Actual scripts, configs, tests and CSS/build consumers. Build-only dependencies are not initial browser payload. |
| Node types | Retain for now | Tool/config type contracts; no runtime payload benefit from removal. Do not mistake absence of app runtime imports for unused tooling types. |
| Python direct dependencies | Retain | `requests` in CLI/auth/smoke scripts, `psycopg` in media bridge, coverage in gates. |
| Go direct dependencies | No demonstrated removal | Advisor found import evidence across owning modules, including test-only WebAuthn helpers. Not a full transitive/security audit or `go mod tidy` result. |

### Loading bugs found during implementation

The initial map tests exposed MapLibre's indirect default worker URL resolving to a missing worker in Vite's optimized dependency directory. Production output also lacked an emitted worker. Host-local build adapters now use Vite's explicit `?worker&url` asset and MapLibre's `setWorkerUrl` API; both emitted clients include the worker.

Chromium retained a failed dynamic import, so a same-document “Retry map” could not recover the network-failed module. The UI instead offers the documented, explicit “Reload page” action. That reload preserves the deep-link URL, but ordinary in-memory year/sport/city selections reset as with any page reload; it is not advertised as a quiet data refresh.

The first tile test fixture also contained an invalid PNG. It was replaced with a valid synthetic image; external OSM tiles are not used as acceptance evidence.

Sources: [Vite worker URL imports](https://vite.dev/guide/features.html#import-with-query-suffixes), [Vite load-error handling](https://vite.dev/guide/build.html#load-error-handling), installed MapLibre 6.11.2 declarations (`setWorkerUrl`), [TanStack v8 toggle implementation](https://github.com/TanStack/table/blob/v8.21.3/packages/table-core/src/features/RowSorting.ts), [ECharts modular imports](https://echarts.apache.org/handbook/en/basics/import/).

## Emitted packaging measurements

Same pinned tools, local production builds. Raw bytes and gzip of each emitted file with `mtime=0`; static route graph follows Vite manifest `imports`, excluding `dynamicImports`. These are artifact costs, not measured network transfer, cache behavior, FCP/LCP or production latency.

| Measurement | Baseline raw / gzip bytes | Updated raw / gzip bytes |
| --- | --- | --- |
| Public landing static JS graph | 1,745,375 / 511,521 | 676,541 / 228,740 |
| Public aggregate emitted JS | 1,752,670 / 515,281 | 2,255,462 / 657,213 |
| Private aggregate emitted JS | 2,107,711 / 647,187 | 2,617,525 / 791,393 |

The public static landing graph falls about 61% raw / 55% gzip. Aggregate output grows because the previously missing functional worker is now emitted (509,702 raw / 144,167 gzip per host), plus chunk separation overhead. Worker bytes are fetched when a map is initialized, not on no-route public landing. Private aggregate values exceed the existing report-only bundle baselines; this is visible, not a lowered threshold or a claim of an aggregate reduction. Private overview still statically imports its map engine; deferring that import is a follow-up opportunity.

Before map changes, native table replacement alone saved 46,071 raw / 11,767 gzip bytes in public aggregate JS. Unused Three.js/uPlot removal did not reduce private emitted JS; minor hash/compression variation is not a meaningful saving.

## Replacement priorities

1. Prototype Leaflet against the actual raster/GeoJSON/bounds/marker/reduced-motion/keyboard/attribution contracts before considering a map-engine swap. Official stable registry version observed: 1.9.4. Integrity-verified upstream `dist/leaflet.js` is 147,552 raw / 42,434 gzip bytes; CSS is 14,806 / 3,522. Those are upstream artifact sizes, not a measured Iroha replacement bundle or a proven drop-in. [Official GeoJSON API](https://leafletjs.com/reference.html#geojson), [bounds API](https://leafletjs.com/reference.html#map-fitbounds).
2. Defer private overview map imports and public activity-detail-specific code where emitted graphs show benefit, retaining truthful loading/error and exact-record access.
3. Retain ECharts until a bounded simple-chart prototype proves equivalent unknown-value handling, provenance, theme, reduced motion, resolved paint and controls. Do not replace it with uPlot just because uPlot was declared.
4. Do not craft a tile engine, chart framework or copied icon library without evidence that the reduced payload outweighs maintenance/accessibility/interaction costs.

## Verification record

- Initial cleanup: `make check` exit 0; native sorting browser checks 2/2.
- Deferred-map and sorting checks: 6/6 against dev servers and 6/6 against both production preview servers. Both production preview process groups were task-owned, terminated, and ports verified unbound.
- Public type check/build passes after correcting the lazy module's TypeScript namespace to its host adapter.
- Full final validation/browser/integration and independent verdict: pending; no DONE/ready claim yet.

Retained specs: `public-table-sorting.spec.ts`, `public-map-loading.spec.ts`. Baseline, failed probes, build logs and advisor transcript remain in workstation scratch `.tmp/iroha-dependency-cleanup/`; final accepted evidence must be promoted to the owning PR.

Replacement PR will supersede #91, #92, #94 and #97 only once merged. Bot PRs, TS7 PRs and Node-types PRs remain untouched in this slice.
