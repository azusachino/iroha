#!/usr/bin/env python3
"""Response contract for an isolated Caddy shell/asset fixture, never a live host."""
import sys
import time
import urllib.error
import urllib.parse
import urllib.request


def check(base: str) -> None:
    url = urllib.parse.urlsplit(base)
    if url.scheme != "http" or url.hostname not in {"127.0.0.1", "localhost"}:
        raise ValueError("BASE must be an isolated localhost HTTP fixture")

    for attempt in range(30):
        try:
            with urllib.request.urlopen(base + "/", timeout=2):
                break
        except (urllib.error.URLError, TimeoutError):
            if attempt == 29:
                raise
            time.sleep(0.2)

    for path, status, body in [
        ("/", 200, b"shell"),
        ("/_app/immutable/test.js", 200, b"asset"),
        ("/api/v1/auth/session", 200, b"shell"),
        ("/public/v1/meta", 502, None),
    ]:
        try:
            response = urllib.request.urlopen(base + path, timeout=2)
        except urllib.error.HTTPError as error:
            response = error
        with response:
            assert response.status == status, (path, response.status)
            if body is not None:
                assert response.read() == body, path
            headers = response.headers
            assert headers["Strict-Transport-Security"] == "max-age=31536000", path
            assert headers["Permissions-Policy"] == (
                "camera=(), microphone=(), geolocation=(), payment=(), usb=()"
            ), path
            assert headers["X-Content-Type-Options"] == "nosniff", path
            assert headers["Content-Security-Policy"], path
            cache = headers.get("Cache-Control", "")
            if path.startswith("/_app/immutable/"):
                assert cache == "public, max-age=31536000, immutable", cache
            else:
                assert "immutable" not in cache, (path, cache)
    print("public-site response contract passed")


if __name__ == "__main__":
    if len(sys.argv) != 2:
        raise SystemExit("usage: public_site_response_check.py http://127.0.0.1:port")
    check(sys.argv[1].rstrip("/"))
