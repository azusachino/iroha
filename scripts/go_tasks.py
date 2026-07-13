#!/usr/bin/env python3
"""Run a Go task across every module in the workspace.

The module list is discovered from go.work's `use (...)` block, so adding a
module to the workspace automatically brings it under fmt/vet/lint/test/build
with no Makefile edit. Tools (go, golangci-lint) come from the Nix devShell;
the Makefile wraps this script in a single `nix develop --command`, so the
flake is evaluated once per task rather than once per module.
"""

import argparse
import subprocess
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
GO_WORK = REPO_ROOT / "go.work"

# task -> argv run with the module directory as cwd.
TASKS = {
    "fmt": ["golangci-lint", "fmt", "./..."],
    "fmt-check": ["golangci-lint", "fmt", "--diff", "./..."],
    "vet": ["go", "vet", "./..."],
    "lint": ["golangci-lint", "run", "./..."],
    "test": ["go", "test", "./..."],
    "build": ["go", "build", "./..."],
}


def discover_modules(go_work: Path) -> list[str]:
    """Return module directories (relative to repo root) from a go.work file."""
    modules: list[str] = []
    in_block = False
    for raw in go_work.read_text().splitlines():
        line = raw.strip()
        if line.startswith("//") or not line:
            continue
        if line.startswith("use ("):
            in_block = True
            continue
        if in_block:
            if line == ")":
                in_block = False
                continue
            modules.append(line.lstrip("./"))
        elif line.startswith("use "):
            # Single-line form: `use ./apps/foo`.
            modules.append(line[len("use "):].strip().lstrip("./"))
    return modules


def run_task(task: str, modules: list[str]) -> int:
    cmd = TASKS[task]
    for module in modules:
        result = subprocess.run(cmd, cwd=REPO_ROOT / module)
        if result.returncode != 0:
            hint = " (run: make fmt)" if task == "fmt-check" else ""
            print(
                f"go_tasks: {task} failed in {module} (exit {result.returncode}){hint}",
                file=sys.stderr,
            )
            return result.returncode
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("task", choices=sorted(TASKS))
    args = parser.parse_args()

    modules = discover_modules(GO_WORK)
    if not modules:
        print(f"go_tasks: no modules found in {GO_WORK}", file=sys.stderr)
        return 1
    return run_task(args.task, modules)


if __name__ == "__main__":
    raise SystemExit(main())
