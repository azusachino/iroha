# Frontend responsive contract

Iroha uses one breakpoint vocabulary across the private cockpit, public projection,
shared theme package, and adopted design compositions. This keeps a component's
mobile behavior predictable when it moves between a route host and the shared
library.

## Breakpoints

| CSS width | Meaning | Required review widths |
| --- | --- | --- |
| `640px` | compact/mobile composition | 320, 375, 414 |
| `768px` | tablet/stacking composition | 768 |
| `1024px` | wide desktop transition | 1024 and above |

Only `@media (max-width: 640px)`, `@media (max-width: 768px)`, and
`@media (max-width: 1024px)` are permitted in frontend source. The numbers are
intentional CSS-pixel boundaries, not device names; a component should still be
usable at every width between them.

The source check is `make responsive-check`. It scans both applications and
`packages/iroha-shared/src`, so a new theme composition cannot quietly invent a
fourth breakpoint.

## Mobile acceptance

Every route must be opened at 390 × 844 and checked for:

- no page-level horizontal overflow;
- a reachable first heading and first primary action;
- controls that remain readable and do not overlap;
- charts/tables/maps with an intentional compact or overflow treatment;
- no loading skeleton left behind after data settles;
- the same result with reduced motion enabled.

The route audit is browser-based because type-checking cannot prove layout. Run
`make web-mobile-check BASE=https://...` against a running cockpit; it covers
the route inventory and the shared six-language matrix at the compact width.
