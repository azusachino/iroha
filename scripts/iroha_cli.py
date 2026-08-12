#!/usr/bin/env python3
"""Thin command-line client for the private Iroha API."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import sys
from pathlib import Path
from typing import BinaryIO, Callable
from urllib.parse import urlencode

import requests

DEFAULT_API_BASE = "http://127.0.0.1:8080"
DEFAULT_TIMEOUT_S = 30


class CLIError(Exception):
    """An expected user-facing CLI failure."""


class APIError(CLIError):
    """A non-2xx response from the Iroha API."""

    def __init__(self, status: int, code: str, message: str, request_id: str = "") -> None:
        self.status = status
        self.code = code
        self.message = message
        self.request_id = request_id
        detail = f"{code}: {message}"
        if request_id:
            detail += f" (request_id={request_id})"
        super().__init__(detail)


class TransportError(CLIError):
    """A failure before an API response was received."""


def api_base_from_environment() -> str:
    return os.environ.get("IROHA_API_BASE", DEFAULT_API_BASE).rstrip("/")


def _api_error(status: int, body: bytes) -> APIError:
    try:
        value = json.loads(body)
    except (UnicodeDecodeError, json.JSONDecodeError):
        return APIError(status, "http_error", f"Iroha returned HTTP {status}")
    if not isinstance(value, dict):
        return APIError(status, "http_error", f"Iroha returned HTTP {status}")
    return APIError(
        status,
        str(value.get("code") or "http_error"),
        str(value.get("message") or f"Iroha returned HTTP {status}"),
        str(value.get("request_id") or ""),
    )


class IrohaClient:
    """Reusable, deliberately small HTTP client for Iroha's private API."""

    def __init__(
        self,
        base_url: str | None = None,
        timeout_s: float = DEFAULT_TIMEOUT_S,
        session: requests.Session | None = None,
    ) -> None:
        self.base_url = (base_url or api_base_from_environment()).rstrip("/")
        self.timeout_s = timeout_s
        self.session = session or requests.Session()

    def request(self, method: str, path: str, body: bytes | None = None) -> bytes:
        if not path.startswith("/"):
            raise CLIError(f"API path must start with '/': {path}")
        headers = {"Accept": "application/json"}
        if body is not None:
            headers["Content-Type"] = "application/json"
        try:
            response = self.session.request(
                method,
                self.base_url + path,
                data=body,
                headers=headers,
                timeout=self.timeout_s,
            )
        except requests.RequestException as error:
            raise TransportError(f"Iroha API request failed: {error}") from error
        status = response.status_code
        response_body = response.content
        if status < 200 or status >= 300:
            raise _api_error(status, response_body)
        return response_body


def read_json_input(path: str) -> bytes:
    """Read JSON bytes from a file or stdin, preserving the original bytes."""
    if path == "-":
        data = sys.stdin.buffer.read()
    else:
        try:
            data = Path(path).read_bytes()
        except OSError as error:
            raise CLIError(f"cannot read JSON input {path}: {error}") from error
    if not data.strip():
        raise CLIError(f"JSON input {path} is empty")
    try:
        json.loads(data)
    except (UnicodeDecodeError, json.JSONDecodeError) as error:
        raise CLIError(f"JSON input {path} is invalid: {error}") from error
    return data


def write_response(data: bytes, stream: BinaryIO | None = None) -> None:
    """Write API response bytes without parsing or reshaping JSON."""
    output = stream or sys.stdout.buffer
    output.write(data)
    if not data.endswith(b"\n"):
        output.write(b"\n")


def _write_json_file(path: str, value: object) -> None:
    try:
        Path(path).write_text(json.dumps(value, ensure_ascii=False, indent=2) + "\n")
    except OSError as error:
        raise CLIError(f"cannot update JSON input {path}: {error}") from error


def prepare_expense_create_input(path: str, source_ref: str | None = None) -> bytes:
    """Add and persist the stable source identity required by create."""
    original = read_json_input(path)
    try:
        value = json.loads(original)
    except (UnicodeDecodeError, json.JSONDecodeError) as error:
        raise CLIError(f"JSON input {path} is invalid: {error}") from error
    if not isinstance(value, dict):
        raise CLIError(f"JSON input {path} must contain an object")

    source = value.get("source")
    if isinstance(source, dict):
        source = dict(source)
    elif source is None:
        source = {"kind": "cli"}
    else:
        # Leave malformed source values for the canonical API to reject.
        return original
    source.setdefault("kind", "cli")
    if source_ref:
        source["ref"] = source_ref
    elif not source.get("ref"):
        source["ref"] = hashlib.sha256(original).hexdigest()
    value["source"] = source
    updated = json.dumps(value, ensure_ascii=False, indent=2).encode() + b"\n"
    if path != "-":
        _write_json_file(path, value)
    return updated


def _json_value(data: bytes) -> object:
    try:
        return json.loads(data)
    except (UnicodeDecodeError, json.JSONDecodeError) as error:
        raise CLIError(f"Iroha returned invalid JSON: {error}") from error


