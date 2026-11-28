# Tools (Bun TypeScript Checkers)

Bun TypeScript ports of the repository quality/theme/bundle checks originally in `scripts/`.
All scripts import `node:` built-ins only and run directly under Bun with no build step or extra dependencies.

## Scripts

- `tools/quality-floor-guard.ts`: Fails when a diff weakens `CONSTRAINTS.md` or the quality floor (`make quality-floor-check`).

- `tools/check-theme-boundary.ts`: Enforces the shared theme asset and import-direction boundary (`make theme-boundary-check`).
- `tools/check-motion-tokens.ts`: Fails if any `--motion-*` custom property is referenced without a definition in `packages/iroha-shared/src/theme/themes.css` (`make motion-tokens-check`).
- `tools/check-responsive-contract.ts`: Enforces the canonical `640px`, `768px`, and `1024px` `max-width` breakpoints (`make responsive-check`).
- `tools/check-web-coverage.ts`: Verifies `apps/iroha-web/coverage/coverage-summary.json` covers every app and shared source file (called by `make web-test`).
- `tools/web-bundle-report.ts`: Reports raw and per-asset gzip sizes for emitted `.js` and `.css` assets (`make web-bundle-report`).

## Testing & Parity

```bash
bun test tools/tests
make gate-parity
```
