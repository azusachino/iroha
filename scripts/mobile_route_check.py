#!/usr/bin/env python3
"""Run the private cockpit route inventory at a compact browser viewport."""

from __future__ import annotations

import json
import os
import shutil
import subprocess
import urllib.request
import urllib.parse
from datetime import date
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
MOBILE_VIEWPORTS = ((320, 844), (375, 844), (390, 844), (414, 896))
THEMES = ("atlas", "grapher", "field-journal", "phenology", "sound-map", "archive")
MODES = ("light", "dark")
MOTION_MODES = ("normal", "reduced")
FOCUS_CONTRAST_EXPRESSION = (
    "(()=>{const probe=document.createElement('span');"
    "probe.style.cssText='position:fixed;visibility:hidden';document.body.append(probe);"
    "const resolve=token=>{probe.style.color=`var(${token})`;return getComputedStyle(probe).color;};"
    "const rgb=value=>(value.match(/[\\d.]+/g)??[]).slice(0,3).map(Number);"
    "const luminance=value=>{const [r,g,b]=rgb(value).map(channel=>{const scaled=channel/255;"
    "return scaled<=0.04045?scaled/12.92:((scaled+0.055)/1.055)**2.4;});"
    "return 0.2126*r+0.7152*g+0.0722*b;};const focus=luminance(resolve('--color-focus'));"
    "const ratios=['--bg','--surface','--surface-2'].map(token=>{const background=luminance(resolve(token));"
    "return (Math.max(focus,background)+0.05)/(Math.min(focus,background)+0.05);});"
    "probe.remove();return Math.min(...ratios);})()"
)

