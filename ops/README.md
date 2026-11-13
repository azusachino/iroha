# Operations

Iroha has two supported runtime setups. Both are documented end to end; there
is no third path.

| Setup | Use it for | Documentation |
| --- | --- | --- |
| Local development stack | Day-to-day development, import smoke runs, frontend work | [`ops/local-dev/README.md`](local-dev/README.md), [`docs/dev-runtime.md`](../docs/dev-runtime.md) |
| Kubernetes on the homelab cluster | The deployed production instance (single-node k3s on `harus-mini`) | [`ops/k8s/README.md`](k8s/README.md) |

## Directory layout

```text
ops/local-dev/   Podman Compose stack: PostGIS dev database, server, job, web, Caddy
ops/images/      Containerfiles for the four production images + base sample
ops/k8s/         The k3s production runbook (manifests live in harus-k3s, not here)
ops/sql/         SQL operational helpers
```

## Image conventions

- Production images are `azusachino.com/iroha-{server,job,web,db-migrate}`.
  `IMAGE_NS` in the Makefile must match the prefix the harus-k3s repository
  pins in `SERVICE_VERSIONS.yaml` and its manifests; `imagePullPolicy` is
  `Never`, so a mismatch surfaces as `ErrImageNeverPull` at apply time, not as
  a pull failure to debug later.
- No registry is involved: `make images` builds with Podman and imports
  straight into the k3s containerd store on the cluster node.
- Tags are immutable by convention. Release builds use `v$(cat VERSION)`
  (the default `TAG`); working rollouts between releases use a
  `deploy-<shortsha>` tag built from the exact merged commit. Never re-point a
  tag at different content: build a new tag, or re-tag and restart rollouts
  explicitly, and retain the previous images under `rollback-*` tags first.
- Every runtime image builds `FROM` the shared `media-base` image owned by the
  workstation repository. See the prerequisite section in
  [`ops/k8s/README.md`](k8s/README.md); public builders without access to the
  private registry can build the base from `ops/images/Containerfile.base.sample`.
