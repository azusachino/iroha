# Kubernetes deployment (harus-mini k3s)

This is the recommended and only supported Kubernetes setup: the production
instance of Iroha runs on the single-node k3s cluster `harus-mini`. This
repository owns the **image builds** (`ops/images/`); the **cluster manifests**
are owned by the [harus-k3s](https://github.com/azusachino/harus-k3s)
repository in `03-core/iroha/` (namespace `harus-core`). A sanitized,
self-contained Kustomize copy of the same workloads is kept here for anyone
deploying outside the homelab — see [Portable baseline](#portable-baseline)
at the end of this page.

The cluster runs: `iroha-server`, `iroha-job`, and `iroha-web` Deployments (the
web image serves both the private cockpit and the public site under separate
ingress routes), the `iroha-postgis` StatefulSet
(`postgis/postgis:18-3.6-alpine`, 10Gi PVC, canonical store), the
`iroha-db-migrate` Job (recreated on every apply), a shared cluster Valkey used
as the disposable cache backend, and the `iroha-data` PVC for raw evidence.
Secrets land only as a `iroha-secrets` SealedSecret. See the harus-k3s
`03-core/iroha/README.md` for the full topology, networking, and backup story.

## Requirements

- The harus-workstation checkout (build tooling, Podman, mise).
- `kubectl` access to the cluster, and a harus-k3s checkout for the deploy step.
- The shared runtime base image (below).

## Prerequisite: the shared runtime base

Every runtime stage builds `FROM azusachino.com/media-base:py3.14-trixie-r2`,
a local-only image owned by the workstation repository, not by this one. It is
not published to any registry, so build it once before the first Iroha image
build on a machine and after a base refresh:

```sh
# in the harus-workstation repository
make media-base-build            # REFRESH=1 pulls upstream and bypasses cache
make media-base-check
```

The recipe is `containers/media-base/Dockerfile` there. Two consequences:
never pass `--pull` to an Iroha image build (Podman would try to fetch the
local-only tag from a registry that does not exist), and any other machine
needs that image built locally first — strangers can build it from
`ops/images/Containerfile.base.sample` in this repository.

## Build and import

From the repository root:

```sh
make images                     # server, job, db-migrate, web: build + import
make image-server TAG=deploy-abc1234   # one image with an explicit tag
```

These targets build with Podman and pipe `podman save` into
`sudo k3s ctr images import`, so they must run on the cluster node (or wherever
k3s's containerd is reachable) and end with the images present in the k3s
store. `make images-local` builds without importing.

The default tag is `v$(cat VERSION)`. Working rollouts between releases use a
`deploy-<shortsha>` tag built from the exact merged commit — whatever tag you
build must be recorded in harus-k3s `SERVICE_VERSIONS.yaml` next (below).
`IMAGE_NS` must stay `azusachino.com`: `imagePullPolicy` is `Never`, so a
prefix mismatch is an `ErrImageNeverPull` at apply time.

Before rebuilding under a tag that is already deployed, retain the previous
images for rollback:

```sh
for img in server job db-migrate web; do
  podman tag azusachino.com/iroha-$img:deploy-old azusachino.com/iroha-$img:rollback-pre-<change>
done
```

and import the rollback tags the same way (`podman save | sudo k3s ctr images
import`). Both stores retain them.

## Deploy

In the harus-k3s checkout:

1. Update the four `iroha-*` version entries in `SERVICE_VERSIONS.yaml` to the
   imported tags. The manifests in `03-core/iroha/` pin the same tags; keep
   the two in sync (the apply step validates this).
2. `make validate` — the harus-k3s CI gate.
3. `make apply-iroha` — validates the pins, recreates and waits for the
   `iroha-db-migrate` Job, then applies and waits for `iroha-server`,
   `iroha-job`, and `iroha-web` in sequence, and the public-site Service.

   `make apply-iroha-web` rolls only the web deployment;
   `uv run scripts/apply_iroha.py --dry-run` rehearses without applying.

## Verify

- `make apply-iroha` already waits for each rollout; check the pod phase if it
  fails.
- Server health: `/healthz` and `/readyz` return OK (the private cockpit is
  tailnet-only; check from the network).
- Web: `https://iroha.h.azusachino.com/` returns 200.
- From this repository, `make smoke-k3s-cache` verifies the live Valkey cache
  path against the deployed API.
- In harus-k3s, `make check-drift` confirms the applied objects match the
  manifests.

## Rollback

Point `SERVICE_VERSIONS.yaml` (and the manifest pins) back to the retained
`rollback-*` tags and re-run `make apply-iroha`. For an immediate revert
before that, `kubectl -n harus-core rollout undo deployment/iroha-server`
(restores the previous revision's image reference) — then still record the
revert in `SERVICE_VERSIONS.yaml` so the files match the cluster.

## Portable baseline

The YAML files in this directory (`namespace.yaml`, `config.yaml`,
`valkey.yaml`, `app.yaml`, `kustomization.yaml`) are a sanitized copy of the
homelab deployment: the same `iroha-db-migrate` Job, `iroha-server`,
`iroha-job`, and `iroha-web` Deployments (web serves both the private cockpit
and the public site — the `iroha-public-site` Service selects the web pods,
there is no fifth image), the `iroha-postgis` StatefulSet with its own 10Gi
PVC, the `iroha-data` PVC, and a disposable Valkey — everything a cluster
needs, with no dependency on homelab internals.

To deploy it on any Kubernetes cluster:

1. Build the images (`make images-local`) and publish them to a registry you
   control, then edit `kustomization.yaml`'s `images:` block (default
   `newName: azusachino.com/...` is a private, non-pullable name).
2. Create the secret first: `kubectl create namespace iroha` then
   `kubectl -n iroha create secret generic iroha-secrets --from-literal=POSTGRES_PASSWORD=...`
   (optional keys: `IROHA_ANILIST_TOKEN`, `IROHA_BANGUMI_TOKEN`,
   `DRIFTER_API_TOKEN`).
3. Set your hostname in `IROHA_ALLOWED_ORIGINS` (`config.yaml`),
   `IROHA_WEBAUTHN_RP_ID` / `IROHA_WEBAUTHN_ORIGINS` (`app.yaml`, server
   Deployment) — passkeys only work once these match your real origin.
4. `kubectl apply -k ops/k8s/`. The db-migrate Job runs once; on later
   upgrades delete it first (`kubectl -n iroha delete job iroha-db-migrate`)
   so apply re-creates it with the new image.

**Sanitization and sync rule.** The copy deliberately omits internal
hostnames, usernames, ingress routes, network policies, and backup
annotations. When the harus-k3s `03-core/iroha/` workloads change (env,
budgets, probes, topology), update this baseline in the same change or the
two drift — the same failure mode that retired the first portable baseline.

## History

This directory previously held a portable Kustomize baseline (namespace
`iroha`, a separate `iroha-public-site` image, `v0.6.0-rc2` tags). It had
drifted badly from the homelab deployment and was removed on 2026-11-13 in
favor of this runbook; a refreshed, sanitized baseline (this time matching
the deployed workloads, public-site Service selecting web, `iroha_v05`
database, WebAuthn env, drifter token) was added back the same day. The
homelab manifests in harus-k3s remain the single supported Kubernetes setup.
