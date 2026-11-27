import { createRawSnippet } from "svelte";
import { render } from "svelte/server";
import { describe, expect, it } from "vitest";
import PanelFrame from "./PanelFrame.svelte";

const bodyText = createRawSnippet(() => ({
  render: () => `<p>Card body</p>`,
}));

describe("PanelFrame", () => {
  it("renders a single title header and names the section via aria-labelledby", () => {
    const { body } = render(PanelFrame, {
      props: {
        title: "Monthly distance",
        children: bodyText,
      },
    });

    const sectionMatch = body.match(/<section[^>]*aria-labelledby="([^"]+)"/);
    expect(sectionMatch).not.toBeNull();
    const headingId = sectionMatch![1];
    expect(body).toMatch(
      new RegExp(`<h2[^>]*id="${headingId}"[^>]*>Monthly distance</h2>`),
    );
    expect(body).toContain("Card body");
  });

  it("renders the optional meta slot on the right of the header", () => {
    const meta = createRawSnippet(() => ({
      render: () => `<span>Last 12 months</span>`,
    }));
    const { body } = render(PanelFrame, {
      props: {
        title: "Asleep time by night",
        meta,
        children: bodyText,
      },
    });

    expect(body).toContain("panel-frame-meta");
    expect(body).toContain("Last 12 months");
  });

  it("supports headingLevel 3 when nested under an h2 section", () => {
    const { body } = render(PanelFrame, {
      props: {
        title: "Browse entries",
        headingLevel: 3,
        children: bodyText,
      },
    });

    expect(body).toMatch(/<h3[^>]*>Browse entries<\/h3>/);
    expect(body).not.toContain("<h2");
  });

  it("preserves aria-label framing when title is omitted", () => {
    const { body } = render(PanelFrame, {
      props: {
        label: "Canonical ledger",
        children: bodyText,
      },
    });

    expect(body).toMatch(/<section[^>]*aria-label="Canonical ledger"/);
    expect(body).not.toContain("panel-frame-header");
  });
});
