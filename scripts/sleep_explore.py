#!/usr/bin/env python3
"""Exploratory analysis of Apple Health sleep data (throwaway, pre-schema).

Streams HKCategoryTypeIdentifierSleepAnalysis records out of an Apple Health
export zip, groups contiguous stage segments into nightly sessions via a
gap-merge threshold, and prints per-night metrics plus aggregate distributions.

The point is to validate the sessionization heuristic and see the real shape of
the data BEFORE designing tb_sleep_sessions / tb_sleep_segments. Nothing here is
canonical; the Go parser will re-derive these from raw evidence.

Overlap note: Apple's "InBed" value is a coarse envelope that OVERLAPS the fine
stage values (Core/Deep/REM/Awake). Naively summing every segment double-counts,
so per-category totals use an interval UNION.

Usage:
  uv run python scripts/sleep_explore.py <export.zip> [--gap-minutes 60] \
      [--min-asleep-hours 3.0] [--recent 12]
"""

from __future__ import annotations

import argparse
import io
import re
import zipfile
from collections import defaultdict
from datetime import datetime, timedelta

RECORD_RE = re.compile(r'<Record type="HKCategoryTypeIdentifierSleepAnalysis"[^>]*>')
ATTR_RE = re.compile(r'([A-Za-z]+)="([^"]*)"')

# HealthKit sleep value -> our stage slug.
STAGE = {
    "HKCategoryValueSleepAnalysisInBed": "in_bed",
    "HKCategoryValueSleepAnalysisAsleepUnspecified": "asleep_unspecified",
    "HKCategoryValueSleepAnalysisAwake": "awake",
    "HKCategoryValueSleepAnalysisAsleepCore": "core",
    "HKCategoryValueSleepAnalysisAsleepDeep": "deep",
    "HKCategoryValueSleepAnalysisAsleepREM": "rem",
}
# Stages that count as actually asleep (InBed and Awake do not).
ASLEEP = {"asleep_unspecified", "core", "deep", "rem"}
# Modern stage granularity only exists once the watch reports these.
GRANULAR = {"core", "deep", "rem"}

Segment = tuple  # (start: datetime, end: datetime, stage: str, source: str)


def parse_dt(value: str) -> datetime:
    return datetime.strptime(value, "%Y-%m-%d %H:%M:%S %z")


def iter_segments(zip_path: str):
    with zipfile.ZipFile(zip_path) as zf:
        name = next(n for n in zf.namelist() if n.endswith("export.xml"))
        with zf.open(name) as raw:
            for line in io.TextIOWrapper(raw, encoding="utf-8"):
                if "HKCategoryTypeIdentifierSleepAnalysis" not in line:
                    continue
                m = RECORD_RE.search(line)
                if not m:
                    continue
                attrs = dict(ATTR_RE.findall(m.group(0)))
                stage = STAGE.get(attrs.get("value", ""))
                if stage is None:
                    continue
                try:
                    start = parse_dt(attrs["startDate"])
                    end = parse_dt(attrs["endDate"])
                except (KeyError, ValueError):
                    continue
                if end < start:
                    continue
                yield (start, end, stage, attrs.get("sourceName", ""))


def union_seconds(intervals: list[tuple[datetime, datetime]]) -> float:
    if not intervals:
        return 0.0
    ivs = sorted(intervals)
    total = 0.0
    cs, ce = ivs[0]
    for s, e in ivs[1:]:
        if s > ce:
            total += (ce - cs).total_seconds()
            cs, ce = s, e
        elif e > ce:
            ce = e
    total += (ce - cs).total_seconds()
    return total


def sessionize(segments: list[Segment], gap: timedelta) -> list[dict]:
    sessions: list[dict] = []
    cur: dict | None = None
    for seg in segments:
        s, e, _stage, _src = seg
        if cur is not None and s - cur["end"] <= gap:
            cur["segs"].append(seg)
            if e > cur["end"]:
                cur["end"] = e
        else:
            if cur is not None:
                sessions.append(cur)
            cur = {"start": s, "end": e, "segs": [seg]}
    if cur is not None:
        sessions.append(cur)
    return sessions


def session_metrics(sess: dict) -> dict:
    by_stage: dict[str, list[tuple[datetime, datetime]]] = defaultdict(list)
    sources = set()
    for s, e, stage, src in sess["segs"]:
        by_stage[stage].append((s, e))
        sources.add(src)

    stage_secs = {st: union_seconds(iv) for st, iv in by_stage.items()}
    inbed = union_seconds(by_stage.get("in_bed", []))
    asleep = union_seconds([iv for st in ASLEEP for iv in by_stage.get(st, [])])
    span = (sess["end"] - sess["start"]).total_seconds()
    # Fall back to the session envelope when there is no InBed record (older data).
    time_in_bed = inbed if inbed > 0 else span
    granular = any(st in by_stage for st in GRANULAR)

    return {
        "wake_date": sess["end"].date(),
        "bedtime": sess["start"],
        "waketime": sess["end"],
        "time_in_bed_s": time_in_bed,
        "asleep_s": asleep,
        "efficiency": (asleep / time_in_bed) if time_in_bed else 0.0,
        "stage_secs": stage_secs,
        "granular": granular,
        "sources": sources,
        "n_segments": len(sess["segs"]),
    }


