#!/usr/bin/env python3
"""Fail when a diff weakens the quality floor recorded in CONSTRAINTS.md."""

from __future__ import annotations

import argparse
import re
import subprocess
import sys
from dataclasses import dataclass

SUPPRESSIONS = re.compile(
    "|".join(
        map(
            re.escape,
            (
                "@ts-" + "ignore",
                "@ts-" + "nocheck",
                "eslint-" + "disable",
                "biome-" + "ignore",
                "svelte-" + "ignore",
                "# no" + "qa",
                "#" + "noqa",
                "# type:" + " ignore",
                "# type:" + "ignore",
                "# pragma:" + " no cover",
                "# pragma:" + "no cover",
                "istanbul " + "ignore",
                "nosem" + "grep",
                "gitleaks:" + "allow",
                "Stryker " + "disable",
                "//" + " nolint",
                "//" + "nolint",
            ),
        )
    )
)
STUBS = re.compile(
    r"throw new (?:Error|NotImplementedError).*not\s+implemented|"
    + r"raise Not" + r"ImplementedError|catch\s*\([^)]*\)\s*\{\s*\}|"
    + r"catch\s*\{\s*\}|\bTO" + r"DO\b|\bpass\s+#\s*stub",
    re.I,
)
SKIPS = re.compile(
    r"\." + r"(?:sk" + "ip|to" + "do)" + r"\b|\bxit\s*\(|\bxdescribe\s*\(|"
    + r"@pytest" + r"\." + "mark" + r"\." + "sk" + "ip|"
    + r"@unittest" + r"\." + "sk" + r"ip|\bt\.Skip\s*\("
)
MIN_BEFORE = re.compile(r"(?:>=|>|≥|at least|minimum|\bmin\b|no less than)\s*$", re.I)
MAX_BEFORE = re.compile(r"(?:<=|<|≤|at most|maximum|\bmax\b|no more than|under|below)\s*$", re.I)
MIN_AFTER = re.compile(r"^\s*\S*\s*(?:or more|or higher|must not fall|must not drop)", re.I)
MAX_AFTER = re.compile(r"^\s*\S*\s*(?:or less|or lower|must not grow|must not exceed)", re.I)
NUMBER = re.compile(r"\d+(?:\.\d+)?")


@dataclass(frozen=True)
class Change:
    path: str
    text: str


def parse_diff(diff: str) -> tuple[list[Change], list[Change], set[str]]:
    added: list[Change] = []
    removed: list[Change] = []
    deleted: set[str] = set()
    path = old_path = ""
    for line in diff.splitlines():
        if line.startswith("--- "):
            old_path = line[4:].removeprefix("a/")
        elif line.startswith("+++ "):
            new_path = line[4:].removeprefix("b/")
            path = old_path if new_path == "/dev/null" else new_path
            if new_path == "/dev/null":
                deleted.add(path)
        elif line.startswith("+") and not line.startswith("+++"):
            added.append(Change(path, line[1:]))
        elif line.startswith("-") and not line.startswith("---"):
            removed.append(Change(path, line[1:]))
    return added, removed, deleted


def is_test(path: str) -> bool:
    return bool(re.search(r"(?:\.test\.|\.spec\.|_test\.|(?:^|/)test_)", path))


def rule_key(text: str) -> str | None:
    stripped = text.strip()
    if stripped.startswith("|"):
        cells = [cell.strip() for cell in stripped.split("|") if cell.strip()]
        return cells[0] if cells else ""
    if stripped.startswith(("- ", "* ")):
        return stripped[2:].split(":", 1)[0].strip()
    return None


def thresholds(text: str) -> list[tuple[float, str | None]]:
    values = []
    for match in NUMBER.finditer(text):
        before = text[max(0, match.start() - 24) : match.start()]
        after = text[match.end() : match.end() + 40]
        direction = (
            "min" if MIN_BEFORE.search(before) or MIN_AFTER.search(after)
            else "max" if MAX_BEFORE.search(before) or MAX_AFTER.search(after)
            else None
        )
        values.append((float(match.group()), direction))
    return values


