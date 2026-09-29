# Spec: First Iroha Deployment

Status: proposed Module: `first-deployment`

## Objective

Specify a clear first-run journey for the single owner deploying Iroha on the existing k3s/Tailscale path and enabling the intended public guest experience. The guide is a user/operator handoff, not a
replacement for the cluster repository's deployment runbook.

The guide documents the target flow. Until authentication, public APIs, and corresponding deployment changes are implemented and verified, it must label those steps as planned and must not present the
guide as an executable current-state procedure.

## Audience and User Story

As the operator of a self-hosted Iroha instance, I want to deploy the service, create its sole administrator through a private setup path, secure interactive access, and configure Health Auto Export
without exposing private records or administrative setup to the public internet.

The public visitor is a secondary user: they can browse only the anonymous, sanitized public projection and have no account or administrative actions.

## Tech Stack / Source of Truth

Documentation only. The application contract comes from:

- `docs/adr/0008-health-auto-export-http-intake.md`;
- `docs/dev-runtime.md` for local development (out of scope for this first-deployment path);
- `docs/health-auto-export-setup.md` for HAE payload/export details;
- the actual server, web, migration, ingress, and deployment code at implementation time.

Iroha owns portable Kubernetes manifests and their deployment instructions under `ops/k8s/`. Cluster-specific ingress, secret management, storage classes, image distribution, and operations remain with the operator's infrastructure repository.

## Commands

The guide may point to commands maintained by `harus-k3s`, but must not author a second command sequence. Documentation verification in Iroha:

```sh
make fmt-docs-check
make check
```

## Project Structure

- Add `docs/first-deployment.md` for the first-owner application journey.
- Link from `README.md` and, where useful, from the existing HAE setup guide.
- Keep portable Kubernetes manifests and generic deployment steps in Iroha; environment-specific ingress, secret materialization, cluster commands, and live deployment status remain with the operator's infrastructure repository.

## First-Run Workflow Contract

The guide must explain these steps in order:

1. **Choose the deployment path:** direct the operator to Iroha's `ops/k8s/README.md` for its portable Kubernetes baseline, and to their infrastructure repository for environment-specific instructions.
2. **Deploy privately first:** make the app reachable on the tailnet for setup; do not enable public exposure before ingress protections and application auth are in place.
3. **Create the owner account:** visit a tailnet-only, one-time setup path; create the sole administrator with username/password. Creation must be atomic and unavailable once an account exists. There
   is no public registration.
4. **Move to normal access:** use the public canonical app hostname, reachable from both public and tailnet clients, for ordinary login and passkey binding. Password login remains enabled. Explain the
   passkey add/manage flow without implying that passkeys are mandatory for the first setup.
5. **Provision Health Auto Export:** after account setup, generate the dedicated intake token in the authenticated admin UI, display it once, and copy it into HAE's supported custom header. Rotation
   immediately invalidates the previous token, so the operator must update HAE after rotating.
6. **Enable public visitor access deliberately:** public ingress allows only the anonymous, read-only sanitized projection and the authenticated HAE intake endpoint; private API, setup, admin, and
   other writes stay unavailable anonymously. Apply independent ingress and application rate limits. State that exact configured thresholds and proxy trust belong to the deployment/implementation
   instructions, not this guide.
7. **Know the recovery boundary:** there is no public or email password recovery. A self-host operator can run the documented maintenance reset to set a new password hash and revoke sessions; the
   actual command and operational safeguards are owned by the implementation/deployment docs.
8. **Verify:** provide observable checks for successful login, passkey binding, HAE intake, and anonymous access denial to private routes, linking to relevant project/deployment verification
   procedures.

## Writing Style

Write for an owner performing this once, not an engineer reverse-engineering the repository. Use numbered steps, explain security consequences at the step where they matter, and link to the owning
procedure for commands. Keep target behavior distinct from implemented behavior. Do not include example real tokens, credentials, personal URLs, or environment-specific values.

## Testing Strategy

- Run `make fmt-docs-check` and `make check`.
- Validate relative links.
- Compare every behavior statement to the accepted ADR and implemented deployment before changing status from planned to ready.
- When implementation is complete, verify the guide against a fresh database deployment and a public/tailnet route matrix; a prose review alone is not deployment evidence.

## Boundaries

- **Always:** Tailnet-only first-admin bootstrap; one owner; no registration; session-protected private API; separately scoped HAE credential; public data from the sanitized projection only;
  independent ingress and app rate limits.
- **Ask first:** Adding public routes, changing guest-visible data, altering the recovery model, or changing deployment target.
- **Never:** Treat tailnet membership or forwarded headers as application authentication; publish first-admin setup; expose private APIs to anonymous visitors; state planned controls are already live;
  copy `harus-k3s` deployment values into Iroha docs.

## Success Criteria

- An operator can identify the authoritative deployment instructions and the order of first-run actions.
- First-admin creation is described as tailnet-only and one-time, not public registration.
- The guide distinguishes the public canonical app from the private bootstrap path and does not rely on `X-Forwarded-For` for trust.
- HAE uses an independent, one-time-displayed, rotatable credential; the effect of immediate rotation is explicit.
- Anonymous guests are limited to public sanitized data; private APIs and admin actions require login.
- The guide cannot be mistaken for an already verified procedure before implementation and deployment gates pass.

## Open Questions

None at spec approval. Exact ingress hostname, rate-limit thresholds, trusted-proxy CIDRs, and command names are implementation/deployment decisions and must be resolved in their owning repositories
before operational instructions are published as current.
