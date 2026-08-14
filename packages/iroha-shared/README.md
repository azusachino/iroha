# Iroha shared frontend library

This source-only library contains frontend code that is safe and useful in both `apps/iroha-web` and `apps/iroha-public-site`.

It owns the canonical presentation primitives, period/date behavior, pure formatters, calendar helpers, and the six canonical design-language identities. Shared controls own their behavior,
accessibility, and canonical data labels; an app supplies the selected theme appearance and route-specific data. Each app composes those shared identities with its own route components; private API
clients, authentication, export policy, and route registrations stay in their owning app.

Both apps expose the package as `@iroha/shared` through their Vite and TypeScript aliases. Keep imports pointed at this package when a component is genuinely identical across the private cockpit and
public archive. If a component differs only in theme appearance, keep the behavior here and pass the appearance from the theme registry rather than forking the component.

The theme manifest is also shared: it owns each language's mark, swatch, and page lens. The private cockpit owns the route components, while themes.css owns the semantic language tokens that those
components consume. Do not add a second palette or page-intent table in a route.
