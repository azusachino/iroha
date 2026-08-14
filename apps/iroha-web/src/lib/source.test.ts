import { describe, expect, it } from "vitest";
import { sourceBrand } from "@iroha/shared/source";

describe("source brand registry", () => {
  it("maps canonical Apple export sources to a business label", () => {
    expect(sourceBrand("apple_health_export")).toMatchObject({
      id: "apple-health",
      label: "Apple Health",
      mark: "",
    });
  });

  it("recognizes future wearable sources without changing the API shape", () => {
    expect(sourceBrand("garmin_connect")).toMatchObject({
      id: "garmin",
      label: "Garmin",
      mark: "G",
    });
  });

  it("preserves an unknown raw source as a readable fallback", () => {
    expect(sourceBrand("my_import")).toMatchObject({
      id: "unknown",
      label: "My Import",
      raw: "my_import",
    });
  });
});
