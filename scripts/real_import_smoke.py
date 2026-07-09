#!/usr/bin/env python3
"""Smoke-test a real import through the iroha HTTP API.

Three modes:
  - default (no --assert / --assert-reprocess): the original single-import
    smoke check (upload, import, poll, read back a few activity
    sub-resources).
  - --assert: a two-phase delta check against the real Postgres/PostGIS
    database. Runs the same file through the import pipeline twice
    (import #1 = full parse, import #2 = same sha256 + parser_version, should
    hit the reuse guard and short-circuit) and asserts on tables the
    pre-refactor parser never wrote to: tb_apple_source_items,
    tb_import_snapshots, tb_activity_samplings, tb_activity_laps, plus
    route points joined to those new source items, and the absence of any
    NEW standalone provider='gpx' activities created by this run.
  - --assert-reprocess: exercises the purge-then-repersist reprocess path
    (see the "iroha:decision:apple-reprocess-from-raw" ADR). Precondition:
    the server must already be running with a parser_version DIFFERENT from
    the parser_version of the last COMPLETED import of this file (so the
    disposition is dispositionReprocess, not skip). Captures counts before
    the import, runs the import once, captures counts after, and asserts
    the derived data was REPLACED, not appended: activities, apple source
    items, import snapshots, samplings, laps, and route points on source
    items are all unchanged in total, and there are zero apple_health
    activities left without a source item (which would mean the purge
    missed rows and change-detection is now lying).
"""
import argparse
import json
import mimetypes
import os
import subprocess
import sys
import time
import uuid
from pathlib import Path
from urllib.error import HTTPError
from urllib.request import Request, urlopen

DEFAULT_DSN = "postgres://iroha:iroha_dev@127.0.0.1:5432/iroha?sslmode=disable"


def main() -> int:
    parser = argparse.ArgumentParser(description="Smoke-test a real import through the iroha HTTP API.")
    parser.add_argument("file", type=Path)
    parser.add_argument("--api-base", default="http://127.0.0.1:8080")
    parser.add_argument("--source-kind", default="apple_health_export")
    parser.add_argument("--parser-kind", default="apple_health_export")
    parser.add_argument("--uploaded-via", default="telegram")
    parser.add_argument("--timeout-s", type=int, default=360)
    parser.add_argument(
        "--assert",
        dest="do_assert",
        action="store_true",
        help="run the two-phase delta check against Postgres instead of the single-shot smoke",
    )
    parser.add_argument(
        "--assert-reprocess",
        dest="do_assert_reprocess",
        action="store_true",
        help=(
            "run the purge-then-repersist reprocess check against Postgres; "
            "requires the server to be running at a parser_version different "
            "from the last completed import of this file"
        ),
    )
    parser.add_argument("--dsn", default=os.environ.get("IROHA_DATABASE_URL", DEFAULT_DSN))
    args = parser.parse_args()

    if args.do_assert and args.do_assert_reprocess:
        parser.error("--assert and --assert-reprocess are mutually exclusive")

    if args.do_assert_reprocess:
        return run_assert_reprocess_mode(args)
    if args.do_assert:
        return run_assert_mode(args)
    return run_basic_mode(args)


def run_basic_mode(args: argparse.Namespace) -> int:
    raw = upload_raw_file(args)
    print(f"raw_file_id={raw['id']}")
    print(f"raw_duplicate={str(raw.get('duplicate', False)).lower()}")

    job = create_import(args, raw["id"])
    print(f"import_id={job['id']}")

    final = wait_import(args, job["id"])
    print(f"import_status={final['status']}")
    if final.get("error_message"):
        print(f"import_error={final['error_message']}")
        return 1

    # The activities list endpoint returns an envelope: {items, next_cursor,
    # has_more}. Route/samplings/laps endpoints stay bare arrays.
    activities = get_json(args.api_base, "/api/v1/activities?limit=5")["items"]
    print(f"activities_list_len={len(activities)}")
    if activities:
        activity_id = activities[0]["id"]
        route = get_json(args.api_base, f"/api/v1/activities/{activity_id}/route")
        samplings = get_json(args.api_base, f"/api/v1/activities/{activity_id}/samplings")
        laps = get_json(args.api_base, f"/api/v1/activities/{activity_id}/laps")
        print(f"first_activity_route_len={len(route)}")
        print(f"first_activity_samplings_len={len(samplings)}")
        print(f"first_activity_laps_len={len(laps)}")
    return 0


# --- assert mode -----------------------------------------------------------


