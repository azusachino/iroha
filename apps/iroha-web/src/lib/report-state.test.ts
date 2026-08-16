import { describe, expect, it } from "vitest";
import { reportSectionStateCopy } from "@iroha/shared/domain/report";

describe("report section presentation state", () => {
  it("translates the wire empty state into human-facing copy", () => {
    expect(reportSectionStateCopy("empty")).toEqual({
      label: "No records",
      description: "No canonical records are present for this period.",
    });
  });

  it("keeps the wire state out of the available badge copy", () => {
    const copy = reportSectionStateCopy("available");
    expect(copy.label).toBe("Included");
    expect(copy.label.toLowerCase()).not.toContain("available");
  });
});
