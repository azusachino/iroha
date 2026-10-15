"""Disposable API feasibility probe. Synthetic data only; not a companion implementation."""
import json
import os
from pathlib import Path
import secrets
import socket
import subprocess
import tempfile
import time
import urllib.error
import urllib.request
import urllib.parse
import zipfile

ROOT = Path(__file__).resolve().parent
RUN = Path(tempfile.mkdtemp(prefix="run-", dir=ROOT))
NET, DB, APP = "bo-companion-spike-net", "bo-companion-spike-db", "bo-companion-spike-app"
BASE = "http://127.0.0.1:18943"
APP_IMAGE, DB_IMAGE = "ghcr.io/bookorbit/bookorbit:3.3.0", "docker.io/pgvector/pgvector:pg18"
results = {"synthetic_only": True, "source_commit": "e767358443419253eeed782518d6fc4f675d69fd", "checks": []}
token = None
created = []
private = {"POSTGRES_PASSWORD": secrets.token_hex(24), "JWT_SECRET": secrets.token_hex(32), "SETUP_BOOTSTRAP_TOKEN": secrets.token_hex(24)}
password = "SpikeAa1" + secrets.token_hex(16)


def cmd(*args, env=None):
    p = subprocess.run(args, env=env, text=True, capture_output=True, timeout=180)
    if p.returncode:
        text = p.stderr
        for value in [*private.values(), password, token or "unused-token"]:
            text = text.replace(value, "[REDACTED]")
        raise RuntimeError(f"{args[0:2]} failed: {text[-1500:]}")
    return p.stdout


def call(method, path, body=None, headers=None):
    h = {"Content-Type": "application/json", **(headers or {})}
    if token:
        h["Authorization"] = "Bearer " + token
    req = urllib.request.Request(BASE + path, method=method, headers=h,
                                 data=None if body is None else json.dumps(body).encode())
    try:
        with urllib.request.urlopen(req, timeout=15) as r:
            data = r.read()
            return r.status, json.loads(data) if data else None
    except urllib.error.HTTPError as e:
        data = e.read()
        try:
            return e.code, json.loads(data)
        except json.JSONDecodeError:
            return e.code, {"non_json_error": True}


def check(name, passed, details):
    results["checks"].append({"name": name, "passed": bool(passed), "details": details})
    print(name, "PASS" if passed else "FAIL", flush=True)


def wait_for(fn, label, seconds=120):
    deadline = time.monotonic() + seconds
    while time.monotonic() < deadline:
        try:
            value = fn()
            if value:
                return value
        except (OSError, urllib.error.URLError, RuntimeError):
            pass
        time.sleep(1)
    raise RuntimeError("timeout: " + label)


def epub():
    folder = RUN / "books"
    folder.mkdir(exist_ok=True)
    with zipfile.ZipFile(folder / "companion-synthetic.epub", "w") as z:
        z.writestr("mimetype", "application/epub+zip", compress_type=zipfile.ZIP_STORED)
        z.writestr("META-INF/container.xml", '<?xml version="1.0"?><container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>')
        z.writestr("OEBPS/content.opf", '<?xml version="1.0"?><package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="id"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:identifier id="id">urn:uuid:synthetic-companion-spike</dc:identifier><dc:title>Companion Synthetic Fixture</dc:title><dc:creator>Synthetic Author</dc:creator><dc:language>en</dc:language><meta property="dcterms:modified">2026-10-12T00:00:00Z</meta></metadata><manifest><item id="chapter" href="chapter.xhtml" media-type="application/xhtml+xml"/><item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/></manifest><spine><itemref idref="chapter"/></spine></package>')
        z.writestr("OEBPS/chapter.xhtml", '<html xmlns="http://www.w3.org/1999/xhtml"><head><title>Synthetic</title></head><body><p id="p1">Entirely synthetic text authored for an isolated feasibility test.</p></body></html>')
        z.writestr("OEBPS/nav.xhtml", '<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops"><head><title>Contents</title></head><body><nav epub:type="toc"><ol><li><a href="chapter.xhtml">Synthetic chapter</a></li></ol></nav></body></html>')


