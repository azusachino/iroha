#!/usr/bin/env python3
"""Exploratory analysis of Apple Health body & vitals data (throwaway, pre-schema).

Streams the body/vitals quantity records out of an Apple Health export zip and,
per metric, prints cardinality (records, distinct days, readings-per-day),
value sanity (min/mean/max), and date range. The point is to decide the
per-metric DAILY AGGREGATION (avg vs min vs max vs last) before the Go parser
folds these into tb_daily_metrics -- unlike daily-activity's cumulative
sum+source-dedup, vitals are point readings that need a per-metric reducer.

Nothing here is canonical; the Go parser will re-derive from raw evidence.

Usage:
  uv run python scripts/vitals_explore.py <export.zip>
"""

from __future__ import annotations

import argparse
import io
import re
import statistics
import zipfile
from collections import defaultdict

ATTR_RE = re.compile(r'([A-Za-z]+)="([^"]*)"')

# HK quantity type -> (our metric slug, canonical unit). Grouped by expected
# cardinality so the output makes the aggregation decision obvious.
VITALS = {
    "HKQuantityTypeIdentifierRestingHeartRate": ("resting_hr", "count/min"),
    "HKQuantityTypeIdentifierWalkingHeartRateAverage": ("walking_hr_avg", "count/min"),
    "HKQuantityTypeIdentifierHeartRateVariabilitySDNN": ("hrv_sdnn", "ms"),
    "HKQuantityTypeIdentifierVO2Max": ("vo2max", "ml/kg/min"),
    "HKQuantityTypeIdentifierBodyMass": ("body_mass_kg", "kg"),
    "HKQuantityTypeIdentifierBodyFatPercentage": ("body_fat_pct", "%"),
    "HKQuantityTypeIdentifierLeanBodyMass": ("lean_body_mass_kg", "kg"),
    "HKQuantityTypeIdentifierOxygenSaturation": ("spo2", "%"),
    "HKQuantityTypeIdentifierRespiratoryRate": ("respiratory_rate", "count/min"),
    "HKQuantityTypeIdentifierBloodPressureSystolic": ("bp_systolic", "mmHg"),
    "HKQuantityTypeIdentifierBloodPressureDiastolic": ("bp_diastolic", "mmHg"),
}
TYPE_RE = re.compile(r'type="(HKQuantityTypeIdentifier(?:%s))"' % "|".join(
    t[len("HKQuantityTypeIdentifier"):] for t in VITALS
))


def attrs(line: str) -> dict[str, str]:
    return dict(ATTR_RE.findall(line))


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("zip_path")
    args = ap.parse_args()

    # metric -> list[value]; metric -> day -> count
    values: dict[str, list[float]] = defaultdict(list)
    per_day: dict[str, dict[str, int]] = defaultdict(lambda: defaultdict(int))
    dates: dict[str, list[str]] = defaultdict(list)

    with zipfile.ZipFile(args.zip_path) as zf:
        name = next(n for n in zf.namelist() if n.endswith("export.xml"))
        with zf.open(name) as raw:
            for line in io.TextIOWrapper(raw, encoding="utf-8"):
                if "HKQuantityTypeIdentifier" not in line:
                    continue
                m = TYPE_RE.search(line)
                if not m:
                    continue
                a = attrs(line)
                slug, _unit = VITALS[m.group(1)]
                try:
                    val = float(a.get("value", ""))
                except ValueError:
                    continue
                day = a.get("startDate", "")[:10]
                if not day:
                    continue
                values[slug].append(val)
                per_day[slug][day] += 1
                dates[slug].append(day)

    order = [v[0] for v in VITALS.values()]
    print(f"{'metric':<18}{'unit':<11}{'records':>9}{'days':>7}{'rd/day':>8}{'max/day':>8}"
          f"{'min':>9}{'mean':>9}{'max':>9}  range")
    for slug in order:
        vals = values.get(slug)
        unit = next(u for (s, u) in VITALS.values() if s == slug)
        if not vals:
            print(f"{slug:<18}{unit:<11}{'0':>9}  (none found)")
            continue
        days = per_day[slug]
        counts = list(days.values())
        rd = statistics.mean(counts)
        ds = sorted(dates[slug])
        print(f"{slug:<18}{unit:<11}{len(vals):>9}{len(days):>7}{rd:>8.1f}{max(counts):>8}"
              f"{min(vals):>9.2f}{statistics.mean(vals):>9.2f}{max(vals):>9.2f}"
              f"  {ds[0]}..{ds[-1]}")

    print("\nAggregation hint: rd/day~1 -> take the value; rd/day>1 -> pick a daily"
          " reducer (avg for hrv/respiratory, min/overnight for spo2, last/avg for bp);"
          " sparse days (days << export span) just yield fewer tb_daily_metrics rows.")


if __name__ == "__main__":
    main()
