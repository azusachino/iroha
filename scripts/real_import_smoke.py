#!/usr/bin/env python3
import argparse
import json
import mimetypes
import time
import uuid
from pathlib import Path
from urllib.error import HTTPError
from urllib.request import Request, urlopen


def main() -> int:
    parser = argparse.ArgumentParser(description="Smoke-test a real import through the iroha HTTP API.")
    parser.add_argument("file", type=Path)
    parser.add_argument("--api-base", default="http://127.0.0.1:8080")
    parser.add_argument("--source-kind", default="apple_health_export")
    parser.add_argument("--parser-kind", default="apple_health_export")
    parser.add_argument("--uploaded-via", default="telegram")
    parser.add_argument("--timeout-s", type=int, default=360)
    args = parser.parse_args()

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

    activities = get_json(args.api_base, "/api/v1/activities?limit=5")
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
