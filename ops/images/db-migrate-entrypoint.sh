#!/bin/sh
set -eu

: "${DATABASE_URL:=${IROHA_DATABASE_URL:-}}"
if [ -z "$DATABASE_URL" ]; then
  echo "DATABASE_URL or IROHA_DATABASE_URL is required" >&2
  exit 2
fi

case "${1:-up}" in
  up) action=run ;;
  down) action=revert ;;
  status) action=info ;;
  *) echo "unsupported migration action: $1" >&2; exit 2 ;;
esac

exec sqlx migrate "$action" --source /migrations --no-dotenv --database-url "$DATABASE_URL"
