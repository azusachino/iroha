# Iroha shared frontend library

This source-only library contains frontend code that is safe and useful in both `apps/iroha-web` and `apps/iroha-public-site`.

It owns presentation primitives, pure formatters, calendar helpers, and the six canonical design-language identities. Each app composes those shared identities with its own route components; private
API clients, authentication, export policy, and route registrations stay in their owning app.

Both apps expose the package as `@iroha/shared` through their Vite and TypeScript aliases. Keep imports pointed at this package when a component is genuinely identical across the private cockpit and
public archive.
