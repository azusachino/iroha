#!/usr/bin/env python3
"""Enforce the shared theme asset and import-direction boundary."""

from __future__ import annotations

import re
import sys
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[1]

THEME_ADAPTERS = {
    "apps/iroha-web/src/lib/themes/ThemeFrame.svelte",
    "apps/iroha-web/src/lib/themes/ThemeProvider.svelte",
    "apps/iroha-web/src/lib/themes/context.svelte.ts",
    "apps/iroha-web/src/lib/themes/registry.test.ts",
    "apps/iroha-web/src/lib/themes/registry.ts",
}

SOURCE_SUFFIXES = {".css", ".js", ".svelte", ".ts", ".tsx"}
FORBIDDEN_SHARED_IMPORTS = (
    re.compile(r"from\s+['\"]\$lib(?:/|['\"])"),
    re.compile(r"from\s+['\"](?:\.\./)+apps/iroha-(?:web|public-site)(?:/|['\"])"),
    re.compile(r"from\s+['\"]apps/iroha-(?:web|public-site)(?:/|['\"])"),
)


def _relative(path: Path, root: Path) -> str:
    return path.relative_to(root).as_posix()


def find_violations(root: Path = REPO_ROOT) -> list[str]:
    """Return boundary violations in stable, reviewable path order."""

    violations: list[str] = []
    apps_root = root / "apps"
    if apps_root.is_dir():
        for theme_dir in sorted(apps_root.glob("*/src/lib/themes")):
            if not theme_dir.is_dir():
                continue
            for path in sorted(item for item in theme_dir.rglob("*") if item.is_file()):
                relative = _relative(path, root)
                if relative not in THEME_ADAPTERS:
                    violations.append(
                        f"{relative}: themed assets belong under packages/iroha-shared"
                    )

    shared_root = root / "packages/iroha-shared/src"
    if shared_root.is_dir():
        for path in sorted(
            item for item in shared_root.rglob("*") if item.is_file() and item.suffix in SOURCE_SUFFIXES
        ):
            for line_number, line in enumerate(path.read_text().splitlines(), start=1):
                if any(pattern.search(line) for pattern in FORBIDDEN_SHARED_IMPORTS):
                    violations.append(
                        f"{_relative(path, root)}:{line_number}: shared code cannot import app sources"
                    )

    registry = root / "apps/iroha-web/src/lib/themes/registry.ts"
    if not registry.is_file():
        violations.append("apps/iroha-web/src/lib/themes/registry.ts: missing host registry adapter")
    elif "@iroha/shared/theme-ui/registry" not in registry.read_text():
        violations.append(
            "apps/iroha-web/src/lib/themes/registry.ts: registry must re-export the shared registry"
        )

    return sorted(violations)


def main() -> int:
    violations = find_violations()
    if violations:
        print("Theme asset boundary check failed:", file=sys.stderr)
        for violation in violations:
            print(f"- {violation}", file=sys.stderr)
        return 1

    print("Theme asset boundary: clean")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
