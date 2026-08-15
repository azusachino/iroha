#!/usr/bin/env python3
"""Verify the deployed k3s report cache with read-only HTTP requests."""

import argparse
import os
from typing import Any
from urllib.parse import urlencode

import requests

DEFAULT_API_BASE = "https://iroha.h.azusachino.icu"
DEFAULT_MONTH = "2099-01"
REPORT_PATH = "/api/v1/reports/monthly"
CACHE_HEADER = "X-Iroha-Cache"
REPORT_SCHEMA = "monthly-report.v1"


def report_url(api_base: str, month: str, timezone: str | None = None) -> str:
    query: dict[str, str] = {"month": month}
    if timezone:
        query["timezone"] = timezone
    return f"{api_base.rstrip('/')}{REPORT_PATH}?{urlencode(query)}"


def fetch_report(session: Any, url: str, timeout_s: float) -> requests.Response:
    response = session.get(url, headers={"Accept": "application/json"}, timeout=timeout_s)
    if response.status_code != 200:
        detail = response.text[:200].replace("\n", " ")
        raise RuntimeError(f"{url} returned HTTP {response.status_code}: {detail}")

    try:
        payload = response.json()
    except ValueError as error:
        raise RuntimeError(f"{url} returned invalid JSON") from error
    if not isinstance(payload, dict) or payload.get("schema") != REPORT_SCHEMA:
        raise RuntimeError(f"{url} returned an unexpected report schema")
    return response


def verify_cache(
    api_base: str,
    month: str,
    timezone: str | None = None,
    timeout_s: float = 10,
    session: Any | None = None,
) -> dict[str, str]:
    request_url = report_url(api_base, month, timezone)
    client = session or requests.Session()
    first = fetch_report(client, request_url, timeout_s)
    second = fetch_report(client, request_url, timeout_s)
    first_state = first.headers.get(CACHE_HEADER, "")
    second_state = second.headers.get(CACHE_HEADER, "")

    if first_state not in {"MISS", "HIT"}:
        raise RuntimeError(f"first cache state = {first_state!r}, want MISS or HIT")
    if second_state != "HIT":
        raise RuntimeError(f"second cache state = {second_state!r}, want HIT")
    if first.content != second.content:
        raise RuntimeError("cached report body differs from the first response")

    result = {"url": request_url, "first": first_state, "second": second_state}
    print(f"k3s cache smoke passed: {first_state} -> {second_state} {REPORT_PATH}")
    return result


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--api-base",
        default=os.environ.get("IROHA_K3S_API_BASE", DEFAULT_API_BASE),
        help="private Iroha base URL",
    )
    parser.add_argument(
        "--month",
        default=os.environ.get("IROHA_CACHE_SMOKE_MONTH", DEFAULT_MONTH),
        help="report month used for the read-only probe",
    )
    parser.add_argument("--timezone", help="optional explicit IANA timezone for the probe")
    parser.add_argument("--timeout-s", type=float, default=10)
    args = parser.parse_args()

    try:
        verify_cache(args.api_base, args.month, args.timezone, args.timeout_s)
    except (requests.RequestException, RuntimeError) as error:
        print(f"k3s cache smoke failed: {error}")
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
