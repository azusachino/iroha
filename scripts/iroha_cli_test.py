import hashlib
import io
import json
import os
import tempfile
import unittest
from pathlib import Path
from unittest import mock

import iroha_cli
import requests


class FakeResponse:
    def __init__(self, body: bytes, status: int = 200) -> None:
        self.body = body
        self.status_code = status
        self.content = body


class FakeSession:
    def __init__(
        self, response: FakeResponse | None = None, error: Exception | None = None
    ) -> None:
        self.response = response or FakeResponse(b"{}")
        self.error = error
        self.calls = []

    def request(self, method, url, **kwargs):
        self.calls.append((method, url, kwargs))
        if self.error is not None:
            raise self.error
        return self.response


class IrohaCLITransportTest(unittest.TestCase):
    def test_api_base_comes_from_environment(self) -> None:
        with mock.patch.dict(os.environ, {"IROHA_API_BASE": "http://iroha.test/"}):
            self.assertEqual(iroha_cli.api_base_from_environment(), "http://iroha.test")

    def test_request_uses_configured_url_and_returns_bytes_unchanged(self) -> None:
        response_body = b'{"value": 1}\n'
        session = FakeSession(FakeResponse(response_body))
        result = iroha_cli.IrohaClient("http://iroha.test", timeout_s=7, session=session).request(
            "GET", "/api/v1/reports/monthly?month=2026-08"
        )

        self.assertEqual(result, response_body)
        self.assertEqual(
            session.calls[0][0:2], ("GET", "http://iroha.test/api/v1/reports/monthly?month=2026-08")
        )
        self.assertEqual(session.calls[0][2]["timeout"], 7)
        self.assertEqual(session.calls[0][2]["headers"]["Accept"], "application/json")

    def test_request_sends_json_body_without_client_reformatting(self) -> None:
        body = b'{ "amount_minor": 800, "currency": "JPY" }\n'
        session = FakeSession(FakeResponse(b"{}"))
        iroha_cli.IrohaClient("http://iroha.test", session=session).request(
            "POST", "/api/v1/expenses", body
        )
        self.assertEqual(session.calls[0][2]["data"], body)
        self.assertEqual(session.calls[0][2]["headers"]["Content-Type"], "application/json")

    def test_upload_file_uses_multipart_and_cli_source(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "activity.gpx"
            path.write_bytes(b"<gpx />")
            session = FakeSession(FakeResponse(b'{"id":"raw_1"}', 201))
            result = iroha_cli.IrohaClient("http://iroha.test", session=session).upload_file(
                str(path), "gpx"
            )

        self.assertEqual(result, b'{"id":"raw_1"}')
        self.assertEqual(session.calls[0][0:2], ("POST", "http://iroha.test/api/v1/raw-files/"))
        self.assertEqual(session.calls[0][2]["data"], {"source_kind": "gpx", "uploaded_via": "cli"})
        self.assertEqual(session.calls[0][2]["files"]["file"][0], "activity.gpx")

    def test_structured_api_error_is_preserved(self) -> None:
        body = b'{"code":"invalid_month","message":"invalid report month","request_id":"req-1"}'
        with self.assertRaises(iroha_cli.APIError) as raised:
            iroha_cli.IrohaClient(session=FakeSession(FakeResponse(body, 400))).request(
                "GET", "/api/v1/reports/monthly"
            )
        self.assertEqual(
            (raised.exception.status, raised.exception.code, raised.exception.request_id),
            (400, "invalid_month", "req-1"),
        )
        self.assertIn("invalid report month", str(raised.exception))

    def test_transport_error_is_distinct_from_api_error(self) -> None:
        with self.assertRaises(iroha_cli.TransportError):
            iroha_cli.IrohaClient(
                session=FakeSession(error=requests.ConnectionError("connection refused"))
            ).request("GET", "/healthz")

    def test_json_input_preserves_file_bytes_and_validates_json(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "draft.json"
            data = b'{\n  "amount_minor": 800\n}\n'
            path.write_bytes(data)
            self.assertEqual(iroha_cli.read_json_input(str(path)), data)

    def test_json_input_rejects_invalid_file(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "draft.json"
            path.write_text("not json")
            with self.assertRaises(iroha_cli.CLIError):
                iroha_cli.read_json_input(str(path))

    def test_write_response_keeps_json_bytes(self) -> None:
        output = io.BytesIO()
        iroha_cli.write_response(b'{"value":1}\n', output)
        self.assertEqual(output.getvalue(), b'{"value":1}\n')

    def test_response_json_is_not_reinterpreted(self) -> None:
        value = json.loads(b'{"value":1}')
        self.assertEqual(value, {"value": 1})


class FakeClient:
    def __init__(self, response: bytes = b'{"id":"exp_1"}\n') -> None:
        self.response = response
        self.calls = []

    def request(self, method: str, path: str, body: bytes | None = None) -> bytes:
        self.calls.append((method, path, body))
        return self.response

    def upload_file(self, path: str, source_kind: str) -> bytes:
        self.calls.append(("UPLOAD", path, source_kind))
        return b'{"id":"raw_1"}'


class IrohaCLIGeneralResourceTest(unittest.TestCase):
    def test_import_file_uploads_then_enqueues_canonical_import(self) -> None:
        client = FakeClient(b'{"id":"imp_1","status":"pending"}')
        args = iroha_cli.build_parser().parse_args(
            ["import", "file", "activity.gpx", "--kind", "gpx"]
        )
        with mock.patch.object(iroha_cli, "output_result") as output:
            iroha_cli.run_import_command(args, client)
        self.assertEqual(client.calls[0], ("UPLOAD", "activity.gpx", "gpx"))
        self.assertEqual(client.calls[1][0:2], ("POST", "/api/v1/imports/"))
        self.assertEqual(
            json.loads(client.calls[1][2]), {"raw_file_id": "raw_1", "parser_kind": "gpx"}
        )
        output.assert_called_once_with(client.response, "json")

    def test_activity_list_forwards_stable_filters(self) -> None:
        client = FakeClient(b'{"items":[]}')
        args = iroha_cli.build_parser().parse_args(
            [
                "activity",
                "list",
                "--from",
                "2026-08-01T00:00:00Z",
                "--sport",
                "run",
                "--limit",
                "5",
            ]
        )
        with mock.patch.object(iroha_cli, "output_result"):
            iroha_cli.run_read_command(args, client)
        self.assertEqual(
            client.calls[0],
            (
                "GET",
                "/api/v1/activities/?started_from=2026-08-01T00%3A00%3A00Z&sport_type=run&limit=5",
                None,
            ),
        )

    def test_sleep_get_and_daily_list_preserve_json(self) -> None:
        for argv, expected in [
            (["sleep", "get", "sleep_1"], "/api/v1/sleep/sleep_1"),
            (
                ["daily", "list", "--from", "2026-08-01", "--to", "2026-08-31"],
                "/api/v1/daily/?from=2026-08-01&to=2026-08-31",
            ),
        ]:
            with self.subTest(argv=argv):
                client = FakeClient(b'{"items":[]}')
                args = iroha_cli.build_parser().parse_args(argv)
                with mock.patch.object(iroha_cli, "output_result") as output:
                    iroha_cli.run_read_command(args, client)
                self.assertEqual(client.calls[0], ("GET", expected, None))
                output.assert_called_once_with(client.response, "json")

    def test_media_list_and_get_use_existing_read_api(self) -> None:
        client = FakeClient()
        args = iroha_cli.build_parser().parse_args(
            [
                "media",
                "list",
                "--family",
                "anime",
                "--media-type",
                "tv",
                "--completed-year",
                "2026",
            ]
        )
        with mock.patch.object(iroha_cli, "output_result"):
            iroha_cli.run_read_command(args, client)
        self.assertEqual(
            client.calls[0][1], "/api/v1/media/?family=anime&media_type=tv&completed_year=2026"
        )

        args = iroha_cli.build_parser().parse_args(["media", "get", "media_1"])
        with mock.patch.object(iroha_cli, "output_result"):
            iroha_cli.run_read_command(args, client)
        self.assertEqual(client.calls[1][1], "/api/v1/media/media_1")


class IrohaCLIExpenseCommandTest(unittest.TestCase):
    def test_create_adds_hash_ref_and_persists_it_for_retry(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "draft.json"
            original = (
                b'{"occurred_on":"2026-08-12","currency":"JPY",'
                b'"amount_minor":800,"category":"food"}\n'
            )
            path.write_bytes(original)
            client = FakeClient()
            args = iroha_cli.build_parser().parse_args(["expense", "create", "--input", str(path)])

            with mock.patch.object(iroha_cli, "output_result"):
                iroha_cli.run_expense_command(args, client)

            expected_ref = hashlib.sha256(original).hexdigest()
            first_body = json.loads(client.calls[0][2])
            self.assertEqual(first_body["source"], {"kind": "cli", "ref": expected_ref})
            self.assertEqual(json.loads(path.read_bytes())["source"]["ref"], expected_ref)
            second_body = iroha_cli.prepare_expense_create_input(str(path))
            self.assertEqual(json.loads(second_body)["source"]["ref"], expected_ref)

    def test_explicit_source_ref_is_written_to_draft(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "draft.json"
            path.write_text('{"source":{"kind":"cli"}}')
            iroha_cli.prepare_expense_create_input(str(path), "manual-uuid-ref")
            self.assertEqual(json.loads(path.read_bytes())["source"]["ref"], "manual-uuid-ref")

    def test_list_encodes_filters_and_forwards_table_output(self) -> None:
        client = FakeClient(b'{"items":[]}\n')
        args = iroha_cli.build_parser().parse_args(
            [
                "expense",
                "list",
                "--from",
                "2026-08-01",
                "--to",
                "2026-09-01",
                "--currency",
                "JPY",
                "--limit",
                "5",
                "--format",
                "table",
            ]
        )
        with mock.patch.object(iroha_cli, "output_result") as output:
            iroha_cli.run_expense_command(args, client)
        self.assertEqual(
            client.calls[0],
            ("GET", "/api/v1/expenses?from=2026-08-01&to=2026-09-01&currency=JPY&limit=5", None),
        )
        output.assert_called_once_with(b'{"items":[]}\n', "table", iroha_cli.expense_table)

    def test_update_forwards_replacement_json_without_mutation(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "replacement.json"
            body = (
                b'{"occurred_on":"2026-08-12","currency":"JPY",'
                b'"amount_minor":900,"category":"food"}\n'
            )
            path.write_bytes(body)
            client = FakeClient()
            args = iroha_cli.build_parser().parse_args(
                ["expense", "update", "exp_1", "--input", str(path)]
            )
            with mock.patch.object(iroha_cli, "output_result"):
                iroha_cli.run_expense_command(args, client)
            self.assertEqual(client.calls[0], ("PUT", "/api/v1/expenses/exp_1", body))
            self.assertEqual(path.read_bytes(), body)

    def test_delete_has_no_stdout_payload(self) -> None:
        client = FakeClient(b"")
        args = iroha_cli.build_parser().parse_args(["expense", "delete", "exp_1"])
        iroha_cli.run_expense_command(args, client)
        self.assertEqual(client.calls, [("DELETE", "/api/v1/expenses/exp_1", None)])

    def test_table_formatter_has_stable_columns(self) -> None:
        result = iroha_cli.expense_table(
            {
                "items": [
                    {
                        "id": "exp_1",
                        "occurred_on": "2026-08-12",
                        "currency": "JPY",
                        "amount_minor": 800,
                        "category": "food",
                        "merchant": "Ramen",
                    }
                ]
            }
        )
        self.assertIn("id", result)
        self.assertIn("exp_1", result)
        self.assertIn("Ramen", result)


class IrohaCLIReportCommandTest(unittest.TestCase):
    def test_monthly_report_forwards_month_and_timezone_and_preserves_json(self) -> None:
        response = b'{"schema":"monthly-report.v1","sections":{}}\n'
        client = FakeClient(response)
        args = iroha_cli.build_parser().parse_args(
            ["report", "monthly", "--month", "2026-08", "--timezone", "Asia/Tokyo"]
        )
        with mock.patch.object(iroha_cli, "output_result") as output:
            iroha_cli.run_report_command(args, client)
        self.assertEqual(
            client.calls,
            [("GET", "/api/v1/reports/monthly?month=2026-08&timezone=Asia%2FTokyo", None)],
        )
        output.assert_called_once_with(response, "json", iroha_cli.monthly_report_table)

    def test_monthly_report_reads_timezone_from_environment(self) -> None:
        client = FakeClient()
        args = iroha_cli.build_parser().parse_args(["report", "monthly", "--month", "2026-08"])
        with (
            mock.patch.dict(os.environ, {"IROHA_TIMEZONE": "UTC"}),
            mock.patch.object(iroha_cli, "output_result"),
        ):
            iroha_cli.run_report_command(args, client)
        self.assertIn("timezone=UTC", client.calls[0][1])

    def test_monthly_report_omits_timezone_to_use_server_default(self) -> None:
        client = FakeClient()
        args = iroha_cli.build_parser().parse_args(["report", "monthly", "--month", "2026-08"])
        with (
            mock.patch.dict(os.environ, {}, clear=True),
            mock.patch.object(iroha_cli, "output_result"),
        ):
            iroha_cli.run_report_command(args, client)
        self.assertEqual(client.calls[0][1], "/api/v1/reports/monthly?month=2026-08")

    def test_monthly_report_table_is_presentation_only(self) -> None:
        result = iroha_cli.monthly_report_table(
            {
                "period": {"month": "2026-08", "timezone": "UTC"},
                "sections": {
                    "expenses": {
                        "state": "available",
                        "data": {"expense_count": 2, "totals_by_currency": [{}, {}]},
                    },
                    "sleep": {"state": "empty", "data": None},
                },
            }
        )
        self.assertIn("period: 2026-08 (UTC)", result)
        self.assertIn("expenses", result)
        self.assertIn("currencies=2", result)
        self.assertIn("sleep", result)


class IrohaCLIMetricCommandTest(unittest.TestCase):
    def test_metric_list_forwards_catalog(self) -> None:
        response = b'{"schema":"metric-catalog.v1","metrics":[]}'
        client = FakeClient(response)
        args = iroha_cli.build_parser().parse_args(["metric", "list"])
        with mock.patch.object(iroha_cli, "output_result") as output:
            iroha_cli.run_metric_command(args, client)
        self.assertEqual(client.calls, [("GET", "/api/v1/metrics", None)])
        output.assert_called_once_with(response, "json", iroha_cli.metric_table)

    def test_metric_series_preserves_repeatable_dimensions(self) -> None:
        client = FakeClient(b'{"schema":"metric-series.v1"}')
        args = iroha_cli.build_parser().parse_args(
            [
                "metric",
                "series",
                "expenses.amount_minor",
                "--from",
                "2026-01-01",
                "--to",
                "2026-02-01",
                "--grain",
                "month",
                "--timezone",
                "Asia/Tokyo",
                "--dimension",
                "currency:JPY",
                "--dimension",
                "category:food",
            ]
        )
        with mock.patch.object(iroha_cli, "output_result"):
            iroha_cli.run_metric_command(args, client)
        self.assertEqual(client.calls[0][0], "GET")
        self.assertEqual(
            client.calls[0][1],
            "/api/v1/metrics/expenses.amount_minor/series?from=2026-01-01&to=2026-02-01&grain=month&timezone=Asia%2FTokyo&dimension=currency%3AJPY&dimension=category%3Afood",
        )

    def test_metric_series_table_keeps_minor_value(self) -> None:
        result = iroha_cli.metric_series_table(
            {
                "metric_id": "expenses.amount_minor",
                "period": {"timezone": "Asia/Tokyo"},
                "series": [
                    {
                        "dimensions": {"currency": "JPY"},
                        "points": [{"period": "2026-01", "value_minor": 800, "observed_days": 1}],
                    }
                ],
            }
        )
        self.assertIn("800", result)
        self.assertIn("currency=JPY", result)


if __name__ == "__main__":
    unittest.main()
