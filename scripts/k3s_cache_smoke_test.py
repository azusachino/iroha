import unittest

try:
    from k3s_cache_smoke import report_url, verify_cache
except ModuleNotFoundError:
    from scripts.k3s_cache_smoke import report_url, verify_cache


class FakeResponse:
    def __init__(self, cache_state: str, body: bytes = b'{"schema":"monthly-report.v1"}') -> None:
        self.status_code = 200
        self.headers = {"X-Iroha-Cache": cache_state}
        self.content = body
        self.text = body.decode()

    def json(self):
        return {"schema": "monthly-report.v1"}


class FakeSession:
    def __init__(self, responses: list[FakeResponse]) -> None:
        self.responses = responses
        self.calls = []

    def get(self, url, **kwargs):
        self.calls.append((url, kwargs))
        return self.responses.pop(0)


class K3sCacheSmokeTest(unittest.TestCase):
    def test_report_url_uses_canonical_query_encoding(self) -> None:
        self.assertEqual(
            report_url("https://iroha.test/", "2099-01", "Asia/Tokyo"),
            "https://iroha.test/api/v1/reports/monthly?month=2099-01&timezone=Asia%2FTokyo",
        )

    def test_verify_cache_accepts_cold_then_warm_reads(self) -> None:
        session = FakeSession([FakeResponse("MISS"), FakeResponse("HIT")])

        result = verify_cache("https://iroha.test", "2099-01", session=session)

        self.assertEqual(result["first"], "MISS")
        self.assertEqual(result["second"], "HIT")
        self.assertEqual(len(session.calls), 2)
        self.assertEqual(session.calls[0][0], session.calls[1][0])
        self.assertEqual(session.calls[0][1]["headers"]["Accept"], "application/json")

    def test_verify_cache_accepts_a_pre_warmed_probe(self) -> None:
        session = FakeSession([FakeResponse("HIT"), FakeResponse("HIT")])

        result = verify_cache("https://iroha.test", "2099-01", session=session)

        self.assertEqual(result["first"], "HIT")
        self.assertEqual(result["second"], "HIT")

    def test_verify_cache_rejects_a_second_miss(self) -> None:
        session = FakeSession([FakeResponse("MISS"), FakeResponse("MISS")])

        with self.assertRaisesRegex(RuntimeError, "second cache state"):
            verify_cache("https://iroha.test", "2099-01", session=session)


if __name__ == "__main__":
    unittest.main()
