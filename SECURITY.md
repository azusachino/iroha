# Security Policy

## Reporting a vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, report them privately via [GitHub Security Advisories](https://github.com/azusachino/iroha/security/advisories/new), or by email to [`azusa146@gmail.com`](mailto:azusa146@gmail.com).

Please include:

- a description of the issue and its impact,
- steps to reproduce or a proof of concept,
- affected commit / version.

You can expect an acknowledgement within a few days. Since iroha handles personal activity data, please give us a reasonable window to release a fix before any public disclosure.

## Scope

Iroha handles sensitive personal health, activity, and financial data. Private `/api/v1` routes require an owner session; state-changing browser requests also require a CSRF token. Health Auto Export intake uses a separate per-device bearer credential and grants intake access only. Anonymous `/public/v1` routes serve the validated, sanitized public projection. The public site proxies that namespace, not the private API. Keep owner setup, login, and all other private routes behind the intended private ingress; expose only explicitly approved public reads and credentialed intake routes.
