#!/usr/bin/env python3
"""Run a Go task across every module in the workspace.

The module list is discovered from go.work's `use (...)` block, so adding a
module to the workspace automatically brings it under fmt/vet/lint/test/build/
coverage with no Makefile edit. Tools (go, golangci-lint) come from mise; the
Makefile wraps commands in the mise-selected stable tool environment.
"""

import argparse
import subprocess
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
GO_WORK = REPO_ROOT / "go.work"
MIN_GO_STATEMENT_COVERAGE = 30.84

# task -> argv run with the module directory as cwd.
TASKS = {
    "fmt": ["golangci-lint", "fmt", "./..."],
    "fmt-check": ["golangci-lint", "fmt", "--diff", "./..."],
    "vet": ["go", "vet", "./..."],
    "lint": ["golangci-lint", "run", "./..."],
    # "test" and "coverage" run through run_coverage (tests plus the floor).
    "test": [],
    "build": ["go", "build", "./..."],
    # Needs DATABASE_URL naming a disposable database; see apps/iroha-runtime/testdb.
    "test-integration": ["go", "test", "-count=1", "-p", "1", "-tags=integration", "./..."],
    "coverage": [],
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


def run_coverage(modules: list[str]) -> int:
    """Run Go tests with coverage and report the statement-weighted baseline."""
    output_dir = REPO_ROOT / "dist" / "coverage" / "go"
    output_dir.mkdir(parents=True, exist_ok=True)
    results: list[tuple[str, int, int]] = []
    for module in modules:
        profile = output_dir / f"{module.replace('/', '_')}.cover.out"
        result = subprocess.run(
            ["go", "test", f"-coverprofile={profile}", "./..."],
            cwd=REPO_ROOT / module,
        )
        if result.returncode != 0:
            print(
                f"go_tasks: coverage failed in {module} (exit {result.returncode})",
                file=sys.stderr,
            )
            return result.returncode

        covered = total = 0
        for line in profile.read_text().splitlines():
            fields = line.split()
            if len(fields) != 3 or fields[0] == "mode:":
                continue
            statements, count = map(int, fields[1:])
            total += statements
            if count:
                covered += statements
        results.append((module, covered, total))
        percent = covered * 100 / total if total else 0
        print(f"{module}: {covered}/{total} statements ({percent:.2f}%)")

    covered = sum(row[1] for row in results)
    total = sum(row[2] for row in results)
    percent = covered * 100 / total if total else 0
    print(f"Go aggregate: {covered}/{total} statements ({percent:.2f}%)")
    if percent < MIN_GO_STATEMENT_COVERAGE:
        print(
            f"Go statement coverage {percent:.2f}% is below the "
            f"{MIN_GO_STATEMENT_COVERAGE:.2f}% floor",
            file=sys.stderr,
        )
        return 1
    return 0


def run_task(task: str, modules: list[str]) -> int:
    if task in {"test", "coverage"}:
        return run_coverage(modules)
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
