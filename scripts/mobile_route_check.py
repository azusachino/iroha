#!/usr/bin/env python3
"""Run the private cockpit route inventory at a compact browser viewport."""

from __future__ import annotations

import json
import os
import shutil
import subprocess
import urllib.request
from datetime import date
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
MOBILE_VIEWPORTS = ((320, 844), (375, 844), (390, 844), (414, 896))
THEMES = ("atlas", "grapher", "field-journal", "phenology", "sound-map", "archive")
MODES = ("light", "dark")
MOTION_MODES = ("normal", "reduced")

STATIC_ROUTES = (
    ("/", "/"),
    ("/overview", "/overview"),
    ("/motion", "/motion"),
    ("/night", "/night"),
    ("/library", "/library"),
    ("/expenses?month=2026-08", "/expenses"),
    ("/patterns?month=2026-08", "/patterns"),
    ("/reports?month=2026-08", "/reports"),
    ("/metrics?metric=health.steps&month=2026-08", "/metrics"),
    ("/admin", "/admin"),
    ("/manual", "/manual"),
    ("/design", "/design"),
    ("/to-go", "/to-go"),
    ("/activities", "/motion"),
    ("/daily", "/patterns"),
    ("/dashboard", "/overview"),
    ("/media", "/library"),
    ("/sleep", "/night"),
)


def route_inventory(
    activity_id: str, sleep_id: str, media_id: str
) -> list[tuple[str, str]]:
    """Return canonical and compatibility routes in deterministic order."""

    dynamic = (
        (f"/motion/{activity_id}", f"/motion/{activity_id}"),
        (f"/night/{sleep_id}", f"/night/{sleep_id}"),
        (f"/library/{media_id}", f"/library/{media_id}"),
        (f"/activities/{activity_id}", f"/motion/{activity_id}"),
        (f"/sleep/{sleep_id}", f"/night/{sleep_id}"),
        (f"/media/{media_id}", f"/library/{media_id}"),
    )
    return [*STATIC_ROUTES, *dynamic]


def get_json(url: str) -> dict:
    request = urllib.request.Request(url, headers={"Accept": "application/json"})
    with urllib.request.urlopen(request, timeout=15) as response:
        return json.load(response)


def first_id(api_base: str, path: str, label: str) -> str:
    payload = get_json(f"{api_base.rstrip('/')}{path}")
    items = payload.get("items") or []
    if not items or not items[0].get("id"):
        raise RuntimeError(f"cannot audit {label} detail route: API returned no records")
    return str(items[0]["id"])


def browser_command(session: str, *args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["agent-browser", "--session", session, *args],
        cwd=ROOT,
        text=True,
        capture_output=True,
        check=True,
    )


def eval_json(session: str, expression: str) -> dict:
    raw = browser_command(session, "eval", f"JSON.stringify({expression})").stdout.strip()
    value = json.loads(raw)
    return json.loads(value) if isinstance(value, str) else value


def parse_values(name: str, default: tuple[str, ...]) -> tuple[str, ...]:
    raw = os.environ.get(name)
    if not raw:
        return default
    values = tuple(item.strip() for item in raw.split(",") if item.strip())
    if not values:
        raise RuntimeError(f"{name} must contain at least one value")
    return values


def parse_viewports() -> tuple[tuple[int, int], ...]:
    raw = os.environ.get("VIEWPORTS")
    if not raw:
        return MOBILE_VIEWPORTS
    viewports = []
    for value in raw.split(","):
        width_height = value.strip().lower().split("x")
        if len(width_height) != 2 or not all(part.isdigit() for part in width_height):
            raise RuntimeError(f"VIEWPORTS must use WIDTHxHEIGHT values: {value}")
        width, height = (int(part) for part in width_height)
        if width <= 0 or height <= 0:
            raise RuntimeError(f"VIEWPORTS must use positive dimensions: {value}")
        viewports.append((width, height))
    if not viewports:
        raise RuntimeError("VIEWPORTS must contain at least one value")
    return tuple(viewports)


def expected_route_url(route: str, expected_path: str) -> str:
    """Return the URL contract after the route's canonicalization rules run."""

    if route in ("/night", "/sleep") and expected_path == "/night":
        return f"/night?year={date.today().year}"
    if route.partition("?")[0] in ("/expenses", "/patterns", "/reports", "/metrics"):
        return expected_path + ("?" + route.partition("?")[2] if "?" in route else "")
    return expected_path


