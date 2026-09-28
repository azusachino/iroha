# Spec: Product Audience and Access Model

Status: proposed Module: `product-audience`

## Objective

Update the Iroha repository README so a prospective user can tell whether Iroha fits them, who can access each surface, and what is private versus deliberately published.

### Audience

- **Owner/operator:** one person self-hosting Iroha as a personal data cockpit for health, movement, activity, sleep, media, expenses, and related history. They own the account and control ingestion
  and publication.
- **Logged-out visitor:** a person browsing the owner's intentionally published, sanitized public projection. Visitors do not get an account and do not see private canonical data.

Iroha is not a multi-user SaaS, social network, or open-registration service. Tailnet access is an additional network boundary, not a substitute for app authentication.

## Tech Stack / Source of Truth

Documentation only. Derive product claims from current implementation and approved decisions, especially:

- `docs/adr/0008-health-auto-export-http-intake.md` for the accepted target auth and public-access model;
- `docs/frontend-design-contract.md` for product position and experience;
- `docs/public-site-publishing.md` for the existing sanitized export boundary;
- server routes, public-export code, and auth implementation for implementation status.

When the target design is not implemented, identify it as planned rather than describing it as available.

## Commands

Documentation verification:

```sh
make fmt-docs-check
make check
```

No build, database, or deployment command is part of this capability.

## Project Structure

- Update `README.md` in the product overview and surfaces sections.
- Keep detailed first-deployment steps in the separate `docs/first-deployment.md` capability.
- Do not turn `docs/frontend-design-contract.md` into a public product overview; it remains an internal UI/design contract.

## Content Contract

The README must:

1. State that Iroha is for a single self-hosting owner maintaining a personal record from imports and connected sources.
2. Identify logged-out visitors as a separate audience for the owner's sanitized public projection.
3. Clearly distinguish private account/API data from public projection data; public access must not imply access to private records.
4. Explain that account registration is closed after one owner is provisioned; there are no visitor accounts.
5. Distinguish current shipped functionality from the planned authenticated public-web experience.
6. Link to the first-deployment guide and existing public publishing documentation where relevant.

## Code / Writing Style

Use the README's concise Markdown style: short headings, brief paragraphs, and small tables where they improve scanning. Prefer direct language and stable relative links. Do not copy deployment values
or commands from `harus-k3s`.

## Testing Strategy

- Run `make fmt-docs-check` for formatting.
- Run `make check` for the repository's pre-commit gate.
- Review relative links and verify that every current-versus-planned claim agrees with the source docs and implementation.

## Boundaries

- **Always:** Keep the owner as the sole account holder; describe anonymous access as limited to an explicitly sanitized projection; identify planned features honestly.
- **Ask first:** Changing the audience, data publication boundary, or one-owner model.
- **Never:** Claim the private API is safe for anonymous access; imply registration or multi-user access; expose private data as public; duplicate cluster-owned deployment details.

## Success Criteria

- A new reader can answer: who owns an instance, what a visitor can see, and what remains private.
- The README does not imply visitors can register or that the public projection grants private API access.
- Current and planned behavior are clearly distinguished.
- The first-deployment workflow is linked without duplicating its content.

## Open Questions

None at spec approval. Revisit if the accepted single-owner/public-projection decisions change.
