# ADR-0008: Adopt Health Auto Export for Automated Daily Health Intake

## Status

Accepted (authentication decision updated)

## Date

2026-09-27

## Context

Iroha requires an automated, reliable daily intake channel for personal health data (steps, resting heart rate, sleep duration and stages, and workout GPS routes) from iOS/Apple Watch.

Prior approaches presented severe operational friction:

1. **Full `export.xml` ZIP uploads** require manual exports resulting in 500MB–1GB+ archives. They remain authoritative for historical backfills, but the friction prevents daily dashboard currency.
2. **Apple Health Shortcuts (`iroha.health.shortcut.v1`)** lack reliable JSON serialization and background scheduling, and cannot reliably query all needed HealthKit data.
3. **HAE requires a stable custom header** for its unattended REST integration. A credential stored in HAE must therefore be treated as a long-lived machine credential, not as the user's interactive
   login.
4. **Public hosting changes the trust boundary.** Iroha's private API contains sensitive health, activity, expense and other personal data and has write operations. A network perimeter alone is not
   sufficient when the service is publicly reachable.

[Health Auto Export](https://help.healthyapps.dev/en/health-auto-export/automations/) (HAE) provides a mature, background-capable iOS/watchOS client that queries HealthKit natively and POSTs
structured JSON (Format v2) to a custom REST endpoint.

Kite (`refs/web-auth/kite/`) is the UX reference for first use and passkey management: create the first administrator with a local username/password, sign in, then bind and manage passkeys in account
security settings. Iroha adopts that workflow's useful shape without adopting Kite's OAuth, LDAP, MFA, RBAC, or multi-user scope.

## Decision

### Health intake and publishing boundary

1. **Adopt Health Auto Export Format v2 JSON** as the primary producer and ingestion format for continuous daily health intake:
   - Introduce `KindHealthAutoExport = "health_auto_export"` in `iroha-core`.
   - Parse sparse Format v2 payloads, including timestamps, sleep durations, GPS route points, and heart-rate samplings. Unconfigured metrics are omitted safely.
2. Integrate the current public-site experience into `iroha-web` as the logged-out landing experience. Logged-out clients may access only explicitly allowlisted, read-only public APIs under a separate
   public namespace, backed by the sanitized public-export projection. Preserve the current public-export field and route policy, including route traces. Do not expose private `/api/v1` handlers to
   anonymous clients.
3. Keep `POST /api/v1/intake/health` as the sole publicly reachable write endpoint. Public ingress must deny other private API paths. Require its dedicated HAE credential on every request, including
   requests arriving over the tailnet; tailnet presence does not bypass application authentication. Use a custom header supported by HAE, not the web login session or a broad API credential.
4. Fail closed if the intake credential has not been provisioned. Never interpret an unset credential as permission for unauthenticated intake.
5. Apply independent rate limits at both public ingress and the application for anonymous public APIs, login/setup, and HAE intake. Numeric budgets are implementation-time configuration, not fixed by
   this ADR. The application may use forwarded client IP data only when the immediate peer is an explicitly configured trusted proxy; ignore such headers from other peers.

### First administrator, login, and passkeys

5. Implement a single-owner local account flow with no general registration:
   - On an uninitialized database, expose a one-time setup screen to create the sole administrator with username and password. Make setup reachable only through a tailnet-only hostname/ingress; never
     expose first-admin creation on the public ingress.
   - Atomically allow setup only while no account exists; after creation, permanently disable setup and registration through public routes.
   - Establish a session after successful setup. Use the public canonical app hostname for normal login and passkey binding, reachable from both public and tailnet clients.
   - Keep password login enabled. Do not add OIDC, OAuth, LDAP, multi-user account management, or additional user registration.
6. Add passkey login and account binding using WebAuthn:
   - An authenticated owner can name, add, list, and remove passkeys in account security settings.
   - Adding or removing a passkey requires recent password confirmation. A passkey is never bound merely because a browser reached the account page.
   - Login uses a short-lived, single-use WebAuthn challenge and verifies the browser response server-side before establishing a session.
   - Store credential public-key material and metadata, never a passkey private key. Validate RP ID and expected origin against the configured canonical HTTPS origin; do not derive trust from
     unvalidated forwarding headers.
7. Require an authenticated session for every private `/api/v1` route other than the narrowly scoped HAE intake endpoint and explicitly minimal health probes. A tailnet request receives the same
   application authorization behavior as a public request. The network boundary may further restrict reachability but does not grant identity.
8. Use server-managed sessions in `HttpOnly`, `Secure`, `SameSite=Lax` cookies, with expiry, logout/revocation, and CSRF protection for state-changing browser requests. Do not keep auth tokens in
   browser local storage. Retire the user-facing CLI rather than preserving its direct private-API access; provide supported domain workflows in the web app or retire workflows that have no web
   equivalent. Admin/security operations live in the authenticated web app. Other non-browser clients do not inherit a browser session or a tailnet bypass and require a separately approved design if
   introduced.

### Database and intake-token lifecycle

9. Persist auth state in PostgreSQL through forward SQLx migrations, and separate user/profile data from authentication data and domain evidence:
   - The single-owner user record contains account identity/profile data only. Password hashes, passkey credentials, sessions, and the HAE intake verifier live in separate auth/security records with
     explicit ownership and lifecycle.
   - Store a password hash (Argon2id or an equivalently approved password-hashing scheme), never the plaintext password.
   - Passkey credentials have a unique credential ID, WebAuthn credential/public-key state, user-provided name, creation time, and last-used time.
   - Sessions are server-side, expire, and are looked up by a hash of a random opaque session identifier; logout and credential changes can revoke them.
   - The HAE intake credential is represented by a verifier, not recoverable plaintext. Because the token is generated with high entropy, a cryptographic hash is suitable; generate at least 32 random
     bytes and encode for HAE's custom-header field.
10. An authenticated owner can rotate the HAE token from the admin page:
    - Generate and persist the replacement verifier, invalidating the prior token atomically.
    - Return/show the plaintext token once for copying into HAE; never provide a read-back endpoint, log it, or store it in browser persistence.
    - Explain that rotation requires updating HAE's configured custom header. Provide an explicit confirmation step to avoid accidental intake interruption.
11. Keep WebAuthn ceremony state short-lived and single-use. It may be ephemeral server-side state; it is not a durable account credential. Cap request bodies before parsing or persistence.
12. Do not build an in-app password-recovery flow. Provide a self-host operator-only maintenance reset that writes a new password hash and revokes active sessions. This is a break-glass maintenance
    operation, not a user-facing CLI or public endpoint.

### Ingestion semantics

13. Keep the 10 MiB HAE request-body limit to accommodate workout GPS trackpoints and heart-rate series.
14. **Continuous daily intake** processes rolling 2–3 day HAE payloads with `bounded_replacement` semantics, leaving historical records outside the declared window untouched.
15. **Historical bulk backfill** retains `POST /api/v1/raw-files` and `POST /api/v1/imports` for full Apple Health exports, FIT, TCX, and GPX archives, but these private routes require an
    authenticated session.
16. Retire the experimental Apple Health Shortcut receiver and parser; HAE is the single automated daily intake path.

## Consequences

- HAE can reach the public intake route using its required stable custom-header token; compromise of that token authorizes intake only, not access to private data or other API operations. Rotation
  immediately invalidates the previous token.
- Anonymous visitors receive only the allowlisted public-export projection through rate-limited public APIs. Private-data routes require application login even over the tailnet; network membership is
  not identity.
- First-admin setup is tailnet-only, one-time, and closes registration permanently. Normal app access uses one canonical public hostname from either network.
- The first-use experience is username/password setup followed by passkey enrollment. Password login remains enabled. There is no in-app password recovery; host operators can use the break-glass
  maintenance reset, which revokes sessions.
- Auth introduces migrations, password hashing, WebAuthn validation, session lifecycle, CSRF controls, and trusted-proxy/rate-limit configuration. These are security-critical and require
  endpoint-level tests before public exposure.
- User/profile records are separate from credentials and sessions. Iroha remains single-owner; multi-user administration, third-party identity providers, MFA, email recovery, and broad service/API
  keys are out of scope.
- The user-facing CLI is retired. Domain workflows must move to the web app or be retired; future non-browser clients need an explicit design.
- 0.5 implements the core of this decision: the HAE intake credential (stored verifier, fail-closed; one credential per device, so a rotation overlaps until the old one is revoked), owner setup,
  password login, server sessions with CSRF protection, token management on the admin page, and the break-glass `iroha-admin password reset` (item 12). The operator-only `iroha-admin intake-token`
  command manages the same credentials from the server container. Per-client limits for login, setup, and intake, per-credential intake quotas, and a public ingress that admits only
  `POST /api/v1/intake/health` followed in the same release. Passkeys remain unimplemented.
