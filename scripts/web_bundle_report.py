#!/usr/bin/env python3
"""Report raw and per-asset gzip sizes for a built private web app."""

from __future__ import annotations

import argparse
import gzip
from pathlib import Path
from typing import Iterable

REPO_ROOT = Path(__file__).resolve().parent.parent
DEFAULT_BUILD = REPO_ROOT / "apps" / "iroha-web" / "build"


def sizes(contents: Iterable[bytes]) -> tuple[int, int, int]:
    """Return asset count, raw bytes and the sum of individually gzipped bytes."""
    count = raw_bytes = gzip_bytes = 0
    for content in contents:
        count += 1
        raw_bytes += len(content)
        gzip_bytes += len(gzip.compress(content, mtime=0))
    return count, raw_bytes, gzip_bytes


def assets(root: Path, suffix: str) -> list[Path]:
    return sorted(path for path in root.rglob(f"*{suffix}") if path.is_file())


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("root", nargs="?", type=Path, default=DEFAULT_BUILD)
    args = parser.parse_args()
    if not args.root.is_dir():
        parser.error(f"build directory does not exist: {args.root}")

    for label, suffix in (("JavaScript", ".js"), ("CSS", ".css")):
        files = assets(args.root, suffix)
        count, raw_bytes, gzip_bytes = sizes(path.read_bytes() for path in files)
        print(
            f"{label}: {raw_bytes} raw bytes, {gzip_bytes} gzip bytes "
            f"across {count} files"
        )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
