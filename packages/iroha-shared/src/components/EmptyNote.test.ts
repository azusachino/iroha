import { createRawSnippet } from "svelte";
import { render } from "svelte/server";
import { describe, expect, it } from "vitest";
import EmptyNote from "./EmptyNote.svelte";

describe("EmptyNote", () => {
  it("renders a text prop as a muted empty-note paragraph", () => {
    const { body } = render(EmptyNote, {
      props: {
        text: "No records yet for night sessions.",
      },
    });

    expect(body).toContain("empty-note");
    expect(body).toContain("No records yet for night sessions.");
  });

  it("renders children snippet when provided", () => {
    const children = createRawSnippet(() => ({
      render: () => `<span>Custom empty state</span>`,
    }));
    const { body } = render(EmptyNote, {
      props: {
        children,
      },
    });

    expect(body).toContain("empty-note");
    expect(body).toContain("<span>Custom empty state</span>");
  });
});