def find_floor_violations(diff: str) -> list[tuple[str, str]]:
    added, removed, deleted = parse_diff(diff)
    violations: list[tuple[str, str]] = []

    def flag(rule: str, path: str) -> None:
        finding = (rule, path)
        if finding not in violations:
            violations.append(finding)

    for change in added:
        if SUPPRESSIONS.search(change.text):
            flag("silenced-checker", change.path)
        if STUBS.search(change.text):
            flag("unfinished-work", change.path)
        if SKIPS.search(change.text):
            flag("test-made-easier", change.path)
        if change.path.endswith("CONSTRAINTS.md") and re.match(r"\|\s*(?:W|E)\d+\s*\|", change.text):
            flag("new-exception", change.path)

    for path in deleted:
        if is_test(path):
            flag("test-deleted", path)
    for change in removed:
        if is_test(change.path) and change.path not in deleted and re.search(
            r"\b(?:expect|assert(?:[A-Z]\w*)?|require|should)\b", change.text
        ):
            flag("assertion-removed", change.path)

    removed_rules = [item for item in removed if item.path.endswith("CONSTRAINTS.md") and rule_key(item.text) is not None]
    added_rules = [item for item in added if item.path.endswith("CONSTRAINTS.md") and rule_key(item.text) is not None]
    for old in removed_rules:
        replacement = next((item for item in added_rules if rule_key(item.text) == rule_key(old.text)), None)
        if replacement is None:
            if not re.match(r"\|\s*(?:W|E)\d+\s*\|", old.text.strip()):
                flag("rule-removed", old.path)
            continue
        before = thresholds(old.text)
        after = thresholds(replacement.text)
        for direction in ("min", "max", None):
            previous = [value for value in before if value[1] == direction]
            current = [value for value in after if value[1] == direction]
            for index, (old_value, _) in enumerate(previous):
                if index >= len(current):
                    flag("threshold-removed", old.path)
                    break
                new_value = current[index][0]
                weakened = (
                    new_value < old_value if direction == "min"
                    else new_value > old_value if direction == "max"
                    else new_value != old_value
                )
                if weakened:
                    flag("threshold-weakened" if direction else "threshold-changed", old.path)
                    break
    return violations


def git(args: list[str], *, diff_exit: bool = False) -> str | None:
    result = subprocess.run(
        ["git", *args], text=True, capture_output=True, check=False
    )
    if result.returncode == 0 or (diff_exit and result.returncode == 1):
        return result.stdout
    return None


def working_diff(base: str) -> str:
    merge_base = git(["merge-base", base, "HEAD"])
    if not merge_base or not merge_base.strip():
        raise RuntimeError(f"no merge base against {base}")
    tracked = git(["diff", "--unified=0", merge_base.strip(), "--"])
    if tracked is None:
        raise RuntimeError(f"could not diff against {merge_base.strip()}")
    untracked = git(["ls-files", "--others", "--exclude-standard"])
    if untracked is None:
        raise RuntimeError("could not list untracked files")
    diffs = [tracked]
    for path in untracked.splitlines():
        if not path:
            continue
        diff = git(["diff", "--no-index", "--unified=0", "/dev/null", "--", path], diff_exit=True)
        if diff is None:
            raise RuntimeError(f"could not diff untracked file {path}")
        diffs.append(diff)
    return "\n".join(diffs)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base", default="origin/main")
    args = parser.parse_args()
    try:
        violations = find_floor_violations(working_diff(args.base))
    except RuntimeError as error:
        print(f"quality-floor-guard: {error}", file=sys.stderr)
        return 2
    if not violations:
        print("quality-floor-guard: clean")
        return 0
    print(f"quality-floor-guard: {len(violations)} floor violation(s):", file=sys.stderr)
    for rule, path in violations:
        print(f"  [{rule}] {path}", file=sys.stderr)
    print("Fix the code or route a reviewed change through a tracked exception.", file=sys.stderr)
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
