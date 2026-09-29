import gzip
import unittest

from web_bundle_report import sizes


class WebBundleReportTest(unittest.TestCase):
    def test_sums_raw_and_per_asset_gzip_sizes(self):
        contents = [b"first asset", b"second asset"]
        count, raw_bytes, gzip_bytes = sizes(contents)

        self.assertEqual(count, 2)
        self.assertEqual(raw_bytes, sum(map(len, contents)))
        self.assertEqual(
            gzip_bytes,
            sum(len(gzip.compress(content, mtime=0)) for content in contents),
        )


if __name__ == "__main__":
    unittest.main()
