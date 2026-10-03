# ADR-0009: Use a Native Bun Workspace and Stable Project Tools

## Status

Accepted by owner. This supersedes the unmerged pnpm proposal in [#117](https://github.com/azusachino/iroha/issues/117); implementation acceptance still requires fresh verification.

## Date

2026-10-03

## Context

The private frontend, public frontend and shared UI used three independent Bun installs and lockfiles. Node was implicit. During [#116](https://github.com/azusachino/iroha/issues/116) coverage diagnosis,
the app-scoped report contained zero of101 shared sources despite the documented coverage contract. Enabling external coverage alone measured84/101 and silently omitted17 files after transformation
errors. A repository-scoped test root measured every shared source, but isolated Svelte installations required a custom runtime resolver.

The owner approved a native workspace, initially considering pnpm, then explicitly chose to retain Bun. All project tools should follow LTS where available, otherwise stable releases, rather than exact
version pins. The package layout caused the measured failure; replacing the manager was unnecessary.

## Decision

- Use one native Bun workspace and root `bun.lock` for both frontend hosts and shared UI. Install once with a frozen lockfile. Each host declares the shared package with `workspace:*`.
- Keep Bun as package manager and script launcher/runtime where compatible. Retain SvelteKit/Vite, Vitest and Playwright; do not substitute Bun's bundler or test runner. Keep Node LTS available because
  the established Vitest V8 coverage requires Node's V8 engine, not Bun's JavaScriptCore. Do not force those tests through `bun --bun`.
- Make shared Svelte usage a peer dependency. The root supplies Svelte for repository-scoped tests; all consumers must resolve the same physical runtime. Preserve the shared ECharts registry contract.
- Retain the repository-scoped coverage root so never-imported shared TS/Svelte files are transformed. Remove the custom Svelte resolver; require a complete filesystem-to-report inventory after tests.
- Use Node `lts` and other mise tools `latest`. uv selects stable Python3, subject to the project minimum. Container build tools follow equivalent LTS/stable tags. Application dependencies remain locked;
  runtime service images and Go language/dependency requirements are not tool selectors and are unchanged.
- Validate local-check executables against mise's selection. Keys contain actual paths and versions, not floating selector strings. CI's native Go cache also includes the resolved Go version.
- Keep coverage floors, browser retries/assertions and deployment/privacy boundaries unchanged. This does not authorize publication, rollout or live migrations.

## Alternatives

Keeping the old layout with `allowExternal` failed complete source accounting. Keeping its custom Svelte resolver would preserve unnecessary resolution debt. A pnpm workspace proved coherent runtime
resolution, but a native Bun workspace achieves the same goal without replacing the manager. A framework rewrite is unrelated to the measured failure.

## Consequences

Tool versions can differ across installation dates; verification records must report actual versions. Frozen application locks do not make floating tools immutable. Both hosts explicitly declare the
already-locked GeoJSON types they import and zrender required by their Vite dedupe contract. Dependency lifecycle hooks are denied through an empty `trustedDependencies` list; the current native bindings are precompiled and require no trust grants.

The transitional pnpm lock was built using native frozen Bun-to-Yarn export and pnpm import. Bun then migrated it natively into the root `bun.lock`; no custom converter ships with the project. The
private Vite manifest now states its existing8.2.2 resolution instead of a^8.3.1 range overridden by Bun; public Vite retains8.3.1. Existing private PostCSS/NanoID overrides remain scoped so the public
site's newer locked patches are not downgraded. Lock conversion and supported macOS/Linux builds must be verified; successful import is not acceptance evidence.

The initial web floor was established with incomplete shared-source accounting. Preserve that historical record and its numeric floors; corrected reports need full inventory evidence. Other #116 audit,
recoverability, release-provenance and public-serving gaps remain separate.

## References

- [Bun workspaces](https://bun.sh/docs/pm/workspaces)
- [Bun lockfile migration](https://bun.sh/docs/pm/lockfile)
- [Bun lifecycle scripts](https://bun.sh/docs/pm/lifecycle)
- [mise version selection](https://mise.jdx.dev/configuration.html)
