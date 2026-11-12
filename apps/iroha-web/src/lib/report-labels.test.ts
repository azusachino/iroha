import { describe, expect, it } from "vitest";

import { disambiguateLabels } from "@iroha/shared/domain/report";

describe("disambiguateLabels", () => {
  it("keeps unique labels untouched", () => {
    const rows = disambiguateLabels([
      { label: "Steps", value: "1,234" },
      { label: "Distance", value: "7.8 km" },
    ]);
    expect(rows).toEqual([
      { label: "Steps", value: "1,234", key: "Steps" },
      { label: "Distance", value: "7.8 km", key: "Distance" },
    ]);
  });

  it("disambiguates repeated labels so each-block keys stay unique", () => {
    // Regression: vo2max arriving under two unit spellings produced two rows
    // labeled "Health · VO₂ max" and crashed reports with each_key_duplicate.
    const rows = disambiguateLabels([
      { label: "Health · VO₂ max", value: "38.9 ml/kg/min" },
      { label: "Health · VO₂ max", value: "38.5 ml/(kg·min)" },
      { label: "Health · VO₂ max", value: "38.1 ml/kg_min" },
    ]);
    expect(rows.map((row) => row.key)).toEqual([
      "Health · VO₂ max",
      "Health · VO₂ max#2",
      "Health · VO₂ max#3",
    ]);
    expect(rows.map((row) => row.label)).toEqual([
      "Health · VO₂ max",
      "Health · VO₂ max (2)",
      "Health · VO₂ max (3)",
    ]);
    expect(new Set(rows.map((row) => row.key)).size).toBe(rows.length);
  });

  it("preserves extra row fields", () => {
    const rows = disambiguateLabels([
      { label: "A", value: "1", detail: "first" },
      { label: "A", value: "2", detail: "second" },
    ]);
    expect(rows[1].detail).toBe("second");
  });
});
