#!/usr/bin/env python3
"""Fail if a --motion-* custom property is referenced but never defined.

Guards the semantic-motion plan's token set (themes.css) against drift: a
consumer (app.css, a shared component) can reference var(--motion-x) years
after the token was renamed or removed, and nothing else catches that --
the browser just silently falls back to no transition. This is deliberately
one-directional (referenced-but-undefined only); an unused *definition* is
not an error, since Task 3 landed tokens before Task 4 consumed the first
one, and a family can legitimately be defined ahead of its first consumer.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[1]
DEFINITION_FILE_RELATIVE = "packages/iroha-shared/src/theme/themes.css"
DEFINE_RE = re.compile(r"--(motion-[\w-]+)\s*:")
USE_RE = re.compile(r"var\(\s*--(motion-[\w-]+)")
SOURCE_ROOTS = (
    "apps/iroha-web/src",
    "apps/iroha-public-site/src",
    "packages/iroha-shared/src",
)
SOURCE_SUFFIXES = {".css", ".svelte"}


def defined_tokens(root: Path = REPO_ROOT) -> set[str]:
    definition_file = root / DEFINITION_FILE_RELATIVE
    if not definition_file.is_file():
        return set()
    return set(DEFINE_RE.findall(definition_file.read_text()))


def find_violations(root: Path = REPO_ROOT) -> list[str]:
    known = defined_tokens(root)
    violations: list[str] = []
    for source_root_name in SOURCE_ROOTS:
        source_root = root / source_root_name
        if not source_root.is_dir():
            continue
        for path in sorted(
            item
            for item in source_root.rglob("*")
            if item.is_file() and item.suffix in SOURCE_SUFFIXES
        ):
            for line_number, line in enumerate(path.read_text().splitlines(), start=1):
                for match in USE_RE.finditer(line):
                    name = match.group(1)
                    if name not in known:
                        relative = path.relative_to(root).as_posix()
                        violations.append(
                            f"{relative}:{line_number}: var(--{name}) has no matching "
                            f"--{name}: definition in {DEFINITION_FILE_RELATIVE}"
                        )
    return sorted(violations)


def main() -> int:
    violations = find_violations()
    if violations:
        print("Motion token check failed:", file=sys.stderr)
        for violation in violations:
            print(f"- {violation}", file=sys.stderr)
        return 1

    print(f"Motion tokens: {len(defined_tokens())} defined, all references resolve")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