def run_assert_mode(args: argparse.Namespace) -> int:
    run_started_at = psql_scalar(args.dsn, "select now()::text;")
    print(f"run_started_at={run_started_at}")

    print("\n=== baseline (before import #1) ===")
    baseline = capture_counts(args.dsn)
    print_counts(baseline)

    print("\n=== import #1 (full pipeline expected) ===")
    raw1 = upload_raw_file(args)
    print(f"raw_file_id={raw1['id']} duplicate={raw1.get('duplicate', False)}")
    job1 = create_import(args, raw1["id"])
    print(f"import_id={job1['id']}")
    final1 = wait_import(args, job1["id"])
    print(f"import_status={final1['status']}")
    if final1.get("error_message"):
        print(f"import_error={final1['error_message']}")
        return 1
    counts1 = capture_counts(args.dsn)
    print_counts(counts1)

    print("\n=== import #2 (same file; reuse guard expected) ===")
    raw2 = upload_raw_file(args)
    print(f"raw_file_id={raw2['id']} duplicate={raw2.get('duplicate', False)}")
    job2 = create_import(args, raw2["id"])
    print(f"import_id={job2['id']}")
    final2 = wait_import(args, job2["id"])
    print(f"import_status={final2['status']}")
    if final2.get("error_message"):
        print(f"import_error={final2['error_message']}")
        return 1
    counts2 = capture_counts(args.dsn)
    print_counts(counts2)

    new_gpx_after_1 = new_gpx_activity_count(args.dsn, run_started_at)
    new_gpx_after_2 = new_gpx_activity_count(args.dsn, run_started_at)

    print("\n=== assertions ===")
    failures = []

    def check(label: str, condition: bool, detail: str) -> None:
        status = "PASS" if condition else "FAIL"
        print(f"[{status}] {label}: {detail}")
        if not condition:
            failures.append(label)

    d1 = delta(baseline, counts1)
    check(
        "import#1 apple_source_items(workout) > 0",
        counts1["source_items_workout"] > 0,
        f"counts1.source_items_workout={counts1['source_items_workout']}",
    )
    check(
        "import#1 import_snapshots increased by exactly 1",
        d1["snapshots"] == 1,
        f"delta snapshots={d1['snapshots']} (baseline={baseline['snapshots']} -> {counts1['snapshots']})",
    )
    check(
        "import#1 samplings > 0",
        counts1["samplings_total"] > 0,
        f"counts1.samplings_total={counts1['samplings_total']}",
    )
    check(
        "import#1 samplings include heart_rate",
        "heart_rate" in counts1["sampling_types"],
        f"counts1.sampling_types={sorted(counts1['sampling_types'])}",
    )
    check(
        "import#1 route points joined to new source items > 0",
        counts1["route_points_on_source_items"] > 0,
        f"counts1.route_points_on_source_items={counts1['route_points_on_source_items']}",
    )
    check(
        "import#1 creates zero NEW standalone gpx activities",
        new_gpx_after_1 == 0,
        f"new_gpx_activities_since_run_start={new_gpx_after_1}",
    )

    d2 = delta(counts1, counts2)
    check(
        "import#2 (reuse) apple_source_items unchanged",
        d2["source_items_total"] == 0,
        f"delta source_items_total={d2['source_items_total']} ({counts1['source_items_total']} -> {counts2['source_items_total']})",
    )
    check(
        "import#2 (reuse) import_snapshots unchanged",
        d2["snapshots"] == 0,
        f"delta snapshots={d2['snapshots']} ({counts1['snapshots']} -> {counts2['snapshots']})",
    )
    check(
        "import#2 (reuse) samplings unchanged",
        d2["samplings_total"] == 0,
        f"delta samplings_total={d2['samplings_total']} ({counts1['samplings_total']} -> {counts2['samplings_total']})",
    )
    check(
        "import#2 (reuse) laps unchanged",
        d2["laps_total"] == 0,
        f"delta laps_total={d2['laps_total']} ({counts1['laps_total']} -> {counts2['laps_total']})",
    )
    check(
        "import#2 (reuse) route points on source items unchanged",
        d2["route_points_on_source_items"] == 0,
        f"delta route_points_on_source_items={d2['route_points_on_source_items']} ({counts1['route_points_on_source_items']} -> {counts2['route_points_on_source_items']})",
    )
    check(
        "import#2 creates zero NEW standalone gpx activities (delta vs #1)",
        new_gpx_after_2 == new_gpx_after_1,
        f"new_gpx_after_1={new_gpx_after_1} new_gpx_after_2={new_gpx_after_2}",
    )

    print(f"\nlaps_total after import#1 = {counts1['laps_total']} (informational; not hard-asserted)")

    print("\n=== summary ===")
    if failures:
        print(f"FAILED: {len(failures)} assertion(s) failed: {failures}")
        return 1
    print("ALL ASSERTIONS PASSED")
    return 0