def _table(rows: list[list[object]], headers: list[str]) -> str:
    values = [[str(value) if value is not None else "" for value in row] for row in rows]
    widths = [len(header) for header in headers]
    for row in values:
        for index, value in enumerate(row):
            widths[index] = max(widths[index], len(value))
    lines = ["  ".join(header.ljust(widths[index]) for index, header in enumerate(headers))]
    lines.append("  ".join("-" * width for width in widths))
    lines.extend("  ".join(value.ljust(widths[index]) for index, value in enumerate(row)) for row in values)
    return "\n".join(lines) + "\n"


def expense_table(value: object) -> str:
    if isinstance(value, dict) and "items" in value:
        items = value["items"]
        rows = [
            [item.get("id"), item.get("occurred_on"), item.get("currency"), item.get("amount_minor"), item.get("category"), item.get("merchant", "")]
            for item in items
        ]
        return _table(rows, ["id", "occurred_on", "currency", "amount_minor", "category", "merchant"])
    if isinstance(value, dict):
        return _table(
            [[value.get("id"), value.get("occurred_on"), value.get("currency"), value.get("amount_minor"), value.get("category"), value.get("merchant", "")]],
            ["id", "occurred_on", "currency", "amount_minor", "category", "merchant"],
        )
    raise CLIError("Iroha returned an unexpected expense response")


def monthly_report_table(value: object) -> str:
    if not isinstance(value, dict) or not isinstance(value.get("period"), dict) or not isinstance(value.get("sections"), dict):
        raise CLIError("Iroha returned an unexpected monthly report response")
    period = value["period"]
    rows = []
    for name, section in value["sections"].items():
        if not isinstance(section, dict):
            raise CLIError(f"Iroha returned an invalid {name} report section")
        data = section.get("data")
        if section.get("state") == "empty" or data is None:
            summary = ""
        elif name == "movement":
            summary = f"activities={data.get('activity_count', '')} distance_m={data.get('distance_m', '')}"
        elif name == "sleep":
            summary = f"sessions={data.get('session_count', '')} main={data.get('main_sleep_count', '')} naps={data.get('nap_count', '')}"
        elif name == "daily_health":
            summary = f"observed_days={data.get('observed_days', '')} metrics={len(data.get('metric_averages', []))}"
        elif name == "media":
            summary = f"events={data.get('event_count', '')} completed={data.get('completed_count', '')}"
        elif name == "expenses":
            summary = f"expenses={data.get('expense_count', '')} currencies={len(data.get('totals_by_currency', []))}"
        else:
            summary = ""
        rows.append([name, section.get("state", ""), summary])
    return f"period: {period.get('month', '')} ({period.get('timezone', '')})\n" + _table(rows, ["section", "state", "summary"])


def metric_table(value: object) -> str:
    if not isinstance(value, dict):
        raise CLIError("Iroha returned an unexpected metric response")
    metrics = value.get("metrics")
    if metrics is None:
        metrics = [value.get("metric", {})]
    rows = [[item.get("id"), item.get("label"), item.get("unit"), item.get("preferred_view")] for item in metrics if isinstance(item, dict)]
    return _table(rows, ["id", "label", "unit", "view"])


def metric_series_table(value: object) -> str:
    if not isinstance(value, dict) or not isinstance(value.get("period"), dict):
        raise CLIError("Iroha returned an unexpected metric series response")
    rows = []
    for series in value.get("series", []):
        if not isinstance(series, dict):
            continue
        dimensions = ",".join(f"{key}={item}" for key, item in sorted((series.get("dimensions") or {}).items()))
        for point in series.get("points", []):
            if isinstance(point, dict):
                rows.append([dimensions, point.get("period"), point.get("value_minor", point.get("value")), point.get("observed_days")])
    return f"metric: {value.get('metric_id', '')} ({value.get('period', {}).get('timezone', '')})\n" + _table(rows, ["dimensions", "period", "value", "observed_days"])


def output_result(data: bytes, output_format: str, table_formatter: Callable[[object], str] | None = None) -> None:
    if not data:
        return
    if output_format == "json":
        write_response(data)
        return
    if table_formatter is None:
        raise CLIError("table output is not supported for this command")
    print(table_formatter(_json_value(data)), end="")


def run_expense_command(args: argparse.Namespace, client: IrohaClient) -> int:
    if args.expense_action == "create":
        body = prepare_expense_create_input(args.input, args.source_ref)
        output_result(client.request("POST", "/api/v1/expenses", body), args.format, expense_table)
        return 0
    if args.expense_action == "list":
        path = "/api/v1/expenses"
        filters = {
            "from": args.from_date,
            "to": args.to,
            "currency": args.currency,
            "category": args.category,
            "limit": args.limit,
            "cursor": args.cursor,
        }
        if any(value is not None for value in filters.values()):
            path += "?" + urlencode({key: value for key, value in filters.items() if value is not None})
        output_result(client.request("GET", path), args.format, expense_table)
        return 0
    if args.expense_action == "get":
        output_result(client.request("GET", f"/api/v1/expenses/{args.expense_id}"), args.format, expense_table)
        return 0
    if args.expense_action == "update":
        body = read_json_input(args.input)
        output_result(client.request("PUT", f"/api/v1/expenses/{args.expense_id}", body), args.format, expense_table)
        return 0
    if args.expense_action == "delete":
        client.request("DELETE", f"/api/v1/expenses/{args.expense_id}")
        return 0
    raise CLIError(f"unsupported expense command: {args.expense_action}")


