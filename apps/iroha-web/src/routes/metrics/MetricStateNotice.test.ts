import { render } from "svelte/server";
import { describe, expect, it } from "vitest";
import MetricStateNotice from "./MetricStateNotice.svelte";

describe("MetricStateNotice", () => {
  it("renders the required-dimension chooser notice", () => {
    const { body } = render(MetricStateNotice, {
      props: { kind: "required", labels: ["Currency"] },
    });

    expect(body).toContain("Selection required");
    expect(body).toContain("Choose Currency");
    expect(body).toContain("before Iroha can request");
  });

  it("preserves the explicit dimension in empty-state copy", () => {
    const { body } = render(MetricStateNotice, {
      props: {
        kind: "empty",
        metricLabel: "Expenses",
        month: "2026-08",
        dimensionSummary: "Currency: EUR",
      },
    });

    expect(body).toContain("No expenses values in this window.");
    expect(body).toContain("Currency: EUR");
    expect(body).toContain("The explicit selection remains unchanged.");
  });
});