try:
    for name in (APP, DB):
        if subprocess.run(["podman", "container", "exists", name], capture_output=True).returncode == 0:
            raise RuntimeError("refusing existing container: " + name)
    if subprocess.run(["podman", "network", "exists", NET], capture_output=True).returncode == 0:
        raise RuntimeError("refusing existing network")
    with socket.socket() as s:
        s.bind(("127.0.0.1", 18943))
    for key, image in (("app_image", APP_IMAGE), ("db_image", DB_IMAGE)):
        info = json.loads(cmd("podman", "image", "inspect", image))[0]
        results[key] = {"id": info["Id"], "digests": info.get("RepoDigests"), "labels": info.get("Labels")}
    epub()
    for p in (RUN / "app-data", RUN / "db-data"):
        p.mkdir(exist_ok=True)
    cmd("podman", "network", "create", "--internal", NET)
    created.append(NET)
    env = {**os.environ, **private, "POSTGRES_USER": "bookorbit", "POSTGRES_DB": "bookorbit"}
    cmd("podman", "run", "-d", "--name", DB, "--network", NET, "--network-alias", "postgres",
        "-e", "POSTGRES_USER", "-e", "POSTGRES_DB", "-e", "POSTGRES_PASSWORD",
        "-e", "PGDATA=/var/lib/postgresql/data/pgdata",
        "-v", f"{RUN / 'db-data'}:/var/lib/postgresql/data:Z",
        "-v", f"{ROOT / 'source/docker/postgres/init.sql'}:/docker-entrypoint-initdb.d/init.sql:ro,Z", DB_IMAGE, env=env)
    created.append(DB)
    wait_for(lambda: subprocess.run(["podman", "exec", DB, "pg_isready", "-U", "bookorbit"], capture_output=True).returncode == 0, "database")
    env.update({"NODE_ENV": "production", "PORT": "3000", "POSTGRES_HOST": "postgres", "POSTGRES_PORT": "5432", "APP_URL": BASE,
                "CLIENT_URL": BASE, "PUID": "0", "PGID": "0", "SWAGGER_ENABLED": "true", "TZ": "UTC", "BOOKS_PATH": "/books"})
    opts = ["podman", "run", "-d", "--name", APP, "--network", NET, "-p", "127.0.0.1:18943:3000"]
    for key in (*private, "POSTGRES_USER", "POSTGRES_DB", "NODE_ENV", "PORT", "POSTGRES_HOST", "POSTGRES_PORT", "APP_URL", "CLIENT_URL", "PUID", "PGID", "SWAGGER_ENABLED", "TZ", "BOOKS_PATH"):
        opts += ["-e", key]
    cmd(*opts, "-v", f"{RUN / 'books'}:/books:Z", "-v", f"{RUN / 'app-data'}:/data:Z", APP_IMAGE, env=env)
    created.append(APP)
    wait_for(lambda: call("GET", "/api/v1/auth/setup-status")[0] == 200, "application", 180)
    code, _ = call("POST", "/api/v1/auth/setup", {"username": "spike-admin", "name": "Synthetic Spike", "email": "spike@example.invalid", "password": password}, {"x-setup-token": private["SETUP_BOOTSTRAP_TOKEN"]})
    assert code in (200, 201), "setup failed"
    code, login = call("POST", "/api/v1/auth/login", {"username": "spike-admin", "password": password, "clientKind": "native", "deviceLabel": "Disposable companion spike"})
    assert code in (200, 201), "login failed"
    token = login["accessToken"]
    code, info = call("GET", "/api/v1/app-info")
    results["server_version"] = info
    assert code == 200, "app-info failed"
    code, schema = call("GET", "/api/docs-json")
    check("openapi_capture", code == 200 and "paths" in schema, {"http_status": code})
    if code == 200 and "paths" in schema:
        (ROOT / "openapi.json").write_text(json.dumps(schema, indent=2))
        results["book_route_methods"] = {p: list(v) for p, v in schema["paths"].items() if p.endswith("/books")}
    code, absent = call("POST", "/api/v1/books", {"title": "Synthetic metadata-only WeRead book", "authors": ["Synthetic Author"]})
    check("metadata_only_create_route_available", code in (200, 201), {"http_status": code, "response": absent})
    code, lib = call("POST", "/api/v1/libraries", {"name": "Synthetic Companion Spike", "icon": "book", "folders": ["/books"], "watch": False, "fileWriteEnabled": False, "fileRenameEnabled": False})
    assert code in (200, 201), f"library create failed: {code} {lib}"
    query = {"pagination": {"page": 0, "size": 50}, "sort": [], "collapseSeries": False}
    books = wait_for(lambda: call("POST", "/api/v1/books/query", query)[1].get("items"), "scan synthetic EPUB")
    assert len(books) == 1, "unexpected book count"
    bid = books[0]["id"]
    fid = next(f["id"] for f in books[0]["files"] if f["role"] == "primary")
    def detail():
        c, d = call("GET", f"/api/v1/books/{bid}")
        assert c == 200
        return d
    def progress():
        c, rows = call("GET", f"/api/v1/books/{bid}/progress")
        assert c == 200
        return next(r for r in rows if r["fileId"] == fid)
    code, _ = call("PATCH", f"/api/v1/books/{bid}/metadata", {"title": "Synthetic WeRead Book", "authors": ["Synthetic Author"], "isbn13": "9780000000002"})
    check("matched_file_metadata_write", code == 200 and detail()["title"] == "Synthetic WeRead Book", {"http_status": code})
    code, _ = call("PATCH", f"/api/v1/books/{bid}/status", {"status": "reading", "startedAt": "2024-01-02"})
    d = detail()
    check("status_write_with_source_date", code == 200 and d["readStatus"]["status"] == "reading", {"http_status": code, "readStatus": d["readStatus"]})
    code, _ = call("POST", f"/api/v1/books/files/{fid}/progress", {"percentage": 20})
    p = progress()
    check("matched_file_progress_write", code in (200, 201) and p["percentage"] == 20, {"http_status": code, "percentage": p["percentage"]})
    code, rejected = call("POST", f"/api/v1/books/files/{fid}/progress", {"percentage": 25, "sourceProvider": "weread", "sourceRecordId": "synthetic-wr-1", "sourceUpdatedAt": "2024-01-02T00:00:00Z"})
    check("progress_native_provenance_supported", code in (200, 201), {"http_status": code, "response": rejected})
    before = progress()
    time.sleep(0.1)
    call("POST", f"/api/v1/books/files/{fid}/progress", {"percentage": 20})
    after = progress()
    check("replay_preserves_progress_timestamp", before.get("updatedAt") == after.get("updatedAt"), {"before": before.get("updatedAt"), "after": after.get("updatedAt")})
    call("POST", f"/api/v1/books/files/{fid}/progress", {"percentage": 70})
    call("POST", f"/api/v1/books/files/{fid}/progress", {"percentage": 10})
    check("stale_companion_write_preserves_local_progress", progress()["percentage"] == 70, {"local_percent": 70, "late_incoming_percent": 10, "result_percent": progress()["percentage"]})
    code, ann = call("POST", f"/api/v1/books/{bid}/annotations", {"text": "Synthetic WeRead highlight", "note": "Synthetic note"})
    check("unlocated_weread_annotation_supported", code in (200, 201), {"http_status": code, "response": ann})
    code, field = call("POST", "/api/v1/custom-metadata/fields", {"label": "Synthetic source evidence", "type": "text", "enabledLibraryIds": [lib["id"]]})
    assert code in (200, 201), f"custom field failed: {code} {field}"
    evidence = json.dumps({"provider": "weread", "sourceBookId": "synthetic-wr-1", "sourceAccount": "synthetic-only", "sourceUpdatedAt": "2024-01-02T00:00:00Z", "progressPercent": 20})
    code, _ = call("PATCH", f"/api/v1/books/{bid}/metadata", {"customMetadata": [{"fieldId": field["id"], "value": evidence}]})
    d = detail()
    check("generic_custom_metadata_roundtrip", code == 200 and any(v.get("value") == evidence for v in d.get("customMetadata", [])), {"http_status": code, "field_id": field["id"]})
    call("PATCH", f"/api/v1/books/{bid}/metadata", {"title": "Updated title only"})
    check("custom_evidence_survives_unrelated_metadata_patch", any(v.get("value") == evidence for v in detail().get("customMetadata", [])), {})
    # Generic metadata is book-scoped, not a safe private per-reader evidence channel.
    admin_token = token
    code, viewer = call("POST", "/api/v1/users", {"username": "spike-viewer", "name": "Synthetic Viewer", "email": "viewer@example.invalid", "libraryIds": [lib["id"]]})
    assert code in (200, 201), "synthetic viewer create failed"
    reset_token = urllib.parse.parse_qs(urllib.parse.urlparse(viewer["resetUrl"]).query)["token"][0]
    code, _ = call("POST", "/api/v1/auth/reset-password", {"token": reset_token, "newPassword": password})
    assert code == 204, f"synthetic viewer password setup failed: {code}"
    code, viewer_login = call("POST", "/api/v1/auth/login", {"username": "spike-viewer", "password": password, "clientKind": "native"})
    assert code in (200, 201), "synthetic viewer login failed"
    token = viewer_login["accessToken"]
    viewer_detail = detail()
    source_visible = any(v.get("value") == evidence for v in viewer_detail.get("customMetadata", []))
    check("custom_evidence_private_to_source_user", not source_visible, {"visible_to_second_library_viewer": source_visible, "viewer_readStatus": viewer_detail.get("readStatus")})
    token = admin_token
    results["completed"] = True
