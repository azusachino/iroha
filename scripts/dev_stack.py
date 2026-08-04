#!/usr/bin/env python3
import os
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


def usage() -> int:
    print("usage: uv run python scripts/dev_stack.py {start|deps|stop|status|logs|reset|wait}", file=sys.stderr)
    return 2


def podman_bin() -> str:
    configured = os.environ.get("PODMAN")
    if configured:
        return configured
    found = shutil.which("podman")
    if found:
        return found
    print("podman is required. Install it or set PODMAN=/path/to/podman.", file=sys.stderr)
    return ""


def compose_bin() -> str:
    configured = os.environ.get("PODMAN_COMPOSE")
    if configured:
        return configured
    found = shutil.which("podman-compose")
    if found:
        return found
    print(
        "podman-compose is required. Install it or set PODMAN_COMPOSE=/path/to/podman-compose.",
        file=sys.stderr,
    )
    return ""


def run(cmd: list[str], env: dict[str, str] | None = None) -> int:
    print("+ " + " ".join(cmd))
    return subprocess.call(cmd, cwd=ROOT, env=env)


def check_call(cmd: list[str], env: dict[str, str] | None = None) -> None:
    print("+ " + " ".join(cmd))
    subprocess.check_call(cmd, cwd=ROOT, env=env)


def podman_cmd() -> list[str]:
    binary = podman_bin()
    compose = compose_bin()
    if not binary or not compose:
        raise SystemExit(2)
    return [compose, "-f", str(COMPOSE_FILE), "-f", str(APP_COMPOSE_FILE), "-p", "iroha-dev"]


def ensure_machine() -> None:
    podman = podman_bin()
    if not podman:
        raise SystemExit(2)
    result = subprocess.run([podman, "info"], cwd=ROOT, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE, text=True)
    if result.returncode == 0:
        return
    start = subprocess.run([podman, "machine", "start"], cwd=ROOT, capture_output=True, text=True)
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
    cmd = podman_cmd() + [
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


def start_database() -> None:
    check_call(podman_cmd() + ["up", "-d", "db"])
    if wait_for_db() != 0:
        raise SystemExit(1)


def start_app() -> int:
    return run(podman_cmd() + ["up", "--build", "-d", "server", "job", "web"])


def main() -> int:
    if len(sys.argv) != 2:
        return usage()

    action = sys.argv[1]
    if action not in {"start", "deps", "stop", "status", "logs", "reset", "wait"}:
        return usage()

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

    if action == "stop":
        return run(podman_cmd() + ["down"])

    if action == "status":
        return run(podman_cmd() + ["ps"])

    if action == "logs":
        return run(podman_cmd() + ["logs", "db"])

    check_call(podman_cmd() + ["down", "-v"])
    start_database()
    if migrate() != 0:
        return 1
    return start_app()


if __name__ == "__main__":
    raise SystemExit(main())
