# Iroha shared frontend library

This source-only library contains frontend code that is safe and useful in both `apps/iroha-web` and `apps/iroha-public-site`.

It is intentionally limited to presentation primitives, pure formatters, and calendar helpers. Private API clients, authentication, export policy, and design-language registries stay in their owning
app.

Both apps expose the package as `@iroha/shared` through their Vite and TypeScript aliases. Keep imports pointed at this package when a component is genuinely identical across the private cockpit and
public archive.
