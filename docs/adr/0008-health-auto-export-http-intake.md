# ADR-0008: Adopt Health Auto Export for Automated Daily Health Intake

## Status

Accepted

## Date

2026-09-27

## Context

Iroha requires an automated, reliable daily intake channel for personal health data (steps, resting heart rate, sleep duration and stages, and workout GPS routes) from iOS/Apple Watch.

Prior approaches presented severe operational friction:
1. **Full `export.xml` ZIP uploads**: Require manual exports resulting in 500MB–1GB+ archives. While authoritative for historical backfills, this manual friction prevents daily dashboard currency.
2. **Apple Health Shortcuts (`iroha.health.shortcut.v1`)**: iOS Shortcuts lacks native JSON serialization, has unstable background scheduling, fails across midnight boundaries, and cannot reliably query complex types like sleep stages or activity ring goals. Additionally, ADR-0007 noted that no on-device producer was ever successfully built for this envelope.
3. **Authentication friction**: Forcing mandatory bearer token authentication required storing long-lived plain-text tokens inside third-party mobile apps, even though Iroha operates within a private, WireGuard-secured Tailscale network.

[Health Auto Export](https://help.healthyapps.dev/en/health-auto-export/automations/) (HAE) provides a mature, background-capable iOS/watchOS client that queries HealthKit natively and POSTs structured JSON (Format v2) to any custom REST endpoint.

## Decision

1. **Adopt Health Auto Export Format v2 JSON** as the primary producer and ingestion format for continuous daily health intake:
   - Introduce `KindHealthAutoExport = "health_auto_export"` in `iroha-core`.
   - Implement `ParseHealthAutoExport` in `iroha-providers` supporting multi-layout timestamps, fractional-hour sleep conversions, defensive unit fallbacks, GPS route points, and heart rate samplings.
   - Enforce sparse metric parsing: unconfigured metrics in the client payload are safely omitted without failing the import.

2. **Perimeter-Based Authentication with Configurable Token Fallback**:
   - `iroha-server` runs on a private Tailnet or local network perimeter.
   - When `IROHA_HEALTH_INTAKE_TOKEN` is unset or empty, `POST /api/v1/intake/health` accepts unauthenticated requests from within the tailnet boundary, removing plain-text secret storage from mobile apps.
   - When `IROHA_HEALTH_INTAKE_TOKEN` is set, `Authorization: Bearer <token>` is strictly enforced.

3. **10MB Body Limit**:
   - Increase `healthIntakeMaxBytes` from 2MB to 10MB to accommodate high-frequency workout GPS trackpoints and heart-rate series.

4. **Two-Tier Ingestion Architecture**:
   - **Continuous Daily Intake**: `POST /api/v1/intake/health` processes rolling 2–3 day HAE payloads with `bounded_replacement` semantics, leaving historical records untouched.
   - **Historical Bulk Backfill**: Retain `POST /api/v1/raw-files` and `POST /api/v1/imports` for full `apple_health_export` ZIP, FIT, TCX, and GPX archives with full snapshot reconciliation.

5. **Retirement and Removal of Apple Health Shortcut**:
   - Completely remove the experimental `apple_health_shortcut` parser, tests, and setup docs. Because no on-device producer was ever successfully deployed (as noted in ADR-0007), removing this dead code eliminates dual-routing complexity and makes Health Auto Export the single, clean daily intake path.

## Consequences

- Daily health metrics and workouts sync unattended in the background from iPhone and Apple Watch.
- Elimination of bearer token requirements simplifies iOS app setup while preserving optional token enforcement for defense-in-depth.
- Full GPS tracks and heart rate series are captured natively with workout activities.
- Clear separation between lightweight daily bounded intake and heavy historical bulk archive backfills.
