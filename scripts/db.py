#!/usr/bin/env python3
import os
import subprocess
import sys


MIGRATIONS_DIR = "apps/iroha-server/db/migrations"


def main() -> int:
    if len(sys.argv) != 2 or sys.argv[1] not in {"apply", "rollback", "status"}:
        print("usage: uv run python scripts/db.py {apply|rollback|status}", file=sys.stderr)
        return 2

    database_url = os.environ.get("DATABASE_URL") or os.environ.get("IROHA_DATABASE_URL")
    if not database_url:
        print("DATABASE_URL or IROHA_DATABASE_URL is required", file=sys.stderr)
        return 2

    action = {"apply": "up", "rollback": "down", "status": "status"}[sys.argv[1]]
    cmd = ["goose", "-dir", MIGRATIONS_DIR, "postgres", database_url, action]
    return subprocess.call(cmd)


if __name__ == "__main__":
    raise SystemExit(main())
