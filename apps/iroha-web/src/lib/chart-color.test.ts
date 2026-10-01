import { describe, expect, it } from "vitest";
import { resolveColor } from "@iroha/shared/theme-ui/components/chart-color";

const styles = {
  getPropertyValue: (name: string) => (name === "--accent" ? " #296bcb " : ""),
};

describe("canvas chart paint resolution", () => {
  it("resolves a simple CSS token and trims its computed value", () => {
    expect(resolveColor("var(--accent)", styles, "#000")).toBe("#296bcb");
  });
  it("preserves literal paint values", () => {
    expect(resolveColor("#39c5bb", styles, "#000")).toBe("#39c5bb");
    expect(resolveColor("rgb(10, 20, 30)", styles, "#000")).toBe(
      "rgb(10, 20, 30)",
    );
  });
  it("uses the supplied fallback for absent or unresolved simple tokens", () => {
    expect(resolveColor(undefined, styles, "#000")).toBe("#000");
    expect(resolveColor("", styles, "#000")).toBe("#000");
    expect(resolveColor("var(--missing)", styles, "#000")).toBe("#000");
  });
});
