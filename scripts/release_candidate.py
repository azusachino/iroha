#!/usr/bin/env python3
import json
import os
import shutil
import socket
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid
from pathlib import Path

from mobile_route_check import FOCUS_CONTRAST_EXPRESSION

ROOT = Path(__file__).resolve().parents[1]
SERVER_DIR = ROOT / "apps" / "iroha-server"
WEB_DIR = ROOT / "apps" / "iroha-web"
SEED_FILE = ROOT / "scripts" / "release_candidate_seed.sql"
PERFORMANCE_SEED_FILE = ROOT / "scripts" / "release_candidate_performance_seed.sql"
RESET_FILE = ROOT / "scripts" / "release_candidate_reset.sql"
POSTGIS_IMAGE = "docker.io/kartoza/postgis:18.4-3.6.4--v2026.06.21"
DATABASE_URL_ENV = "DATABASE_URL"
IROHA_DATABASE_URL_ENV = "IROHA_DATABASE_URL"
IROHA_SERVER_ADDR_ENV = "IROHA_SERVER_ADDR"
IROHA_TIMEZONE_ENV = "IROHA_TIMEZONE"
IROHA_ALLOWED_ORIGINS_ENV = "IROHA_ALLOWED_ORIGINS"
IROHA_DATA_DIR_ENV = "IROHA_DATA_DIR"
PUBLIC_IROHA_API_BASE_ENV = "PUBLIC_IROHA_API_BASE"
PUBLIC_IROHA_TIMEZONE_ENV = "PUBLIC_IROHA_TIMEZONE"
THEMES = ("atlas", "grapher", "field-journal", "phenology", "cadence", "archive")
MODES = ("light", "dark")
REPORT_EXPECTED_TEXT = "1 completed"
METRIC_EXPECTED_TEXT = "Steps"
ROUTES = (
    ("expenses?month=2026-08", 1, "Fixture merchant 55", 2),
    ("reports?month=2026-08", 4, REPORT_EXPECTED_TEXT, 4),
    ("metrics?metric=health.steps&date=2026-08", 1, METRIC_EXPECTED_TEXT, 1),
)


def free_port() -> int:
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


def require_commands() -> None:
    missing = [name for name in ("podman", "agent-browser", "mise") if shutil.which(name) is None]
    if missing:
        raise RuntimeError("missing required commands: " + ", ".join(missing))


def run(command: list[str], *, env: dict[str, str] | None = None, stdin=None) -> None:
    print("+ " + " ".join(command), flush=True)
    subprocess.run(command, cwd=ROOT, env=env, stdin=stdin, check=True)


def wait_url(url: str, timeout_s: int = 90) -> None:
    deadline = time.monotonic() + timeout_s
    while time.monotonic() < deadline:
        try:
            with urllib.request.urlopen(url, timeout=2) as response:
                if response.status < 500:
                    return
        except OSError:
            time.sleep(0.5)
    raise RuntimeError(f"timed out waiting for {url}")


def browser_command(session: str, *args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["agent-browser", "--session", session, *args],
        cwd=ROOT,
        text=True,
        capture_output=True,
        check=True,
    )


def get_json(url: str) -> dict:
    request = urllib.request.Request(url, headers={"Accept": "application/json"})
    with urllib.request.urlopen(request, timeout=10) as response:
        return json.load(response)


def timed_json(url: str) -> tuple[dict, dict[str, str], float]:
    started = time.perf_counter()
    request = urllib.request.Request(url, headers={"Accept": "application/json"})
    with urllib.request.urlopen(request, timeout=30) as response:
        body = json.load(response)
        headers = {key.lower(): value for key, value in response.headers.items()}
    elapsed_ms = (time.perf_counter() - started) * 1000
    return body, headers, elapsed_ms


