#!/usr/bin/env python3
import os
import shutil
import subprocess
import sys
import time
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
COMPOSE_FILE = ROOT / "ops" / "local-dev" / "compose.yaml"
DATABASE_URL = os.environ.get(
    "DATABASE_URL",
    "postgres://iroha:iroha_dev@127.0.0.1:5432/iroha?sslmode=disable",
)
DB_HOST = os.environ.get("IROHA_DB_HOST", "127.0.0.1")
DB_PORT = os.environ.get("IROHA_DB_PORT", "5432")
DB_USER = os.environ.get("IROHA_DB_USER", "iroha")
DB_NAME = os.environ.get("IROHA_DB_NAME", "iroha")


def usage() -> int:
    print("usage: uv run python scripts/dev_stack.py {start|stop|status|logs|reset|wait}", file=sys.stderr)
    return 2


def bianpai_bin() -> str:
    configured = os.environ.get("BIANPAI")
    if configured:
        return configured
    found = shutil.which("bianpai")
    if found:
        return found
    fallback = Path("/private/tmp/bianpai")
    if fallback.exists():
        return str(fallback)
    print(
        "bianpai is required. Set BIANPAI=/path/to/bianpai or install it on PATH.",
        file=sys.stderr,
    )
    return ""


def run(cmd: list[str], env: dict[str, str] | None = None) -> int:
    print("+ " + " ".join(cmd))
    return subprocess.call(cmd, cwd=ROOT, env=env)


def check_call(cmd: list[str], env: dict[str, str] | None = None) -> None:
    print("+ " + " ".join(cmd))
    subprocess.check_call(cmd, cwd=ROOT, env=env)


def bianpai_cmd() -> list[str]:
    binary = bianpai_bin()
    if not binary:
        raise SystemExit(2)
    return [binary, "--backend", "container", "-f", str(COMPOSE_FILE)]


def wait_for_db(timeout_s: int = 60) -> int:
    deadline = time.monotonic() + timeout_s
    cmd = ["pg_isready", "-h", DB_HOST, "-p", DB_PORT, "-U", DB_USER, "-d", DB_NAME]
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
    return run(["uv", "run", "python", "scripts/db.py", "apply"], env=env)


def main() -> int:
    if len(sys.argv) != 2:
        return usage()

    action = sys.argv[1]
    if action not in {"start", "stop", "status", "logs", "reset", "wait"}:
        return usage()

    if action == "wait":
        return wait_for_db()

    cmd = bianpai_cmd()
    if action == "start":
        check_call(["container", "system", "start"])
        check_call(cmd + ["up", "-d"])
        if wait_for_db() != 0:
            return 1
        return migrate()

    if action == "stop":
        return run(cmd + ["down"])

    if action == "status":
        return run(cmd + ["ps"])

    if action == "logs":
        return run(cmd + ["logs", "db"])

    check_call(cmd + ["down", "-v"])
    check_call(["container", "system", "start"])
    check_call(cmd + ["up", "-d"])
    if wait_for_db() != 0:
        return 1
    return migrate()


if __name__ == "__main__":
    raise SystemExit(main())
