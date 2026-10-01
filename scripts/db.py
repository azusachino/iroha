#!/usr/bin/env python3
import os
import subprocess
import sys
from urllib.parse import urlsplit


MIGRATIONS_DIR = "apps/iroha-server/db/migrations"


def main() -> int:
    if len(sys.argv) != 2 or sys.argv[1] not in {"apply", "apply-test", "rollback", "status"}:
        print("usage: uv run python scripts/db.py {apply|apply-test|rollback|status}", file=sys.stderr)
        return 2

    database_url = os.environ.get("DATABASE_URL") or os.environ.get("IROHA_DATABASE_URL")
    if not database_url:
        print("DATABASE_URL or IROHA_DATABASE_URL is required", file=sys.stderr)
        return 2

    if sys.argv[1] == "apply-test" and not urlsplit(database_url).path.removeprefix("/").startswith("iroha_test"):
        print("apply-test requires an iroha_test* database", file=sys.stderr)
        return 2

    action = {
        "apply": "run",
        "apply-test": "run",
        "rollback": "revert",
        "status": "info",
    }[sys.argv[1]]
    cmd = [
        "sqlx",
        "migrate",
        action,
        "--source",
        MIGRATIONS_DIR,
        "--no-dotenv",
        "--database-url",
        database_url,
    ]
    return subprocess.call(cmd)


if __name__ == "__main__":
    raise SystemExit(main())
