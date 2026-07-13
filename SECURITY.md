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

iroha is currently a personal, self-hosted project. There is no hosted production deployment; the default local configuration disables authentication (`IROHA_LOCAL_NO_AUTH=true`) and is intended for
localhost only — do not expose it to untrusted networks without enabling auth.
