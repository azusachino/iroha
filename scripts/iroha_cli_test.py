import io
import json
import os
import tempfile
import unittest
from pathlib import Path
from unittest import mock

import iroha_cli


class FakeResponse:
    def __init__(self, body: bytes, status: int = 200) -> None:
        self.body = body
        self.status = status

    def __enter__(self):
        return self

    def __exit__(self, *_args):
        return False

    def getcode(self) -> int:
        return self.status

    def read(self) -> bytes:
        return self.body


class IrohaCLITransportTest(unittest.TestCase):
    def test_api_base_comes_from_environment(self) -> None:
        with mock.patch.dict(os.environ, {"IROHA_API_BASE": "http://iroha.test/"}):
            self.assertEqual(iroha_cli.api_base_from_environment(), "http://iroha.test")

    def test_request_uses_configured_url_and_returns_bytes_unchanged(self) -> None:
        response_body = b'{"value": 1}\n'
        calls = []

        def open_url(request, timeout):
            calls.append((request, timeout))
            return FakeResponse(response_body)

        result = iroha_cli.IrohaClient("http://iroha.test", timeout_s=7, opener=open_url).request(
            "GET", "/api/v1/reports/monthly?month=2026-08"
        )

        self.assertEqual(result, response_body)
        self.assertEqual(calls[0][0].full_url, "http://iroha.test/api/v1/reports/monthly?month=2026-08")
        self.assertEqual(calls[0][1], 7)
        self.assertEqual(calls[0][0].method, "GET")

    def test_request_sends_json_body_without_client_reformatting(self) -> None:
        body = b'{ "amount_minor": 800, "currency": "JPY" }\n'
        captured = {}

        def open_url(request, timeout):
            captured["data"] = request.data
            captured["content_type"] = request.get_header("Content-type")
            return FakeResponse(b"{}")

        iroha_cli.IrohaClient("http://iroha.test", opener=open_url).request("POST", "/api/v1/expenses", body)
        self.assertEqual(captured, {"data": body, "content_type": "application/json"})

    def test_structured_api_error_is_preserved(self) -> None:
        body = b'{"code":"invalid_month","message":"invalid report month","request_id":"req-1"}'
        with self.assertRaises(iroha_cli.APIError) as raised:
            iroha_cli.IrohaClient(opener=lambda *_args, **_kwargs: FakeResponse(body, 400)).request(
                "GET", "/api/v1/reports/monthly"
            )
        self.assertEqual((raised.exception.status, raised.exception.code, raised.exception.request_id), (400, "invalid_month", "req-1"))
        self.assertIn("invalid report month", str(raised.exception))

    def test_transport_error_is_distinct_from_api_error(self) -> None:
        def open_url(*_args, **_kwargs):
            raise OSError("connection refused")

        with self.assertRaises(iroha_cli.TransportError):
            iroha_cli.IrohaClient(opener=open_url).request("GET", "/healthz")

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


if __name__ == "__main__":
    unittest.main()
