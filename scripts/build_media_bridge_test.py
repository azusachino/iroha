import unittest

import build_media_bridge


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


if __name__ == "__main__":
    unittest.main()
