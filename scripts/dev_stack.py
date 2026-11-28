#!/usr/bin/env python3
import os
import shlex
import shutil
import subprocess
import sys
import time
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
COMPOSE_FILE = ROOT / "ops" / "local-dev" / "compose.yaml"
APP_COMPOSE_FILE = ROOT / "ops" / "local-dev" / "compose.app.yaml"
DATABASE_URL = os.environ.get(
    "DATABASE_URL",
    "postgres://iroha:iroha_dev@127.0.0.1:5432/iroha?sslmode=disable",
)
DB_USER = os.environ.get("IROHA_DB_USER", "iroha")
DB_NAME = os.environ.get("IROHA_DB_NAME", "iroha")
DB_PASSWORD = os.environ.get("IROHA_DB_PASSWORD", "iroha_dev")
# DB-backed Go tests wipe tables, so they get their own database on the dev
# server. The name prefix is what apps/iroha-runtime/testdb accepts.
TEST_DB_NAME = "iroha_test"
TEST_DATABASE_URL = os.environ.get(
    "IROHA_TEST_DATABASE_URL",
    f"postgres://{DB_USER}:{DB_PASSWORD}@127.0.0.1:5432/{TEST_DB_NAME}?sslmode=disable",
)


def usage() -> int:
    print("usage: uv run python scripts/dev_stack.py {start|deps|stop|status|logs|reset|wait|test-db|container-cmd}", file=sys.stderr)
    return 2


def container_cmd() -> list[str]:
    configured = os.environ.get("IROHA_CONTAINER", "").strip()
    if configured:
        return shlex.split(configured)
    podman = os.environ.get("PODMAN", "").strip()
    if podman:
        return [podman]
    found = shutil.which("podman")
    if found:
        return [found]
    colima = shutil.which("colima")
    if colima:
        return [colima, "nerdctl"]
    print(
        "podman or colima is required. Install one or set IROHA_CONTAINER='colima nerdctl' (or PODMAN=/path/to/podman).",
        file=sys.stderr,
    )
    raise SystemExit(2)


def run(cmd: list[str], env: dict[str, str] | None = None) -> int:
    print("+ " + " ".join(cmd))
    return subprocess.call(cmd, cwd=ROOT, env=env)


def check_call(cmd: list[str], env: dict[str, str] | None = None) -> None:
    print("+ " + " ".join(cmd))
    subprocess.check_call(cmd, cwd=ROOT, env=env)


def compose_cmd() -> list[str]:
    prefix = container_cmd()
    if os.environ.get("IROHA_CONTAINER", "").strip() or Path(prefix[0]).name == "colima":
        compose_prefix = [*prefix, "compose"]
    else:
        compose = os.environ.get("PODMAN_COMPOSE", "").strip() or shutil.which("podman-compose")
        if not compose:
            print(
                "podman-compose is required. Install it or set PODMAN_COMPOSE=/path/to/podman-compose.",
                file=sys.stderr,
            )
            raise SystemExit(2)
        compose_prefix = [compose]
    return [*compose_prefix, "-f", str(COMPOSE_FILE), "-f", str(APP_COMPOSE_FILE), "-p", "iroha-dev"]


def ensure_machine() -> None:
    container = container_cmd()
    result = subprocess.run([*container, "info"], cwd=ROOT, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE, text=True)
    if result.returncode == 0:
        return
    if Path(container[0]).name == "colima":
        start = subprocess.run([container[0], "start"], cwd=ROOT, capture_output=True, text=True)
        if start.returncode == 0:
            return
        print(
            "Colima is unavailable. Start an existing VM with `colima start`.",
            file=sys.stderr,
        )
        raise SystemExit(2)
    start = subprocess.run([*container, "machine", "start"], cwd=ROOT, capture_output=True, text=True)
    if start.returncode == 0:
        return
    print(
        "Podman is unavailable. Start an existing machine with `podman machine start`; "
        "if none exists, initialize one with a deliberate disk size (for example `--disk-size 30`).",
        file=sys.stderr,
    )
    raise SystemExit(2)


def wait_for_db(timeout_s: int = 60) -> int:
    deadline = time.monotonic() + timeout_s
    cmd = compose_cmd() + [
        "exec",
        "-T",
        "db",
        "pg_isready",
        "-h",
        "127.0.0.1",
        "-p",
        "5432",
        "-U",
        DB_USER,
        "-d",
        DB_NAME,
    ]
    while time.monotonic() < deadline:
        result = subprocess.run(cmd, cwd=ROOT, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
        if result.returncode == 0:
            print(result.stdout, end="")
            return 0
        time.sleep(1)
    print(f"database did not become ready within {timeout_s}s", file=sys.stderr)
    return 1


def migrate() -> int:
    env = os.environ.copy()
    env["DATABASE_URL"] = DATABASE_URL
    return run([sys.executable, "scripts/db.py", "apply"], env=env)


def recreate_test_db() -> int:
    """Drop and recreate the disposable integration database, then migrate it."""
    base = compose_cmd() + ["exec", "-T", "db", "env", f"PGPASSWORD={DB_PASSWORD}", "psql", "-h", "127.0.0.1", "-U", DB_USER, "-d", DB_NAME, "-v", "ON_ERROR_STOP=1"]
    for statement in (
        f"drop database if exists {TEST_DB_NAME} with (force)",
        f"create database {TEST_DB_NAME} owner {DB_USER}",
    ):
        if run(base + ["-c", statement]) != 0:
            return 1
    env = os.environ.copy()
    env["DATABASE_URL"] = TEST_DATABASE_URL
    return run([sys.executable, "scripts/db.py", "apply"], env=env)


def start_database() -> None:
    check_call(compose_cmd() + ["up", "-d", "db"])
    if wait_for_db() != 0:
        raise SystemExit(1)


def start_app() -> int:
    return run(compose_cmd() + ["up", "--build", "-d", "server", "job", "web"])


def main() -> int:
    if len(sys.argv) != 2:
        return usage()

    action = sys.argv[1]
    if action not in {"start", "deps", "stop", "status", "logs", "reset", "wait", "test-db", "container-cmd"}:
        return usage()

    if action == "container-cmd":
        print(shlex.join(container_cmd()))
        return 0

    if action == "wait":
        return wait_for_db()

    ensure_machine()
    if action in {"start", "deps"}:
        start_database()
        if migrate() != 0:
            return 1
        if action == "start":
            return start_app()
        return 0

    if action == "test-db":
        start_database()
        return recreate_test_db()

    if action == "stop":
        return run(compose_cmd() + ["down"])

    if action == "status":
        return run(compose_cmd() + ["ps"])

    if action == "logs":
        return run(compose_cmd() + ["logs", "db"])

    check_call(compose_cmd() + ["down", "-v"])
    start_database()
    if migrate() != 0:
        return 1
    return start_app()


if __name__ == "__main__":
    raise SystemExit(main())
