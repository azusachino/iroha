#!/usr/bin/env python3
"""Rebuild changed application services in the Podman Compose stack."""
import argparse
import subprocess
import sys
import time
from pathlib import Path

import dev_stack


GO_ROOTS = (dev_stack.ROOT / "apps",)
WEB_ROOTS = tuple(dev_stack.ROOT / path for path in (
    "apps/iroha-web", "apps/iroha-public-site", "packages/iroha-shared",
))
WORKSPACE_FILES = tuple(dev_stack.ROOT / path for path in (
    "package.json", "bun.lock",
))


def snapshot() -> dict[Path, int]:
    paths = [
        path
        for root in GO_ROOTS
        for path in root.rglob("*.go")
        if "/iroha-web/" not in str(path)
    ]
    for root in WEB_ROOTS:
        for pattern in ("*.svelte", "*.ts", "*.css", "package.json"):
            paths.extend(path for path in root.rglob(pattern)
                         if not {"node_modules", ".svelte-kit", "build"}.intersection(path.parts))
    paths.extend(WORKSPACE_FILES)
    paths.extend([dev_stack.COMPOSE_FILE, dev_stack.APP_COMPOSE_FILE])
    return {path: path.stat().st_mtime_ns for path in paths if path.is_file()}


def changed_services(before: dict[Path, int], after: dict[Path, int]) -> list[str]:
    changed = {path for path, mtime in after.items() if before.get(path) != mtime}
    changed |= {path for path in before if path not in after}
    if not changed:
        return []
    services = set()
    frontend = {path for path in changed if path in WORKSPACE_FILES
                or any(path.is_relative_to(root) for root in WEB_ROOTS)}
    if changed - frontend:
        services.update(("server", "job"))
    if frontend or dev_stack.APP_COMPOSE_FILE in changed:
        services.add("web")
    return sorted(services)


def rebuild(services: list[str]) -> int:
    command = dev_stack.podman_cmd() + ["up", "--build", "-d", *services]
    print("+ " + " ".join(command), flush=True)
    return subprocess.call(command, cwd=dev_stack.ROOT)


def main() -> int:
    parser = argparse.ArgumentParser(description="watch source files and rebuild Podman Compose services")
    parser.add_argument("--interval-s", type=float, default=1.0)
    args = parser.parse_args()

    dev_stack.ensure_machine()
    previous = snapshot()
    print("watching Podman Compose services; press Ctrl-C to stop", flush=True)
    try:
        while True:
            time.sleep(args.interval_s)
            current = snapshot()
            services = changed_services(previous, current)
            previous = current
            if services and rebuild(services) != 0:
                return 1
    except KeyboardInterrupt:
        return 130


if __name__ == "__main__":
    raise SystemExit(main())