def post_json(url: str, payload: dict) -> dict:
    request = urllib.request.Request(
        url,
        data=json.dumps(payload).encode(),
        headers={"Accept": "application/json", "Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(request, timeout=30) as response:
        return json.load(response)


def psql_scalar(container: str, statement: str) -> str | None:
    result = subprocess.run(
        [
            "podman",
            "exec",
            "--env",
            "PGPASSWORD=iroha_dev",
            container,
            "psql",
            "-h",
            "127.0.0.1",
            "-At",
            "-U",
            "iroha",
            "-d",
            "iroha",
            "-c",
            statement,
        ],
        cwd=ROOT,
        text=True,
        capture_output=True,
    )
    if result.returncode != 0:
        return None
    return result.stdout.strip()


def percentile(values: list[float], fraction: float) -> float:
    ordered = sorted(values)
    if not ordered:
        return 0.0
    index = min(len(ordered) - 1, int((len(ordered) - 1) * fraction))
    return ordered[index]


def performance_gate(server_url: str, container: str) -> None:
    urls = {
        "metric_day": f"{server_url}/api/v1/metrics/health.steps/series?from=2025-12-01&to=2026-01-01&grain=day",
        "metric_month": f"{server_url}/api/v1/metrics/movement.distance_m/series?from=2025-12-01&to=2026-01-01&grain=month&dimension=sport:run",
        "metric_year": f"{server_url}/api/v1/metrics/sleep.asleep_s/series?from=2016-01-01&to=2026-01-01&grain=year",
        "report_series": f"{server_url}/api/v1/reports/monthly-series?end=2025-12&months=12",
    }
    measurements: dict[str, dict[str, float | str]] = {}
    cold_values: list[float] = []
    hit_values: list[float] = []
    for name, url in urls.items():
        body, headers, cold_ms = timed_json(url)
        if not body.get("series") and name.startswith("metric_"):
            raise RuntimeError(f"performance metric returned no series: {name}")
        if headers.get("x-iroha-cache") != "MISS":
            raise RuntimeError(f"performance cold request was not a MISS: {name} {headers}")
        _, hit_headers, hit_ms = timed_json(url)
        if hit_headers.get("x-iroha-cache") != "HIT":
            raise RuntimeError(f"performance repeated request was not a HIT: {name} {hit_headers}")
        cold_values.append(cold_ms)
        hit_values.append(hit_ms)
        measurements[name] = {
            "cold_ms": round(cold_ms, 2),
            "hit_ms": round(hit_ms, 2),
            "cold_cache": headers["x-iroha-cache"],
            "hit_cache": hit_headers["x-iroha-cache"],
        }

    post_json(
        f"{server_url}/api/v1/expenses",
        {
            "occurred_on": "2025-12-28",
            "currency": "JPY",
            "amount_minor": 777,
            "category": "food",
            "merchant": "Performance gate mutation",
            "source": {"kind": "release-candidate-performance", "ref": "gate-mutation"},
        },
    )
    _, invalidation_headers, invalidation_ms = timed_json(urls["metric_month"])
    if invalidation_headers.get("x-iroha-cache") != "MISS":
        raise RuntimeError(
            f"performance request after canonical mutation was not a MISS: {invalidation_headers}"
        )

    entry_count = psql_scalar(container, "select count(*) from tb_cache_entries")
    min_ttl = psql_scalar(
        container,
        "select coalesce(round(min(extract(epoch from expires_at - now())))::bigint, 0) from tb_cache_entries",
    )
    query_stats = psql_scalar(
        container,
        "select coalesce(sum(calls), 0)::bigint || ',' || coalesce(round(sum(total_exec_time)::numeric, 2), 0) from pg_stat_statements where query like '%tb_%'",
    )
    cold_p95 = percentile(cold_values, 0.95)
    evidence = {
        "requests": measurements,
        "cold_p50_ms": round(percentile(cold_values, 0.50), 2),
        "cold_p95_ms": round(cold_p95, 2),
        "hit_p50_ms": round(percentile(hit_values, 0.50), 2),
        "mutation_miss_ms": round(invalidation_ms, 2),
        "cache_entry_count": int(entry_count) if entry_count else None,
        "cache_min_ttl_s": int(min_ttl) if min_ttl else None,
        "postgres_query_stats": query_stats or "unavailable",
        "durable_read_model_decision": "defer" if cold_p95 <= 500 else "investigate",
    }
    print("+ performance gate " + json.dumps(evidence, sort_keys=True), flush=True)


def get_status(url: str) -> tuple[int, dict]:
    try:
        return 200, get_json(url)
    except urllib.error.HTTPError as error:
        return error.code, json.load(error)


def assert_api_contract(server_url: str) -> None:
    expense_scope = "from=2026-08-01&to=2026-09-01"
    first_page = get_json(f"{server_url}/api/v1/expenses?limit=50&{expense_scope}")
    if len(first_page["items"]) != 50 or not first_page["has_more"]:
        raise RuntimeError(f"expense pagination boundary failed: {first_page}")
    cursor = urllib.parse.quote(first_page["next_cursor"], safe="")
    second_page = get_json(f"{server_url}/api/v1/expenses?limit=50&{expense_scope}&cursor={cursor}")
    if not second_page["items"] or second_page["has_more"]:
        raise RuntimeError(f"expense cursor page failed: {second_page}")
    all_expenses = get_json(f"{server_url}/api/v1/expenses?limit=100&{expense_scope}")["items"]
    walked_ids = [item["id"] for item in first_page["items"] + second_page["items"]]
    all_ids = [item["id"] for item in all_expenses]
    if (
        len(all_expenses) != 56
        or len(set(walked_ids)) != len(walked_ids)
        or set(walked_ids) != set(all_ids)
        or {item["currency"] for item in all_expenses} != {"JPY", "USD"}
    ):
        raise RuntimeError(f"expense multi-currency fixture failed: {len(all_expenses)}")
    if any(item["source"]["kind"] != "release-candidate" for item in all_expenses):
        raise RuntimeError("expense source metadata was not preserved")

    report = get_json(f"{server_url}/api/v1/reports/monthly?month=2026-08")
    if report["period"]["timezone"] != "Asia/Tokyo":
        raise RuntimeError(f"report timezone default failed: {report['period']}")
    sections = report["sections"]
    if sections["movement"]["data"]["activity_count"] != 1:
        raise RuntimeError(f"timezone-edge activity was excluded: {sections['movement']}")
    if sections["sleep"]["data"]["session_count"] != 1:
        raise RuntimeError(f"sleep fixture failed: {sections['sleep']}")
    if sections["daily_health"]["data"]["observed_days"] != 1:
        raise RuntimeError(f"partial daily coverage failed: {sections['daily_health']}")
    if sections["media"]["data"]["completed_count"] != 1:
        raise RuntimeError(f"media fixture failed: {sections['media']}")
    if sections["expenses"]["data"]["expense_count"] != 56:
        raise RuntimeError(f"expense report count failed: {sections['expenses']}")
    empty_report = get_json(f"{server_url}/api/v1/reports/monthly?month=2026-07")
    if any(section["state"] != "empty" for section in empty_report["sections"].values()):
        raise RuntimeError(f"empty report period was not empty: {empty_report['sections']}")

    steps = get_json(
        f"{server_url}/api/v1/metrics/health.steps/series?from=2026-08-01&to=2026-09-01&grain=day"
    )
    step_series = steps["series"][0]
    step_point = next(point for point in step_series["points"] if point["period"] == "2026-08-02")
    if (
        step_point["value"] != 12345
        or step_point["observed_days"] != 1
        or step_series["coverage"] != {"expected_periods": 31, "observed_periods": 1}
        or step_series["source"]["source_kinds"] != ["release-candidate"]
    ):
        raise RuntimeError(f"metric coverage/source fixture failed: {step_series}")
    movement = get_json(
        f"{server_url}/api/v1/metrics/movement.distance_m/series"
        "?from=2026-08-01&to=2026-09-01&grain=month&dimension=sport:run"
    )
    if movement["series"][0]["points"][0]["value"] != 8000:
        raise RuntimeError(f"timezone-edge metric bucket failed: {movement}")
    status, _ = get_status(
        f"{server_url}/api/v1/metrics/movement.distance_m/series"
        "?from=2026-08-01&to=2026-09-01&grain=month&dimension=currency:JPY"
    )
    if status != 400:
        raise RuntimeError(f"invalid metric request returned {status}, want 400")


def assert_table_parity(session: str, theme: str, mode: str, route: str) -> None:
    """Every panel must expose exact rows behind its chart, not only the visual."""
    browser_command(
        session,
        "eval",
        "[...document.querySelectorAll('.metric-panel')].forEach(p=>"
        "[...p.querySelectorAll('button')]"
        ".find(b=>b.textContent.trim()==='Table')?.click())",
    )
    browser_command(session, "wait", "400")
    result = browser_command(
        session,
        "eval",
        "JSON.stringify([...document.querySelectorAll('.metric-panel')]"
        ".map(p=>p.querySelectorAll('table tbody tr').length))",
    )
    rows = json.loads(json.loads(result.stdout))
    if not rows or any(count < 1 for count in rows):
        raise RuntimeError(f"table parity missing for {theme}/{mode}/{route}: {rows}")


def browser_matrix(base_url: str, session: str) -> None:
    browser_command(session, "open", base_url)
    for theme in THEMES:
        for mode in MODES:
            browser_command(session, "storage", "local", "set", "iroha-design-language", theme)
            browser_command(session, "storage", "local", "set", "iroha-theme", mode)
            for route, minimum_charts, expected_text, minimum_panels in ROUTES:
                browser_command(session, "open", f"{base_url}/{route}")
                browser_command(session, "wait", "1200")
                result = browser_command(
                    session,
                    "eval",
                    "JSON.stringify({"
                    "language:document.documentElement.dataset.language,"
                    "theme:document.documentElement.dataset.theme,"
                    "charts:document.querySelectorAll('canvas').length,"
                    "url:location.pathname+location.search,"
                    "text:document.body.innerText,"
                    "panels:document.querySelectorAll('.metric-panel').length,"
                    "metadata:document.querySelectorAll('.metric-panel .metric-metadata').length,"
                    "csv:[...document.querySelectorAll('.metric-panel')]"
                    ".filter(p=>[...p.querySelectorAll('button')]"
                    ".some(b=>b.textContent.trim()==='CSV'&&!b.disabled)).length,"
                    "overflow:document.documentElement.scrollWidth>document.documentElement.clientWidth,"
                    "alerts:document.querySelectorAll('[role=alert]').length,"
                    f"focusContrast:{FOCUS_CONTRAST_EXPRESSION}"
                    "})",
                )
                state = json.loads(json.loads(result.stdout))
                if state["language"] != theme or state["theme"] != mode:
                    raise RuntimeError(f"visual mode mismatch for {theme}/{mode}/{route}: {state}")
                if state["url"] != "/" + route:
                    raise RuntimeError(
                        f"route state mismatch for {theme}/{mode}/{route}: {state['url']}"
                    )
                if state["charts"] < minimum_charts:
                    raise RuntimeError(
                        f"missing charts for {theme}/{mode}/{route}: {state['charts']}"
                    )
                if expected_text not in state["text"]:
                    raise RuntimeError(
                        f"fixture text missing for {theme}/{mode}/{route}: {expected_text}"
                    )
                if state["overflow"] or state["alerts"]:
                    raise RuntimeError(f"runtime failure for {theme}/{mode}/{route}: {state}")
                if state["focusContrast"] < 3:
                    raise RuntimeError(
                        f"focus contrast below 3:1 for {theme}/{mode}/{route}: {state}"
                    )
                if state["panels"] < minimum_panels:
                    raise RuntimeError(
                        f"missing metric panels for {theme}/{mode}/{route}: {state['panels']}"
                    )
                if state["metadata"] != state["panels"] or state["csv"] != state["panels"]:
                    raise RuntimeError(
                        f"panel provenance/export incomplete for {theme}/{mode}/{route}: {state}"
                    )
                assert_table_parity(session, theme, mode, route)
                errors = browser_command(session, "errors").stdout.strip()
                if errors and "No page errors" not in errors:
                    raise RuntimeError(f"browser errors for {theme}/{mode}/{route}: {errors}")
                print(f"checked {theme}/{mode}/{route}")


def terminate(process: subprocess.Popen[bytes] | None) -> None:
    if process is None or process.poll() is not None:
        return
    process.terminate()
    try:
        process.wait(timeout=10)
    except subprocess.TimeoutExpired:
        process.kill()
        process.wait()


def main() -> int:
    require_commands()
    suffix = uuid.uuid4().hex[:10]
    container = f"iroha-rc-{suffix}"
    session = f"iroha-rc-{suffix}"
    db_port, server_port, web_port = free_port(), free_port(), free_port()
    database_url = f"postgres://iroha:iroha_dev@127.0.0.1:{db_port}/iroha?sslmode=disable"
    server_url = f"http://127.0.0.1:{server_port}"
    web_url = f"http://127.0.0.1:{web_port}"
    env = os.environ.copy()
    env.update(
        {
            DATABASE_URL_ENV: database_url,
            IROHA_DATABASE_URL_ENV: database_url,
            IROHA_SERVER_ADDR_ENV: f"127.0.0.1:{server_port}",
            IROHA_TIMEZONE_ENV: "Asia/Tokyo",
            IROHA_ALLOWED_ORIGINS_ENV: web_url,
            IROHA_DATA_DIR_ENV: str(ROOT / ".iroha-data" / "release-candidate"),
            PUBLIC_IROHA_API_BASE_ENV: server_url,
            PUBLIC_IROHA_TIMEZONE_ENV: "Asia/Tokyo",
        }
    )
    server_process = None
    web_process = None
    data_dir = tempfile.TemporaryDirectory(prefix="iroha-rc-data-")
    env[IROHA_DATA_DIR_ENV] = data_dir.name
    try:
        run(
            [
                "podman",
                "run",
                "--detach",
                "--rm",
                "--name",
                container,
                "--publish",
                f"127.0.0.1:{db_port}:5432",
                "--env",
                "POSTGRES_DBNAME=iroha",
                "--env",
                "POSTGRES_USER=iroha",
                "--env",
                "POSTGRES_PASS=iroha_dev",
                POSTGIS_IMAGE,
            ]
        )
        deadline = time.monotonic() + 90
        consecutive_ready = 0
        while time.monotonic() < deadline:
            ready = subprocess.run(
                [
                    "podman",
                    "exec",
                    "--env",
                    "PGPASSWORD=iroha_dev",
                    container,
                    "psql",
                    "-h",
                    "127.0.0.1",
                    "-U",
                    "iroha",
                    "-d",
                    "iroha",
                    "-c",
                    "select 1",
                ],
                cwd=ROOT,
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
            )
            if ready.returncode == 0:
                consecutive_ready += 1
                if consecutive_ready == 3:
                    break
            else:
                consecutive_ready = 0
            time.sleep(1)
        else:
            raise RuntimeError("isolated database did not become ready")

        run([sys.executable, "scripts/db.py", "apply"], env=env)
        run(
            [
                "mise",
                "exec",
                "--",
                "env",
                f"{DATABASE_URL_ENV}={database_url}",
                "go",
                "-C",
                str(SERVER_DIR),
                "test",
                "-p",
                "1",
                "-tags=integration",
                "./...",
            ]
        )
        with RESET_FILE.open("rb") as reset:
            print("+ reset isolated database after integration tests", flush=True)
            subprocess.run(
                [
                    "podman",
                    "exec",
                    "-i",
                    "--env",
                    "PGPASSWORD=iroha_dev",
                    container,
                    "psql",
                    "-h",
                    "127.0.0.1",
                    "-v",
                    "ON_ERROR_STOP=1",
                    "-U",
                    "iroha",
                    "-d",
                    "iroha",
                ],
                cwd=ROOT,
                stdin=reset,
                check=True,
            )
        with SEED_FILE.open("rb") as seed:
            print("+ seed isolated release-candidate database", flush=True)
            subprocess.run(
                [
                    "podman",
                    "exec",
                    "-i",
                    "--env",
                    "PGPASSWORD=iroha_dev",
                    container,
                    "psql",
                    "-h",
                    "127.0.0.1",
                    "-v",
                    "ON_ERROR_STOP=1",
                    "-U",
                    "iroha",
                    "-d",
                    "iroha",
                ],
                cwd=ROOT,
                stdin=seed,
                check=True,
            )
        with PERFORMANCE_SEED_FILE.open("rb") as performance_seed:
            print("+ seed isolated performance database", flush=True)
            subprocess.run(
                [
                    "podman",
                    "exec",
                    "-i",
                    "--env",
                    "PGPASSWORD=iroha_dev",
                    container,
                    "psql",
                    "-h",
                    "127.0.0.1",
                    "-v",
                    "ON_ERROR_STOP=1",
                    "-U",
                    "iroha",
                    "-d",
                    "iroha",
                ],
                cwd=ROOT,
                stdin=performance_seed,
                check=True,
            )
        run(["make", "web-build"], env=env)

        print("+ start production server", flush=True)
        server_process = subprocess.Popen(
            ["mise", "exec", "--", "go", "-C", str(SERVER_DIR), "run", "./cmd/iroha-server"],
            cwd=ROOT,
            env=env,
        )
        wait_url(server_url + "/healthz")
        wait_url(server_url + "/readyz")
        if server_process.poll() is not None:
            raise RuntimeError(f"iroha-server exited with {server_process.returncode}")
        readiness = get_json(server_url + "/readyz")
        if readiness.get("status") != "ready":
            raise RuntimeError(f"database readiness contract failed: {readiness}")
        assert_api_contract(server_url)
        performance_gate(server_url, container)
        print("+ start production web preview", flush=True)
        web_process = subprocess.Popen(
            [
                "mise",
                "exec",
                "--",
                "bun",
                "run",
                "preview",
                "--",
                "--host",
                "127.0.0.1",
                "--port",
                str(web_port),
            ],
            cwd=WEB_DIR,
            env=env,
        )
        wait_url(web_url)
        if web_process.poll() is not None:
            raise RuntimeError(f"web preview exited with {web_process.returncode}")
        browser_matrix(web_url, session)
        print("release-candidate gate passed")
        return 0
    finally:
        subprocess.run(
            ["agent-browser", "--session", session, "close"],
            cwd=ROOT,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        terminate(web_process)
        terminate(server_process)
        subprocess.run(
            ["podman", "rm", "--force", container],
            cwd=ROOT,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        data_dir.cleanup()


if __name__ == "__main__":
    raise SystemExit(main())
