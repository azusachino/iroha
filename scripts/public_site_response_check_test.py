import io
import unittest
from unittest import mock
import urllib.error

import public_site_response_check as checker


class Response(io.BytesIO):
    def __init__(self, body=b"shell", status=200, cache=""):
        super().__init__(body)
        self.status = status
        self.headers = {
            "Strict-Transport-Security": "max-age=31536000",
            "Permissions-Policy": "camera=(), microphone=(), geolocation=(), payment=(), usb=()",
            "X-Content-Type-Options": "nosniff",
            "Content-Security-Policy": "default-src 'self'",
            "Cache-Control": cache,
        }


def responses():
    return [
        Response(),
        Response(),
        Response(b"asset", cache="public, max-age=31536000, immutable"),
        Response(),
        Response(status=502),
    ]


class ResponseCheckTest(unittest.TestCase):
    def test_rejects_remote_base(self):
        for base in ["https://localhost", "http://example.com", ""]:
            with self.assertRaises(ValueError):
                checker.check(base)

    def test_checks_fixture(self):
        with mock.patch.object(checker.urllib.request, "urlopen", side_effect=responses()):
            checker.check("http://127.0.0.1:18080")

    def test_retries_startup(self):
        with (
            mock.patch.object(checker.urllib.request, "urlopen", side_effect=[urllib.error.URLError("starting"), *responses()]),
            mock.patch.object(checker.time, "sleep"),
        ):
            checker.check("http://localhost:18080")

    def test_rejects_missing_hsts_and_html_immutable(self):
        for header, value in [("Strict-Transport-Security", ""), ("Cache-Control", "immutable")]:
            fixture = responses()
            fixture[1].headers[header] = value
            with (
                mock.patch.object(checker.urllib.request, "urlopen", side_effect=fixture),
                self.assertRaises(AssertionError),
            ):
                checker.check("http://localhost:18080")
