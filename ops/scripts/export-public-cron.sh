#!/bin/sh
set -eu

# Regenerates the public-site data export and pushes it only if it changed.
#
# Runs inside the iroha-export-public image (ops/images/Containerfile.server,
# `export-public` target), invoked periodically by a k3s CronJob. That
# CronJob resource is defined in harus-k3s, not this repo -- iroha doesn't
# keep its own k8s manifests (see docs/roadmap.md Milestone 7). This script
# is the CronJob container's command; the CronJob's pod spec owns supplying
# IROHA_DATABASE_URL (in-cluster Postgres) and REPO_URL (a GitHub token or
# deploy key embedded in the URL, or an already-authenticated origin) as
# env/secret-mounted values -- provisioning those is that repo's job, not
# this script's.
#
# Each run clones a fresh, disposable shallow copy of the repo rather than
# reusing a persistent working tree: no drift or corruption risk across
# runs, and no volume to manage for what is otherwise a stateless batch job.

: "${REPO_URL:?REPO_URL is required, e.g. https://x-access-token:\$TOKEN@github.com/azusachino/iroha.git}"
DATA_DIR="${DATA_DIR:-apps/iroha-public-site/static/data}"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

git clone --depth 1 "$REPO_URL" "$WORKDIR"
cd "$WORKDIR"

iroha-export-public --out "$DATA_DIR"

if git diff --quiet -- "$DATA_DIR"; then
  echo "no change in $DATA_DIR, nothing to push"
  exit 0
fi

git config user.name "iroha-export-public"
git config user.email "iroha-export-public@users.noreply.github.com"
git add "$DATA_DIR"
git commit -m "chore: refresh public-site data export"
git push origin HEAD:main