def h(seconds: float) -> float:
    return seconds / 3600.0


def m(seconds: float) -> int:
    return round(seconds / 60.0)


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("zip_path")
    ap.add_argument("--gap-minutes", type=float, default=60.0,
                    help="max gap between segments kept in one session")
    ap.add_argument("--min-asleep-hours", type=float, default=3.0,
                    help="a session with at least this much asleep is 'main sleep' (else nap)")
    ap.add_argument("--recent", type=int, default=12, help="how many recent main-sleep nights to print")
    args = ap.parse_args()

    print(f"reading {args.zip_path} (streaming export.xml)...")
    segments = sorted(iter_segments(args.zip_path), key=lambda s: s[0])
    print(f"sleep segments: {len(segments)}")
    if not segments:
        return 1

    gap = timedelta(minutes=args.gap_minutes)
    sessions = [session_metrics(s) for s in sessionize(segments, gap)]
    min_asleep_s = args.min_asleep_hours * 3600
    mains = [s for s in sessions if s["asleep_s"] >= min_asleep_s]
    naps = [s for s in sessions if s["asleep_s"] < min_asleep_s]

    print(f"gap-merge threshold: {args.gap_minutes:g} min")
    print(f"sessions: {len(sessions)}  (main sleep >= {args.min_asleep_hours:g}h: {len(mains)}, naps/short: {len(naps)})")

    # Sessions per year (by wake date) + granular coverage.
    per_year: dict[int, int] = defaultdict(int)
    per_year_gran: dict[int, int] = defaultdict(int)
    for s in mains:
        y = s["wake_date"].year
        per_year[y] += 1
        if s["granular"]:
            per_year_gran[y] += 1
    print("\nmain-sleep nights by year (granular = has Core/Deep/REM):")
    for y in sorted(per_year):
        print(f"  {y}: {per_year[y]:4d} nights   granular {per_year_gran[y]:4d}")

    # Overall averages over main sleep.
    def avg(xs):
        return sum(xs) / len(xs) if xs else 0.0

    print("\nmain-sleep averages (all years):")
    print(f"  time in bed : {h(avg([s['time_in_bed_s'] for s in mains])):.2f} h")
    print(f"  asleep      : {h(avg([s['asleep_s'] for s in mains])):.2f} h")
    print(f"  efficiency  : {avg([s['efficiency'] for s in mains]) * 100:.1f} %")

    # Stage mix over granular main-sleep nights (share of asleep+awake time).
    gran_mains = [s for s in mains if s["granular"]]
    if gran_mains:
        mix = defaultdict(float)
        for s in gran_mains:
            for st in ("core", "deep", "rem", "awake"):
                mix[st] += s["stage_secs"].get(st, 0.0)
        tot = sum(mix.values()) or 1.0
        print(f"\nstage mix over {len(gran_mains)} granular nights (% of core+deep+rem+awake):")
        for st in ("core", "deep", "rem", "awake"):
            print(f"  {st:5s}: {mix[st] / tot * 100:4.1f} %   ({h(mix[st] / len(gran_mains)):.2f} h/night avg)")

    # Session-duration histogram (validates the gap threshold isn't fragmenting nights).
    print("\nsession asleep-duration histogram (all sessions):")
    buckets = [(0, 0.5), (0.5, 1), (1, 3), (3, 5), (5, 7), (7, 9), (9, 24)]
    for lo, hi in buckets:
        n = sum(1 for s in sessions if lo <= h(s["asleep_s"]) < hi)
        bar = "#" * (n * 40 // max(len(sessions), 1))
        print(f"  [{lo:>4.1f},{hi:>4.1f})h {n:5d} {bar}")

    # Recent main-sleep nights.
    print(f"\nlast {args.recent} main-sleep nights:")
    print(f"  {'wake date':11s} {'bed':5s} {'wake':5s} {'inbed':>6s} {'asleep':>6s} "
          f"{'eff':>5s} {'core':>5s} {'deep':>5s} {'rem':>5s} {'awake':>5s}")
    for s in sorted(mains, key=lambda s: s["waketime"])[-args.recent:]:
        ss = s["stage_secs"]
        print(f"  {s['wake_date'].isoformat():11s} "
              f"{s['bedtime'].strftime('%H:%M')} {s['waketime'].strftime('%H:%M')} "
              f"{h(s['time_in_bed_s']):5.1f}h {h(s['asleep_s']):5.1f}h "
              f"{s['efficiency'] * 100:4.0f}% "
              f"{m(ss.get('core', 0)):4d}m {m(ss.get('deep', 0)):4d}m "
              f"{m(ss.get('rem', 0)):4d}m {m(ss.get('awake', 0)):4d}m")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
