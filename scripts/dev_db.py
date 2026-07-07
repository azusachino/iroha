#!/usr/bin/env python3
import subprocess
import sys


CONTAINER_NAME = "iroha-postgres"
VOLUME_NAME = "iroha-postgres-data"
IMAGE = "postgis/postgis:17-3.5"


def run(cmd: list[str]) -> int:
    print("+ " + " ".join(cmd))
    return subprocess.call(cmd)


def main() -> int:
    if len(sys.argv) != 2 or sys.argv[1] not in {"start", "stop", "status", "logs", "reset"}:
        print("usage: uv run python scripts/dev_db.py {start|stop|status|logs|reset}", file=sys.stderr)
        return 2

    action = sys.argv[1]

    if action == "start":
        run(["container", "system", "start"])
        run(["container", "volume", "create", VOLUME_NAME])
        return run(
            [
                "container",
                "run",
                "--name",
                CONTAINER_NAME,
                "--detach",
                "--env",
                "POSTGRES_DB=iroha",
                "--env",
                "POSTGRES_USER=iroha",
                "--env",
                "POSTGRES_PASSWORD=iroha_dev",
                "--publish",
                "5432:5432",
                "--volume",
                f"{VOLUME_NAME}:/var/lib/postgresql/data",
                IMAGE,
            ]
        )

    if action == "stop":
        return run(["container", "stop", CONTAINER_NAME])

    if action == "status":
        return run(["container", "list"])

    if action == "logs":
        return run(["container", "logs", CONTAINER_NAME])

    run(["container", "stop", CONTAINER_NAME])
    run(["container", "delete", CONTAINER_NAME])
    run(["container", "volume", "delete", VOLUME_NAME])
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
