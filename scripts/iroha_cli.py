#!/usr/bin/env python3
"""Thin command-line transport for the private Iroha API.

Domain commands are added in later slices. This module owns only API URL
configuration, JSON input handling, successful-response passthrough, and
structured error reporting.
"""

from __future__ import annotations

import argparse
import json
import os
import sys
from pathlib import Path
from typing import BinaryIO, Callable
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

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
        opener: Callable[..., object] = urlopen,
    ) -> None:
        self.base_url = (base_url or api_base_from_environment()).rstrip("/")
        self.timeout_s = timeout_s
        self.opener = opener

    def request(self, method: str, path: str, body: bytes | None = None) -> bytes:
        if not path.startswith("/"):
            raise CLIError(f"API path must start with '/': {path}")
        headers = {"Accept": "application/json"}
        if body is not None:
            headers["Content-Type"] = "application/json"
        request = Request(self.base_url + path, data=body, headers=headers, method=method)
        try:
            with self.opener(request, timeout=self.timeout_s) as response:
                status = getattr(response, "status", response.getcode())
                response_body = response.read()
        except HTTPError as error:
            raise _api_error(error.code, error.read()) from error
        except (OSError, URLError, TimeoutError) as error:
            raise TransportError(f"Iroha API request failed: {error}") from error
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


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="iroha_cli.py", description=__doc__)
    parser.add_argument(
        "--api-base",
        default=None,
        help="Iroha API base URL (defaults to IROHA_API_BASE or the local server)",
    )
    return parser


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    parser.parse_args(argv)
    parser.print_help()
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except CLIError as error:
        print(f"iroha-cli: {error}", file=sys.stderr)
        raise SystemExit(1) from error
