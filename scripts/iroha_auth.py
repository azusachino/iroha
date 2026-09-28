"""Owner-session login for scripts that call Iroha's private API (ADR-0008).

Every private /api/v1 route needs the owner session cookie, and state-changing
requests also need the session's CSRF token. Scripts call `login()` once;
`install_urllib()` then makes every later `urllib.request.urlopen` in the
process carry both, and `requests_session()` does the same for `requests`.

Credentials come from IROHA_USERNAME and IROHA_PASSWORD. Only disposable
local stacks may create the owner (`allow_setup=True`); against a real
deployment a missing owner is an error, never an implicit setup.
"""

from __future__ import annotations

import http.cookies
import json
import os
import urllib.request
from dataclasses import dataclass

SESSION_COOKIE = "iroha_session"
CSRF_HEADER = "X-CSRF-Token"
USERNAME_ENV = "IROHA_USERNAME"
PASSWORD_ENV = "IROHA_PASSWORD"


@dataclass(frozen=True)
class OwnerSession:
    token: str
    csrf_token: str


def credentials_from_env() -> tuple[str, str]:
    username = os.environ.get(USERNAME_ENV, "")
    password = os.environ.get(PASSWORD_ENV, "")
    if not username or not password:
        raise RuntimeError(f"set {USERNAME_ENV} and {PASSWORD_ENV} to log in to Iroha")
    return username, password


def login(
    api_base: str,
    username: str | None = None,
    password: str | None = None,
    *,
    allow_setup: bool = False,
) -> OwnerSession:
    """Log in (or, on a disposable stack, create the owner) and return the session."""
    if username is None or password is None:
        username, password = credentials_from_env()
    base = api_base.rstrip("/")
    state = _json_request(base + "/api/v1/auth/session")[1]
    if state.get("setup_required"):
        if not allow_setup:
            raise RuntimeError("Iroha has no owner account yet; create it in the web app first")
        path = "/api/v1/auth/setup"
    else:
        path = "/api/v1/auth/login"
    headers, body = _json_request(base + path, {"username": username, "password": password})
    token = _session_cookie(headers)
    if not token or not body.get("csrf_token"):
        raise RuntimeError(f"{path} did not return a session")
    return OwnerSession(token=token, csrf_token=body["csrf_token"])


def install_urllib(session: OwnerSession) -> None:
    """Make every later urllib request carry the session cookie and CSRF token."""
    urllib.request.install_opener(urllib.request.build_opener(_SessionHandler(session)))


def requests_session(session: OwnerSession):
    """Return a requests.Session that carries the owner session."""
    import requests

    client = requests.Session()
    client.headers[CSRF_HEADER] = session.csrf_token
    client.headers["Cookie"] = f"{SESSION_COOKIE}={session.token}"
    return client


class _SessionHandler(urllib.request.BaseHandler):
    handler_order = 100

    def __init__(self, session: OwnerSession) -> None:
        self._session = session

    def _add(self, request: urllib.request.Request) -> urllib.request.Request:
        request.add_unredirected_header("Cookie", f"{SESSION_COOKIE}={self._session.token}")
        if request.get_method() not in ("GET", "HEAD"):
            request.add_unredirected_header(CSRF_HEADER, self._session.csrf_token)
        return request

    http_request = _add
    https_request = _add


def _json_request(url: str, payload: dict | None = None) -> tuple[object, dict]:
    data = None if payload is None else json.dumps(payload).encode()
    request = urllib.request.Request(
        url,
        data=data,
        method="GET" if data is None else "POST",
        headers={"Accept": "application/json", "Content-Type": "application/json"},
    )
    with urllib.request.urlopen(request, timeout=15) as response:
        return response.headers, json.loads(response.read() or b"{}")


def _session_cookie(headers) -> str:
    for value in headers.get_all("Set-Cookie") or []:
        cookie = http.cookies.SimpleCookie()
        cookie.load(value)
        if SESSION_COOKIE in cookie:
            return cookie[SESSION_COOKIE].value
    return ""
