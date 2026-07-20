# Security Policy

## Reporting a vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, report them privately via [GitHub Security Advisories](https://github.com/azusachino/iroha/security/advisories/new), or by email to **azusa146@gmail.com**.

Please include:

- a description of the issue and its impact,
- steps to reproduce or a proof of concept,
- affected commit / version.

You can expect an acknowledgement within a few days. Since iroha handles personal activity data, please give us a reasonable window to release a fix before any public disclosure.

## Scope

iroha is currently a personal, self-hosted project with no application-level authentication on the private API (`/api/v1`) — the deployment's network boundary (private LAN/NAS, not exposed publicly)
is the intended security control. Do not expose `iroha-server` to an untrusted network. `/public/v1` is the only surface designed for eventual public exposure; it serves sanitized, non-sensitive data
only.