def run_assert_reprocess_mode(args: argparse.Namespace) -> int:
    """Exercise the reprocess (purge-then-repersist) path.

    This mode does NOT change the server's parser_version - it can't, that's
    a server-startup config. It assumes the caller has already restarted the
    server with a parser_version different from the last completed import
    of this file's sha256, so the import job created here lands on
    dispositionReprocess. If the server is instead still on the SAME
    parser_version as the last completed import, this will hit
    dispositionSkip and the "unchanged" assertions will trivially pass
    without exercising the purge at all - see the printed disposition hint
    below if that looks likely.
    """
    print("\n=== baseline (before reprocess import) ===")
    baseline = capture_counts(args.dsn)
    print_counts(baseline)

    if baseline["apple_health_activities_without_source_item"] != 0:
        print(
            "[WARN] baseline already has apple_health activities without a "
            "source_item - prior state is already inconsistent; results below "
            "may not isolate this run's behavior"
        )

    print("\n=== reprocess import (parser_version expected to differ from last completed import) ===")
    raw = upload_raw_file(args)
    print(f"raw_file_id={raw['id']} duplicate={raw.get('duplicate', False)}")
    job = create_import(args, raw["id"])
    print(f"import_id={job['id']}")
    final = wait_import(args, job["id"])
    print(f"import_status={final['status']}")
    if final.get("error_message"):
        print(f"import_error={final['error_message']}")
        return 1

    after = capture_counts(args.dsn)
    print_counts(after)

    d = delta(baseline, after)
    print(f"delta={d}")

    print("\n=== assertions ===")
    failures = []

    def check(label: str, condition: bool, detail: str) -> None:
        status = "PASS" if condition else "FAIL"
        print(f"[{status}] {label}: {detail}")
        if not condition:
            failures.append(label)

    check(
        "reprocess: activities_total unchanged (replaced, not appended)",
        d["activities_total"] == 0,
        f"delta activities_total={d['activities_total']} ({baseline['activities_total']} -> {after['activities_total']})",
    )
    check(
        "reprocess: apple_health_activities_total unchanged",
        d["apple_health_activities_total"] == 0,
        f"delta apple_health_activities_total={d['apple_health_activities_total']} "
        f"({baseline['apple_health_activities_total']} -> {after['apple_health_activities_total']})",
    )
    check(
        "reprocess: apple_source_items total unchanged",
        d["source_items_total"] == 0,
        f"delta source_items_total={d['source_items_total']} ({baseline['source_items_total']} -> {after['source_items_total']})",
    )
    check(
        "reprocess: import_snapshots unchanged in total (old purged, one new persisted)",
        d["snapshots"] == 0,
        f"delta snapshots={d['snapshots']} ({baseline['snapshots']} -> {after['snapshots']})",
    )
    check(
        "reprocess: samplings unchanged in total",
        d["samplings_total"] == 0,
        f"delta samplings_total={d['samplings_total']} ({baseline['samplings_total']} -> {after['samplings_total']})",
    )
    check(
        "reprocess: laps unchanged in total",
        d["laps_total"] == 0,
        f"delta laps_total={d['laps_total']} ({baseline['laps_total']} -> {after['laps_total']})",
    )
    check(
        "reprocess: route points on source items unchanged in total",
        d["route_points_on_source_items"] == 0,
        f"delta route_points_on_source_items={d['route_points_on_source_items']} "
        f"({baseline['route_points_on_source_items']} -> {after['route_points_on_source_items']})",
    )
    check(
        "reprocess: no apple_health activities left without a source_item (purge order sound)",
        after["apple_health_activities_without_source_item"] == 0,
        f"after.apple_health_activities_without_source_item={after['apple_health_activities_without_source_item']}",
    )

    print("\n=== summary ===")
    if failures:
        print(f"FAILED: {len(failures)} assertion(s) failed: {failures}")
        return 1
    print("ALL ASSERTIONS PASSED")
    return 0


def capture_counts(dsn: str) -> dict:
    return {
        "activities_total": psql_int(dsn, "select count(*) from tb_activities;"),
        "apple_health_activities_total": psql_int(
            dsn,
            """
            select count(*)
            from tb_activities a
            join tb_external_refs er on er.activity_id = a.id
            where er.provider = 'apple_health';
            """,
        ),
        "apple_health_activities_without_source_item": psql_int(
            dsn,
            """
            select count(*)
            from tb_activities a
            join tb_external_refs er on er.activity_id = a.id
            where er.provider = 'apple_health'
              and not exists (
                select 1 from tb_apple_source_items si where si.activity_id = a.id
              );
            """,
        ),
        "source_items_total": psql_int(dsn, "select count(*) from tb_apple_source_items;"),
        "source_items_workout": psql_int(
            dsn, "select count(*) from tb_apple_source_items where item_type = 'workout';"
        ),
        "source_items_by_type": psql_rows(
            dsn,
            "select item_type, count(*) from tb_apple_source_items group by item_type order by item_type;",
        ),
        "snapshots": psql_int(dsn, "select count(*) from tb_import_snapshots;"),
        "samplings_total": psql_int(dsn, "select count(*) from tb_activity_samplings;"),
        "sampling_types": {
            row[0] for row in psql_rows(dsn, "select distinct sampling_type from tb_activity_samplings;")
        },
        "laps_total": psql_int(dsn, "select count(*) from tb_activity_laps;"),
        "route_points_on_source_items": psql_int(
            dsn,
            """
            select count(*)
            from tb_activity_route_points rp
            join tb_apple_source_items si on si.activity_id = rp.activity_id;
            """,
        ),
    }


