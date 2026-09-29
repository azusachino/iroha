# Capability Map: Iroha Audience and First Deployment Docs

Status: proposed

This map scopes the documentation work requested after the authentication design discussion. It does not authorize implementation or deployment changes.

| Module id          | Responsibility                                                                                                                                                 | Depends on         |
| ------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------ |
| `product-audience` | Explain Iroha's primary owner, logged-out visitors, product boundary, and private/public data distinction in the repository README.                            | —                  |
| `first-deployment` | Specify the first-owner setup journey for the k3s/Tailscale deployment, including the app-owned setup/auth/HAE handoff and links to deployment-owned commands. | `product-audience` |

Build order: `product-audience` → `first-deployment`.

## Boundary

- Iroha owns product behavior, authentication, first-run application steps, and portable Kubernetes manifests.
- The operator's infrastructure repository owns environment-specific ingress, secret materialization, storage, image distribution, commands, and cluster operations.
- The public overview distinguishes current behavior from the intended public guest/login experience. Documentation must not imply planned authentication or API behavior is already deployed.