except Exception as e:
    results["completed"] = False
    message = str(e)
    for value in [*private.values(), password, token or "unused-token"]:
        message = message.replace(value, "[REDACTED]")
    results["infrastructure_error"] = message
    if APP in created:
        logs = subprocess.run(["podman", "logs", APP], capture_output=True, text=True, timeout=30)
        text = logs.stdout + logs.stderr
        for value in [*private.values(), password, token or "unused-token"]:
            text = text.replace(value, "[REDACTED]")
        (ROOT / "failure-app-log.txt").write_text(text)
    print(message, flush=True)
finally:
    results["cleanup"] = []
    for name in reversed(created):
        kind = "network" if name == NET else "container"
        args = ["podman", "network", "rm", name] if kind == "network" else ["podman", "rm", "-f", name]
        p = subprocess.run(args, capture_output=True, text=True, timeout=60)
        results["cleanup"].append({"kind": kind, "name": name, "exit_code": p.returncode})
    assert RUN.is_relative_to(ROOT) and RUN.name.startswith("run-")
    p = subprocess.run(["podman", "unshare", "rm", "-rf", "--", str(RUN)], capture_output=True, timeout=60)
    results["cleanup"].append({"kind": "disposable_storage", "exit_code": p.returncode})
    (ROOT / "results.json").write_text(json.dumps(results, indent=2))
    print("Report:", ROOT / "results.json", flush=True)
    if not results.get("completed") or any(c["exit_code"] for c in results["cleanup"]):
        raise SystemExit(1)