def new_gpx_activity_count(dsn: str, since_iso: str) -> int:
    return psql_int(
        dsn,
        f"""
        select count(*)
        from tb_activities a
        join tb_external_refs er on er.activity_id = a.id
        where er.provider = 'gpx' and a.created_at >= '{since_iso}'::timestamptz;
        """,
    )


def delta(before: dict, after: dict) -> dict:
    out = {}
    for key, value in after.items():
        if isinstance(value, (int, float)) and isinstance(before.get(key), (int, float)):
            out[key] = value - before[key]
    return out


def print_counts(counts: dict) -> None:
    for key, value in counts.items():
        print(f"  {key}={value}")


def psql_scalar(dsn: str, sql: str) -> str:
    result = subprocess.run(
        ["psql", dsn, "-tA", "-c", sql],
        check=True,
        capture_output=True,
        text=True,
    )
    return result.stdout.strip()


def psql_int(dsn: str, sql: str) -> int:
    value = psql_scalar(dsn, sql)
    return int(value) if value else 0


def psql_rows(dsn: str, sql: str) -> list[tuple[str, ...]]:
    result = subprocess.run(
        ["psql", dsn, "-tA", "-F", "|", "-c", sql],
        check=True,
        capture_output=True,
        text=True,
    )
    rows = []
    for line in result.stdout.splitlines():
        line = line.strip()
        if not line:
            continue
        rows.append(tuple(line.split("|")))
    return rows


# --- shared HTTP helpers -----------------------------------------------------


def upload_raw_file(args: argparse.Namespace) -> dict:
    boundary = f"----iroha-{uuid.uuid4().hex}"
    content_type = mimetypes.guess_type(args.file.name)[0] or "application/octet-stream"
    parts = [
        form_field(boundary, "source_kind", args.source_kind),
        form_field(boundary, "uploaded_via", args.uploaded_via),
        file_field(boundary, "file", args.file.name, content_type, args.file.read_bytes()),
        f"--{boundary}--\r\n".encode(),
    ]
    body = b"".join(parts)
    return post(
        args.api_base,
        "/api/v1/raw-files/",
        body,
        {"Content-Type": f"multipart/form-data; boundary={boundary}"},
        timeout=max(args.timeout_s, 120),
    )


def create_import(args: argparse.Namespace, raw_file_id: str) -> dict:
    body = json.dumps({"raw_file_id": raw_file_id, "parser_kind": args.parser_kind}).encode()
    return post(args.api_base, "/api/v1/imports", body, {"Content-Type": "application/json"}, timeout=30)


def wait_import(args: argparse.Namespace, import_id: str) -> dict:
    deadline = time.monotonic() + args.timeout_s
    while time.monotonic() < deadline:
        current = get_json(args.api_base, f"/api/v1/imports/{import_id}")
        if current["status"] in {"completed", "failed"}:
            return current
        time.sleep(1)
    raise TimeoutError(f"import {import_id} did not finish within {args.timeout_s}s")


def form_field(boundary: str, name: str, value: str) -> bytes:
    return f'--{boundary}\r\nContent-Disposition: form-data; name="{name}"\r\n\r\n{value}\r\n'.encode()


def file_field(boundary: str, name: str, filename: str, content_type: str, content: bytes) -> bytes:
    header = (
        f'--{boundary}\r\nContent-Disposition: form-data; name="{name}"; filename="{filename}"\r\n'
        f"Content-Type: {content_type}\r\n\r\n"
    ).encode()
    return header + content + b"\r\n"


def post(api_base: str, path: str, body: bytes, headers: dict[str, str], timeout: int) -> dict:
    request = Request(api_base.rstrip("/") + path, data=body, method="POST", headers=headers)
    return load_response(request, timeout)


def get_json(api_base: str, path: str) -> dict | list:
    request = Request(api_base.rstrip("/") + path)
    return load_response(request, 30)


def load_response(request: Request, timeout: int) -> dict | list:
    try:
        with urlopen(request, timeout=timeout) as response:
            return json.load(response)
    except HTTPError as error:
        detail = error.read().decode()
        raise RuntimeError(f"HTTP {error.code}: {detail}") from error


if __name__ == "__main__":
    raise SystemExit(main())