def assert_route(
    session: str,
    base_url: str,
    route: str,
    expected_path: str,
    theme: str,
    mode: str,
    motion: str,
    viewport: tuple[int, int],
) -> dict:
    browser_command(session, "errors", "--clear")
    browser_command(session, "open", f"{base_url.rstrip('/')}{route}")
    browser_command(session, "wait", "1000")
    state = eval_json(
        session,
        "{"
        "url:location.pathname+location.search,"
        "language:document.documentElement.dataset.language,"
        "theme:document.documentElement.dataset.theme,"
        "width:innerWidth,height:innerHeight,"
        "overflow:document.documentElement.scrollWidth>document.documentElement.clientWidth+1||"
        "document.body.scrollWidth>document.body.clientWidth+1,"
        "headings:document.querySelectorAll('h1,h2').length,"
        "pending:document.querySelectorAll('[aria-busy=\\\"true\\\"],.skeleton').length,"
        "unnamed:[...document.querySelectorAll('a,button,input,select,textarea,summary')]"
        ".filter(el=>!el.getAttribute('aria-label')&&!el.getAttribute('title')&&"
        "!el.textContent.trim()&&!(el.labels&&el.labels.length)).length,"
        "charts:[...document.querySelectorAll('canvas')]"
        ".filter(canvas=>!canvas.closest('[role=img],[aria-label]')).length,"
        "maps:[...document.querySelectorAll('.map')]"
        ".filter(map=>!map.closest('.map-shell')||!map.closest('.map-shell').querySelector('details')).length,"
        "calendars:[...document.querySelectorAll('.pk-grid,.month-grid')]"
        ".filter(grid=>grid.getAttribute('role')!=='grid'||!grid.querySelector('[role=gridcell]')).length,"
        "reduced:matchMedia('(prefers-reduced-motion: reduce)').matches"
        "}",
    )
    expected_url = expected_route_url(route, expected_path)
    if state["url"] != expected_url:
        raise RuntimeError(f"route redirect mismatch for {route}: {state['url']} != {expected_url}")
    if state["language"] != theme or state["theme"] != mode:
        raise RuntimeError(f"theme state mismatch for {route}: {state}")
    if state["width"] != viewport[0] or state["height"] != viewport[1]:
        raise RuntimeError(f"viewport mismatch for {route}: {state}")
    if state["overflow"]:
        raise RuntimeError(f"horizontal overflow for {theme}/{mode}/{motion}/{route}: {state}")
    if state["headings"] < 1:
        raise RuntimeError(f"route has no heading for {route}: {state}")
    if state["pending"]:
        raise RuntimeError(f"route still has loading UI for {route}: {state}")
    if state["unnamed"]:
        raise RuntimeError(f"route has unnamed interactive controls for {route}: {state}")
    if state["charts"]:
        raise RuntimeError(f"chart has no accessible label for {route}: {state}")
    if state["maps"]:
        raise RuntimeError(f"map has no keyboard data alternative for {route}: {state}")
    if state["calendars"]:
        raise RuntimeError(f"calendar semantics incomplete for {route}: {state}")
    if state["reduced"] != (motion == "reduced"):
        raise RuntimeError(f"reduced-motion emulation mismatch for {route}: {state}")
    errors = browser_command(session, "errors").stdout.strip()
    if errors and "No page errors" not in errors:
        raise RuntimeError(f"browser errors for {route}: {errors}")
    return state


def main() -> int:
    if shutil.which("agent-browser") is None:
        raise RuntimeError("agent-browser is required; run make web-visual-install first")
    base_url = os.environ.get("BASE", "http://127.0.0.1:4173")
    api_base = os.environ.get("API_BASE", base_url)
    themes = parse_values("THEMES", THEMES)
    modes = parse_values("MODES", MODES)
    motions = parse_values("MOTION", MOTION_MODES)
    viewports = parse_viewports()
    activity_id = first_id(api_base, "/api/v1/activities?limit=1", "activity")
    sleep_id = first_id(api_base, "/api/v1/sleep?limit=1", "sleep")
    media_id = first_id(api_base, "/api/v1/media?limit=1", "library")
    routes = route_inventory(activity_id, sleep_id, media_id)
    session = f"iroha-mobile-{os.getpid()}"
    report: list[dict] = []
    try:
        browser_command(session, "open", base_url)
        for viewport in viewports:
            browser_command(session, "set", "viewport", str(viewport[0]), str(viewport[1]))
            for theme in themes:
                browser_command(session, "storage", "local", "set", "iroha-design-language", theme)
                for mode in modes:
                    browser_command(session, "storage", "local", "set", "iroha-theme", mode)
                    for motion in motions:
                        if motion == "reduced":
                            browser_command(session, "set", "media", mode, "reduced-motion")
                        else:
                            browser_command(session, "set", "media", mode)
                        for route, expected_path in routes:
                            state = assert_route(
                                session,
                                base_url,
                                route,
                                expected_path,
                                theme,
                                mode,
                                motion,
                                viewport,
                            )
                            report.append(
                                {
                                    "theme": theme,
                                    "mode": mode,
                                    "motion": motion,
                                    "viewport": viewport,
                                    "route": route,
                                    "state": state,
                                }
                            )
                            print(f"checked {viewport[0]}x{viewport[1]}/{theme}/{mode}/{motion}{route}", flush=True)
    finally:
        browser_command(session, "close")

    output = Path(os.environ.get("OUT", "dist/mobile-route-audit.json"))
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps({"viewports": viewports, "checks": report}, indent=2) + "\n")
    print(f"mobile route audit passed: {len(report)} checks; report={output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
