"""Fail when Vitest silently omits app or shared source from its coverage report."""

import json
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
SOURCE_DIRS = ("apps/iroha-web/src", "packages/iroha-shared/src")
METRICS = ("statements", "branches", "functions", "lines")


def missing_sources(root: Path, report: dict) -> list[str]:
    root = root.resolve()
    if not isinstance(report, dict):
        raise ValueError("report must be an object")
    measured = {
        Path(name).resolve()
        for name, metrics in report.items()
        if name != "total"
        and isinstance(metrics, dict)
        and all(isinstance(metrics.get(metric), dict) for metric in METRICS)
    }
    expected = {
        path.resolve()
        for directory in SOURCE_DIRS
        for path in (root / directory).rglob("*")
        if path.is_file()
        and path.suffix in {".js", ".ts", ".svelte"}
        and not path.name.endswith((".test.js", ".spec.js", ".test.ts", ".spec.ts"))
    }
    if not expected:
        raise ValueError("source inventory is empty")
    return sorted(str(path.relative_to(root)) for path in expected - measured)


def main() -> int:
    try:
        report = json.loads(
            (ROOT / "apps/iroha-web/coverage/coverage-summary.json").read_text()
        )
        missing = missing_sources(ROOT, report)
    except (OSError, ValueError, TypeError) as error:
        print(f"web-coverage: cannot verify report ({type(error).__name__})", file=sys.stderr)
        return 1
    if missing:
        print(f"web-coverage: {len(missing)} source files omitted", file=sys.stderr)
        for path in missing:
            print(f"  {path}", file=sys.stderr)
        return 1
    print("web-coverage: all app and shared source files measured")
    return 0


if __name__ == "__main__":
    sys.exit(main())
