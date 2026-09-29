# Kubernetes deployment

This directory contains Iroha's portable Kubernetes baseline. It deploys the
private web cockpit, server, worker, persistent PostGIS database, migration
Job, disposable Valkey cache, and separate public-site frontend. PostGIS is
the authoritative store; Valkey is only a rebuildable cache. It does not
install an Ingress controller or publish any route; configure private access
and public access deliberately in your cluster.

## Requirements

- Kubernetes with Kustomize (`kubectl apply -k`) and a default StorageClass.
- Build tools for Iroha's images, or an OCI registry where you publish them.
- A secret named `iroha-secrets` in namespace `iroha` with `POSTGRES_PASSWORD`.
  Optional `IROHA_ANILIST_TOKEN` and `IROHA_BANGUMI_TOKEN` keys enable private
  provider sync. Do not commit credentials. Create this Secret through your
  cluster's secret manager or a secure interactive process; do not put its
  values in shell history, manifests, or source control.

## Build and deploy

From the Iroha repository root, build the application images using the
project's pinned version:

```sh
make image-server image-job image-db-migrate image-web image-public-site
```

These targets build and import images into the local k3s containerd store.
For another cluster, build and publish the same five images to a registry
accessible to Kubernetes. The Containerfiles are in `ops/images/`; for
example, build the server image with:

```sh
podman build -t registry.example/iroha-server:vX.Y.Z --target server -f ops/images/Containerfile.server .
podman push registry.example/iroha-server:vX.Y.Z
```

Repeat for `job` and `db-migrate` targets in `Containerfile.server`, plus
`Containerfile.web` and `Containerfile.public-site`. Update the `images`
entries in `kustomization.yaml` to your registry and immutable version tags.

Create the `iroha` namespace and required Secret through your secure
secret-management workflow, then apply the baseline:

```sh
kubectl apply -k ops/k8s
kubectl -n iroha wait --for=condition=complete job/iroha-db-migrate --timeout=10m
kubectl -n iroha rollout status statefulset/iroha-postgis
kubectl -n iroha rollout status deployment/iroha-server
kubectl -n iroha rollout status deployment/iroha-job
kubectl -n iroha rollout status deployment/iroha-web
kubectl -n iroha rollout status deployment/iroha-public-site
```

The migration Job is a fixed-name, one-shot resource. Before applying a changed
migration image or re-running migrations, delete the completed Job first:

```sh
kubectl -n iroha delete job iroha-db-migrate --ignore-not-found
kubectl apply -k ops/k8s
```

The private web app is unauthenticated at the network edge only when protected
by Iroha's application login; never expose private routes to anonymous users.
The public-site service serves the sanitized `/public/v1` projection only via
its proxy. Add ingress rules in your own cluster configuration, keeping the
private app and API private. The baseline's Services are ClusterIP and expose
no NodePorts or LoadBalancers.

## Data and recovery

The default manifests request a 20Gi PVC for uploaded raw evidence and a 10Gi
PostGIS volume. Valkey has no persistent volume because its contents are
rebuildable. Set an appropriate StorageClass and capacity for your cluster.
Back up both persistent volumes: PostGIS contains canonical records and the
evidence volume contains source files. Kubernetes PVCs alone are not backups.