STATIC_ROUTES = (
    ("/", "/"),
    (f"/?date={date.today().isoformat()}", "/"),
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


def route_inventory(activity_id: str, sleep_id: str, media_id: str) -> list[tuple[str, str]]:
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


def launch_browser(
    session: str, reduced_motion: bool, *args: str
) -> subprocess.CompletedProcess[str]:
    command = ["agent-browser", "--session", session]
    if reduced_motion:
        command.extend(["--args", "--force-prefers-reduced-motion"])
    return subprocess.run(
        [*command, *args],
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

    if expected_path in ("/motion", "/night") and "?" not in route:
        return f"{expected_path}?date={date.today().year}"
    route_path, separator, query = route.partition("?")
    if route_path == "/metrics":
        params = [
            ("date" if key in ("month", "year") else key, value)
            for key, value in urllib.parse.parse_qsl(query, keep_blank_values=True)
        ]
        return expected_path + ("?" + urllib.parse.urlencode(params) if params else "")
    if route_path in ("/expenses", "/patterns", "/reports"):
        return expected_path + ("?" + route.partition("?")[2] if "?" in route else "")
    return expected_path


def report_route(route: str) -> str:
    """Redact record identifiers before persisting an audit report."""

    path, separator, query = route.partition("?")
    parts = path.strip("/").split("/")
    if len(parts) == 2 and parts[0] in {
        "activities",
        "library",
        "media",
        "motion",
        "night",
        "sleep",
    }:
        path = f"/{parts[0]}/:id"
    return path + (separator + query if separator else "")


def accessibility_failures(state: dict, viewport: tuple[int, int]) -> list[str]:
    """Return deterministic accessibility-contract failures for a route state."""

    failures = []
    skip_link = state["skipLink"]
    if not skip_link["exists"] or not skip_link["targetExists"]:
        failures.append("missing or invalid skip link")
    if state["mainCount"] != 1:
        failures.append("expected exactly one main landmark")
    if state["footerInMain"]:
        failures.append("theme footer is inside the main landmark")
    if state["h1Count"] != 1 or state["firstHeading"] != "H1":
        failures.append("H1 is not the single first heading")
    if viewport[0] <= 640 and state["focusOrderMismatch"]:
        failures.append("compact focus order differs from visual order")
    if state["smallTargetCount"]:
        failures.append(f"{state['smallTargetCount']} standalone controls are smaller than 24x24px")
    if state["focusContrast"] < 3:
        failures.append(
            f"focus indicator contrast is below 3:1: {state['focusContrast']:.2f}:1"
        )
    if state["mouseOnlyRows"]:
        failures.append(f"mouse-only clickable table rows: {state['mouseOnlyRows']}")
    if state["periodDrillLabelFailures"]:
        failures.append(
            f"period controls missing period/evidence labels: {state['periodDrillLabelFailures']}"
        )
    return failures


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
        "navigation:(()=>{const nav=document.querySelector('.main-nav');if(!nav)return null;"
        "const rect=nav.getBoundingClientRect();const items=[...nav.children];return {"
        "overflow:nav.scrollWidth>nav.clientWidth+1,"
        "clipped:items.some(el=>{const box=el.getBoundingClientRect();return box.left<rect.left-1||box.right>rect.right+1}),"
        "count:items.length};})(),"
        "headings:document.querySelectorAll('h1,h2').length,"
        "h1Count:document.querySelectorAll('h1').length,"
        "firstHeading:document.querySelector('h1,h2,h3,h4,h5,h6')?.tagName??null,"
        "skipLink:(()=>{const link=document.querySelector('a.skip-link[href^=\\\"#\\\"]');"
        "const target=link&&document.querySelector(link.getAttribute('href'));return {"
        "exists:Boolean(link),targetExists:Boolean(target)};})(),"
        "mainCount:document.querySelectorAll('main').length,"
        "footerInMain:Boolean(document.querySelector("
        "'main :is(.atlas-footer,.grapher-footer,.field-journal-footer,.bloom-footer,.mix-footer,.vault-footer,.footer)')),"
        "focusOrderMismatch:(()=>{const items=[...document.querySelectorAll("
        "'.appbar a[href],.appbar button:not([disabled]),.appbar summary,.appbar select')].filter(el=>"
        "el.getClientRects().length&&getComputedStyle(el).visibility!=='hidden'&&"
        "(!el.closest('details:not([open])')||el.matches('summary'))).map((el,index)=>{"
        "const box=el.getBoundingClientRect();return {index,top:box.top,left:box.left};});"
        "const visual=[...items].sort((a,b)=>Math.abs(a.top-b.top)>8?a.top-b.top:a.left-b.left);"
        "return visual.some((item,index)=>item.index!==items[index].index);})(),"
        "smallTargetCount:[...document.querySelectorAll("
        "'a[href],button,summary,input:not([type=hidden]),select,textarea')].filter(el=>{"
        "if(!el.getClientRects().length||getComputedStyle(el).visibility==='hidden')return false;"
        "if(el.closest('details:not([open])')&&!el.matches('summary'))return false;"
        "if(el.matches('a')&&el.closest('p')&&getComputedStyle(el).display==='inline')return false;"
        "const box=el.getBoundingClientRect();return box.width<24||box.height<24;}).length,"
        f"focusContrast:{FOCUS_CONTRAST_EXPRESSION},"
        "mouseOnlyRows:[...document.querySelectorAll('tbody tr')].filter(row=>"
        "getComputedStyle(row).cursor==='pointer'&&!row.querySelector("
        "'a[href],button,input,select,textarea,summary')&&!row.matches("
        "'[tabindex]:not([tabindex=\"-1\"])[role=\"link\"],"
        "[tabindex]:not([tabindex=\"-1\"])[role=\"button\"]')).length,"
        "periodDrillLabelFailures:[...document.querySelectorAll('[data-period-drill]')].filter(el=>{"
        "const name=el.getAttribute('aria-label')??'';const period=el.textContent.trim();"
        "const row=el.closest('tr');const table=el.closest('table');"
        "const headers=[...(table?.querySelectorAll('thead th')??[])];"
        "const stepsIndex=headers.findIndex(header=>header.textContent.trim().startsWith('Steps'));"
        "const cell=row?.querySelector('[data-period-evidence]')??"
        "(stepsIndex<0?null:row?.children[stepsIndex]);const value=cell?.textContent.trim()??'';"
        "const evidence=value==='—'?'no steps recorded':`${value} steps`;return !period||!value||"
        "!name.includes(period)||!name.includes(evidence);}).length,"
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
    if viewport[0] <= 640 and (
        not state["navigation"]
        or state["navigation"]["overflow"]
        or state["navigation"]["clipped"]
        or state["navigation"]["count"] != 5
    ):
        raise RuntimeError(f"mobile navigation does not fit for {route}: {state}")
    failures = accessibility_failures(state, viewport)
    if failures:
        raise RuntimeError(f"accessibility contract failed for {route}: {failures}; {state}")
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
    return {**state, "url": report_route(state["url"])}


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
    report: list[dict] = []
    for motion in motions:
        session = f"iroha-mobile-{os.getpid()}-{motion}"
        try:
            launch_browser(session, motion == "reduced", "open", base_url)
            for viewport in viewports:
                browser_command(session, "set", "viewport", str(viewport[0]), str(viewport[1]))
                for theme in themes:
                    browser_command(
                        session, "storage", "local", "set", "iroha-design-language", theme
                    )
                    for mode in modes:
                        browser_command(session, "storage", "local", "set", "iroha-theme", mode)
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
                                    "route": report_route(route),
                                    "state": state,
                                }
                            )
                            print(
                                f"checked {viewport[0]}x{viewport[1]}/{theme}/{mode}/{motion}{route}",
                                flush=True,
                            )
        finally:
            browser_command(session, "close")

    output = Path(os.environ.get("OUT", "dist/mobile-route-audit.json"))
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps({"viewports": viewports, "checks": report}, indent=2) + "\n")
    print(f"mobile route audit passed: {len(report)} checks; report={output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
