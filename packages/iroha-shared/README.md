# Iroha shared frontend library

This source-only library contains frontend code that is safe and useful in both `apps/iroha-web` and `apps/iroha-public-site`.

It owns the canonical presentation primitives, period/date behavior, pure formatters, calendar helpers, the canonical design-language identities, the current production theme compositions, and the
adopted design compositions. Shared controls and theme components own their behavior, accessibility, visual language, and canonical data labels. Applications supply typed view data, snippets, and
navigation callbacks; private API clients, authentication, export policy, and route state stay in their owning app.

Both apps expose the package as `@iroha/shared` through their Vite and TypeScript aliases. Keep imports pointed at this package when a component is genuinely identical across the private cockpit and
public archive. If a component differs only in theme appearance, keep the behavior here and make each adopted composition explicit in the shared theme package rather than forking it inside an app. A
web-local theme component or visual primitive is migration debt, not a valid second home.

The theme manifest is also shared: it owns each language's mark, swatch, page lens, and extensible route registry. The design-composition manifest and package-owned composition renderer are a
separate, open-ended registry for adopted layout systems; a new composition is a real shared implementation, not a web-only variant. `themes.css` owns the semantic language tokens that both registries
consume. Do not add a second palette, page-intent table, route registry, or theme composition in an app.
