#!/usr/bin/env python3
"""Exploratory analysis of Apple Health daily-activity data (throwaway, pre-schema).

Streams two shapes out of an Apple Health export zip and prints their real shape
BEFORE designing tb_daily_summaries:

1. <ActivitySummary> elements  -- one per day, Apple's already-aggregated rings
   (Move / Exercise / Stand + goals).
2. Daily cumulative <Record>s  -- StepCount / DistanceWalkingRunning /
   FlightsClimbed, which arrive as many small interval records per day AND are
   double-counted across sources (iPhone + Watch report the same steps).

The point is to validate the daily rollup + the cross-source double-count caveat
(the analogue of sleep's InBed overlap) so the Go parser can de-dupe correctly.
Nothing here is canonical; the Go parser will re-derive these from raw evidence.

Usage:
  uv run python scripts/activity_explore.py <export.zip> [--recent 14]
"""

from __future__ import annotations

import argparse
import io
import re
import zipfile
from collections import defaultdict
from datetime import datetime

SUMMARY_RE = re.compile(r"<ActivitySummary ")
RECORD_RE = re.compile(r'<Record type="HKQuantityTypeIdentifier(StepCount|DistanceWalkingRunning|FlightsClimbed)"')
ATTR_RE = re.compile(r'([A-Za-z]+)="([^"]*)"')

# HealthKit quantity type -> our daily-metric slug.
DAILY = {
    "StepCount": "steps",
    "DistanceWalkingRunning": "distance_km",
    "FlightsClimbed": "flights",
}


def attrs(line: str) -> dict[str, str]:
    return dict(ATTR_RE.findall(line))


def local_date(value: str) -> str:
    # Attribute dates look like "2024-01-15 07:30:00 +0900"; the date prefix is
    # the local calendar date, which is the attribution we want.
    return value[:10]


def iter_lines(zip_path: str):
    with zipfile.ZipFile(zip_path) as zf:
        name = next(n for n in zf.namelist() if n.endswith("export.xml"))
        with zf.open(name) as raw:
            for line in io.TextIOWrapper(raw, encoding="utf-8"):
                yield line


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("zip_path")
    ap.add_argument("--recent", type=int, default=14)
    args = ap.parse_args()

    summaries: dict[str, dict[str, float]] = {}
    # metric -> date -> source -> summed value  (to expose cross-source overlap)
    daily: dict[str, dict[str, dict[str, float]]] = {
        m: defaultdict(lambda: defaultdict(float)) for m in DAILY.values()
    }

    for line in iter_lines(args.zip_path):
        if SUMMARY_RE.search(line):
            a = attrs(line)
            d = a.get("dateComponents", "")
            if not d:
                continue
            summaries[d] = {
                "move": float(a.get("activeEnergyBurned", 0) or 0),
                "move_goal": float(a.get("activeEnergyBurnedGoal", 0) or 0),
                "exercise": float(a.get("appleExerciseTime", 0) or 0),
                "exercise_goal": float(a.get("appleExerciseTimeGoal", 0) or 0),
                "stand": float(a.get("appleStandHours", 0) or 0),
                "stand_goal": float(a.get("appleStandHoursGoal", 0) or 0),
            }
            continue
        m = RECORD_RE.search(line)
        if not m:
            continue
        a = attrs(line)
        slug = DAILY[m.group(1)]
        val = float(a.get("value", 0) or 0)
        if slug == "distance_km":
            # DistanceWalkingRunning is in km already if unit=km, but can be mi.
            if a.get("unit") == "mi":
                val *= 1.609344
        date = local_date(a.get("startDate", ""))
        if not date:
            continue
        daily[slug][date][a.get("sourceName", "?")] += val

    # ---- ActivitySummary (rings) ----
    print("=== ActivitySummary (rings) ===")
    if summaries:
        dates = sorted(summaries)
        n = len(dates)
        move_avg = sum(s["move"] for s in summaries.values()) / n
        ex_avg = sum(s["exercise"] for s in summaries.values()) / n
        stand_avg = sum(s["stand"] for s in summaries.values()) / n
        move_closed = sum(1 for s in summaries.values() if s["move_goal"] and s["move"] >= s["move_goal"])
        ex_closed = sum(1 for s in summaries.values() if s["exercise_goal"] and s["exercise"] >= s["exercise_goal"])
        print(f"days: {n}  range: {dates[0]} .. {dates[-1]}")
        print(f"avg move: {move_avg:.0f} kcal   avg exercise: {ex_avg:.1f} min   avg stand: {stand_avg:.1f} h")
        print(f"move ring closed: {move_closed}/{n} ({100*move_closed/n:.0f}%)   exercise ring closed: {ex_closed}/{n} ({100*ex_closed/n:.0f}%)")
    else:
        print("NO ActivitySummary elements found -- rings not in this export.")

    # ---- Daily cumulative records + cross-source double-count ----
    for slug in DAILY.values():
        by_date = daily[slug]
        if not by_date:
            print(f"\n=== {slug} ===  (none found)")
            continue
        dates = sorted(by_date)
        # Two rollups: naive sum-all-sources vs per-day max-single-source.
        naive_total = sum(sum(src.values()) for src in by_date.values())
        max_total = sum(max(src.values()) for src in by_date.values())
        multi_src_days = sum(1 for src in by_date.values() if len(src) > 1)
        n = len(dates)
        unit = "km" if slug == "distance_km" else ""
        print(f"\n=== {slug} ===")
        print(f"days: {n}  range: {dates[0]} .. {dates[-1]}  multi-source days: {multi_src_days}")
        print(f"naive sum-all-sources total: {naive_total:.0f}{unit}   per-day max-source total: {max_total:.0f}{unit}")
        if naive_total:
            print(f"  => naive overcounts by {100*(naive_total-max_total)/naive_total:.1f}% (the double-count gotcha)")
        # sources seen
        srcs: dict[str, int] = defaultdict(int)
        for src in by_date.values():
            for s in src:
                srcs[s] += 1
        print("  sources:", ", ".join(f"{s}({c}d)" for s, c in sorted(srcs.items(), key=lambda x: -x[1])))

    # ---- recent days table (rings + max-source steps) ----
    print(f"\n=== recent {args.recent} days ===")
    steps_by_date = {d: max(src.values()) for d, src in daily["steps"].items()} if daily["steps"] else {}
    all_dates = sorted(set(summaries) | set(steps_by_date), reverse=True)[: args.recent]
    print(f"{'date':<12}{'move':>7}{'exer':>6}{'stand':>7}{'steps':>9}")
    for d in all_dates:
        s = summaries.get(d, {})
        move = f"{s['move']:.0f}" if s else "-"
        ex = f"{s['exercise']:.0f}" if s else "-"
        stand = f"{s['stand']:.0f}" if s else "-"
        steps = f"{steps_by_date.get(d, 0):.0f}" if d in steps_by_date else "-"
        print(f"{d:<12}{move:>7}{ex:>6}{stand:>7}{steps:>9}")


if __name__ == "__main__":
    main()