def run_report_command(args: argparse.Namespace, client: IrohaClient) -> int:
    timezone = args.timezone or os.environ.get("IROHA_TIMEZONE")
    if not timezone:
        raise CLIError("report timezone is required; pass --timezone or set IROHA_TIMEZONE")
    path = "/api/v1/reports/monthly?" + urlencode({"month": args.month, "timezone": timezone})
    output_result(client.request("GET", path), args.format, monthly_report_table)
    return 0


def run_metric_command(args: argparse.Namespace, client: IrohaClient) -> int:
    if args.metric_action == "list":
        output_result(client.request("GET", "/api/v1/metrics"), args.format, metric_table)
        return 0
    if args.metric_action == "get":
        output_result(client.request("GET", f"/api/v1/metrics/{args.metric_id}"), args.format, metric_table)
        return 0
    if args.metric_action == "series":
        params = [("from", args.from_date), ("to", args.to), ("grain", args.grain), ("timezone", args.timezone)]
        params.extend(("dimension", dimension) for dimension in args.dimension or [])
        path = f"/api/v1/metrics/{args.metric_id}/series?{urlencode(params)}"
        output_result(client.request("GET", path), args.format, metric_series_table)
        return 0
    raise CLIError(f"unsupported metric command: {args.metric_action}")


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="iroha_cli.py", description=__doc__)
    parser.add_argument(
        "--api-base",
        default=None,
        help="Iroha API base URL (defaults to IROHA_API_BASE or the local server)",
    )
    resources = parser.add_subparsers(dest="resource", required=True)
    expense = resources.add_parser("expense", help="manage canonical expenses")
    expense_commands = expense.add_subparsers(dest="expense_action", required=True)

    create = expense_commands.add_parser("create")
    create.add_argument("--input", required=True, help="JSON draft path, or '-' for stdin")
    create.add_argument("--source-ref", help="explicit stable source reference")
    create.add_argument("--format", choices=["json", "table"], default="json")

    list_command = expense_commands.add_parser("list")
    list_command.add_argument("--from", dest="from_date")
    list_command.add_argument("--to")
    list_command.add_argument("--currency")
    list_command.add_argument("--category")
    list_command.add_argument("--limit", type=int)
    list_command.add_argument("--cursor")
    list_command.add_argument("--format", choices=["json", "table"], default="json")

    get = expense_commands.add_parser("get")
    get.add_argument("expense_id")
    get.add_argument("--format", choices=["json", "table"], default="json")

    update = expense_commands.add_parser("update")
    update.add_argument("expense_id")
    update.add_argument("--input", required=True, help="JSON replacement path, or '-' for stdin")
    update.add_argument("--format", choices=["json", "table"], default="json")

    delete = expense_commands.add_parser("delete")
    delete.add_argument("expense_id")
    delete.add_argument("--format", choices=["json", "table"], default="json")

    report = resources.add_parser("report", help="read reports")
    report_commands = report.add_subparsers(dest="report_action", required=True)
    monthly = report_commands.add_parser("monthly")
    monthly.add_argument("--month", required=True)
    monthly.add_argument("--timezone")
    monthly.add_argument("--format", choices=["json", "table"], default="json")

    metric = resources.add_parser("metric", help="read the metric catalog and server-aggregated series")
    metric_commands = metric.add_subparsers(dest="metric_action", required=True)
    metric_list = metric_commands.add_parser("list")
    metric_list.add_argument("--format", choices=["json", "table"], default="json")
    metric_get = metric_commands.add_parser("get")
    metric_get.add_argument("metric_id")
    metric_get.add_argument("--format", choices=["json", "table"], default="json")
    metric_series = metric_commands.add_parser("series")
    metric_series.add_argument("metric_id")
    metric_series.add_argument("--from", dest="from_date", required=True)
    metric_series.add_argument("--to", required=True)
    metric_series.add_argument("--grain", choices=["day", "month", "year"], required=True)
    metric_series.add_argument("--timezone", default="UTC")
    metric_series.add_argument("--dimension", action="append", help="repeatable name:value dimension filter")
    metric_series.add_argument("--format", choices=["json", "table"], default="json")
    return parser


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    client = IrohaClient(args.api_base)
    if args.resource == "expense":
        return run_expense_command(args, client)
    if args.resource == "report" and args.report_action == "monthly":
        return run_report_command(args, client)
    if args.resource == "metric":
        return run_metric_command(args, client)
    raise CLIError(f"unsupported resource: {args.resource}")


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except CLIError as error:
        print(f"iroha-cli: {error}", file=sys.stderr)
        raise SystemExit(1) from error
