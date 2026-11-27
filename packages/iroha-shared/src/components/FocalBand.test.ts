import { render } from "svelte/server";
import { describe, expect, it } from "vitest";
import FocalBand from "./FocalBand.svelte";

describe("FocalBand", () => {
  it("renders the primary metric with its unit and detail", () => {
    const { body } = render(FocalBand, {
      props: {
        heading: "Monthly overview",
        primary: {
          label: "Distance",
          value: "142.8",
          unit: "km",
          detail: "12 sessions logged",
        },
        secondary: [{ label: "Duration", value: "11h 40m" }],
      },
    });

    expect(body).toContain("Distance");
    expect(body).toContain("142.8");
    expect(body).toContain("km");
    expect(body).toContain("12 sessions logged");
  });

  it("renders secondary items as a semantic dl with dt and dd", () => {
    const { body } = render(FocalBand, {
      props: {
        heading: "Sleep summary",
        primary: { label: "Asleep", value: "7h 15m" },
        secondary: [
          { label: "Efficiency", value: "88%", detail: "Main sleep" },
          { label: "In bed", value: "8h 12m" },
        ],
      },
    });

    expect(body).toContain("<dl");
    expect(body).toMatch(/<dt[^>]*>Efficiency<\/dt>/);
    expect(body).toContain("88%");
    expect(body).toContain("Main sleep");
    expect(body).toMatch(/<dt[^>]*>In bed<\/dt>/);
    expect(body).toContain("8h 12m");
  });

  it("renders the optional empty message as a muted line", () => {
    const { body } = render(FocalBand, {
      props: {
        heading: "Expenses",
        primary: { label: "Total spend", value: "¥48,200" },
        secondary: [{ label: "Entries", value: "19" }],
        empty: "3 currencies empty: USD, EUR, GBP",
      },
    });

    expect(body).toContain("3 currencies empty: USD, EUR, GBP");
    expect(body).toContain("focal-empty-text");
  });

  it("names the region for a screen reader via aria-labelledby", () => {
    const { body } = render(FocalBand, {
      props: {
        heading: "Daily summary",
        primary: { label: "Steps", value: "10,420" },
        secondary: [{ label: "Move", value: "520 kcal" }],
      },
    });

    const sectionMatch = body.match(/<section[^>]*aria-labelledby="([^"]+)"/);
    expect(sectionMatch).not.toBeNull();
    const headingId = sectionMatch![1];
    expect(body).toMatch(
      new RegExp(`<h2[^>]*id="${headingId}"[^>]*>Daily summary</h2>`),
    );
  });
});
