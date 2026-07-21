#!/bin/sh
set -eu

: "${DATABASE_URL:=${IROHA_DATABASE_URL:-}}"
if [ -z "$DATABASE_URL" ]; then
  echo "DATABASE_URL or IROHA_DATABASE_URL is required" >&2
  exit 2
fi

exec goose -dir /migrations postgres "$DATABASE_URL" "${1:-up}"
