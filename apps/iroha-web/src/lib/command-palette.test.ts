import { describe, expect, it } from "vitest";
import {
  buildPaletteCommands,
  filterPaletteCommands,
  nextSelectedIndex,
} from "./command-palette";

describe("command palette catalog and filtering", () => {
  const commands = buildPaletteCommands();

  it("includes cockpit routes and quick actions without duplicate destinations", () => {
    expect(commands.length).toBe(13);
    expect(commands.some((cmd) => cmd.id === "action-cycle-theme")).toBe(true);
    expect(commands.some((cmd) => cmd.label === "Open public showcase")).toBe(
      true,
    );
    const hrefs = commands
      .map((cmd) => (cmd.action.type === "navigate" ? cmd.action.href : null))
      .filter((href): href is string => href !== null);
    expect(new Set(hrefs).size).toBe(hrefs.length);
  });

  it("returns all commands when query is empty or whitespace", () => {
    expect(filterPaletteCommands(commands, "")).toHaveLength(commands.length);
    expect(filterPaletteCommands(commands, "   ")).toHaveLength(
      commands.length,
    );
  });

  it("matches by label, hint, and keywords case-insensitively", () => {
    const themeMatches = filterPaletteCommands(commands, "DARK");
    expect(themeMatches.map((cmd) => cmd.id)).toEqual(["action-cycle-theme"]);

    const showcaseMatches = filterPaletteCommands(commands, "miles margins");
    expect(showcaseMatches.map((cmd) => cmd.id)).toEqual(["public-archive"]);

    const sleepMatches = filterPaletteCommands(commands, "sleep");
    expect(sleepMatches.map((cmd) => cmd.id)).toEqual(["night"]);
  });

  it("returns an empty array when no command matches", () => {
    expect(filterPaletteCommands(commands, "nonexistent-xyz")).toEqual([]);
  });

  it("wraps selection index cleanly across list boundaries", () => {
    expect(nextSelectedIndex(0, 1, 3)).toBe(1);
    expect(nextSelectedIndex(2, 1, 3)).toBe(0);
    expect(nextSelectedIndex(0, -1, 3)).toBe(2);
    expect(nextSelectedIndex(0, 1, 0)).toBe(-1);
    expect(nextSelectedIndex(-1, 1, 3)).toBe(0);
    expect(nextSelectedIndex(-1, -1, 3)).toBe(2);
  });
});
