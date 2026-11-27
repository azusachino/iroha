import { createRawSnippet } from "svelte";
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
    expect(body).toContain("empty-note");
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

  it("renders optional controls in the band header", () => {
    const controls = createRawSnippet(() => ({
      render: () => `<button type="button">2026-08</button>`,
    }));
    const { body } = render(FocalBand, {
      props: {
        heading: "Monthly overview",
        controls,
        primary: { label: "Distance", value: "142.8", unit: "km" },
        secondary: [{ label: "Sessions", value: "12" }],
      },
    });

    expect(body).toContain("focal-header");
    expect(body).toContain("focal-controls");
    expect(body).toContain("<button type=\"button\">2026-08</button>");
  });

  it("omits the heading element when heading is omitted and names the region from primary.label", () => {
    const { body } = render(FocalBand, {
      props: {
        primary: { label: "Distance", value: "142.8", unit: "km" },
        secondary: [{ label: "Sessions", value: "12" }],
      },
    });

    expect(body).not.toContain("<h2");
    expect(body).not.toContain("aria-labelledby");
    expect(body).toMatch(/<section[^>]*aria-label="Distance"/);
  });

  it("renders a secondary item with href as an anchor with its accessible label", () => {
    const { body } = render(FocalBand, {
      props: {
        primary: { label: "Distance", value: "142.8", unit: "km" },
        secondary: [
          {
            label: "Sleep",
            value: "7h 15m",
            href: "/sleep",
            linkLabel: "Explore sleep →",
          },
        ],
      },
    });

    expect(body).toMatch(
      /<a[^>]*href="\/sleep"[^>]*aria-label="Explore sleep →"[^>]*>\s*Explore sleep →\s*<\/a>/,
    );
  });
});
