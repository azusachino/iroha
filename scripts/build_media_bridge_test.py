import unittest

import build_media_bridge


class FakeCursor:
    def __init__(self) -> None:
        self.executemany_calls: list[tuple[str, list[tuple[str, str, str]]]] = []

    def __enter__(self) -> "FakeCursor":
        return self

    def __exit__(self, *exc_info: object) -> None:
        return None

    def executemany(self, query: str, rows: list[tuple[str, str, str]]) -> None:
        self.executemany_calls.append((query, list(rows)))


class FakeConnection:
    def __init__(self) -> None:
        self.cursor_obj = FakeCursor()
        self.committed = False

    def cursor(self) -> FakeCursor:
        return self.cursor_obj

    def commit(self) -> None:
        self.committed = True


class BuildMediaBridgeTest(unittest.TestCase):
    def test_build_bangumi_to_mal_skips_unmatched(self) -> None:
        records = [
            {"bgm_id": "8", "mal_id": "2904"},
            {"bgm_id": "12", "mal_id": ""},
            {"bgm_id": "50"},
        ]
        self.assertEqual(build_media_bridge.build_bangumi_to_mal(records), {"8": "2904"})

    def test_build_bangumi_to_mal_values_are_strings(self) -> None:
        records = [{"bgm_id": 8, "mal_id": 2904}]
        result = build_media_bridge.build_bangumi_to_mal(records)
        self.assertEqual(result, {"8": "2904"})
        self.assertIsInstance(result["8"], str)

    def test_build_mal_to_anilist_skips_unmatched(self) -> None:
        records = [
            {"mal_id": 290, "anilist_id": 290},
            {"mal_id": 300},
            {"anilist_id": 1225},
        ]
        self.assertEqual(build_media_bridge.build_mal_to_anilist(records), {"290": "290"})

    def test_upsert_hop_sends_one_row_per_mapping_and_commits(self) -> None:
        conn = FakeConnection()
        build_media_bridge.upsert_hop(
            conn, build_media_bridge.HOP_BANGUMI_TO_MAL, {"8": "2904", "12": "300"}
        )
        self.assertTrue(conn.committed)
        [(query, rows)] = conn.cursor_obj.executemany_calls
        self.assertIn("on conflict (hop, source_id) do update", query)
        self.assertEqual(
            sorted(rows),
            [("bangumi_to_mal", "12", "300"), ("bangumi_to_mal", "8", "2904")],
        )


if __name__ == "__main__":
    unittest.main()
