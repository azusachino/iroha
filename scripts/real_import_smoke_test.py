import unittest

try:
    from real_import_smoke import parse_vitals_explore_output
except ModuleNotFoundError:
    from scripts.real_import_smoke import parse_vitals_explore_output


class RealImportSmokeTest(unittest.TestCase):
    def test_parse_vitals_explore_totals(self) -> None:
        output = """metric            unit          records   days  rd/day max/day      min     mean      max  range
resting_hr        count/min          939    900     1.0       2    48.00    57.00    82.00  2020..2024
hrv_sdnn          ms                  20     10     2.0       4     6.00   120.00   269.00  2020..2024
"""

        got = parse_vitals_explore_output(output)

        self.assertEqual(got["resting_hr"]["records"], 939)
        self.assertEqual(got["resting_hr"]["days"], 900)
        self.assertEqual(got["hrv_sdnn"]["avg"], 120)
        self.assertEqual(got["hrv_sdnn"]["max"], 269)


if __name__ == "__main__":
    unittest.main()
