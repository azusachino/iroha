#!/usr/bin/env python3
"""Enforce Iroha's small, shared responsive breakpoint vocabulary."""

from __future__ import annotations

import re
import sys
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[1]
ALLOWED_MAX_WIDTHS = {640, 768, 1024}
MEDIA_RE = re.compile(r"@media\s*\(\s*max-width\s*:\s*(\d+)px\s*\)")
SOURCE_ROOTS = (
    REPO_ROOT / "apps/iroha-web/src",
    REPO_ROOT / "apps/iroha-public-site/src",
    REPO_ROOT / "packages/iroha-shared/src",
)
SOURCE_SUFFIXES = {".css", ".svelte"}


def find_violations(root: Path = REPO_ROOT) -> list[str]:
    violations: list[str] = []
    roots = (
        root / "apps/iroha-web/src",
        root / "apps/iroha-public-site/src",
        root / "packages/iroha-shared/src",
    )
    for source_root in roots:
        if not source_root.is_dir():
            continue
        for path in sorted(
            item
            for item in source_root.rglob("*")
            if item.is_file() and item.suffix in SOURCE_SUFFIXES
        ):
            for line_number, line in enumerate(path.read_text().splitlines(), start=1):
                for match in MEDIA_RE.finditer(line):
                    width = int(match.group(1))
                    if width not in ALLOWED_MAX_WIDTHS:
                        relative = path.relative_to(root).as_posix()
                        violations.append(
                            f"{relative}:{line_number}: max-width {width}px is not in "
                            f"{sorted(ALLOWED_MAX_WIDTHS)}"
                        )
    return sorted(violations)


def main() -> int:
    violations = find_violations()
    if violations:
        print("Responsive breakpoint check failed:", file=sys.stderr)
        for violation in violations:
            print(f"- {violation}", file=sys.stderr)
        return 1

    print("Responsive breakpoints: 640px / 768px / 1024px")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
