import { describe, expect, it } from "vitest";
import { heroTitleFontSize, type HeroTitleClamp } from "./hero-title";

const base: HeroTitleClamp = { minRem: 1.8, vw: 5, maxRem: 3.4 };

describe("heroTitleFontSize", () => {
  it("uses the theme's own clamp unscaled for a normal-length title", () => {
    expect(heroTitleFontSize("Rick and Morty Season 9", base)).toBe(
      "clamp(1.80rem, 5.00vw, 3.40rem)",
    );
  });

  it("scales the clamp down as title length grows", () => {
    const short = heroTitleFontSize("A".repeat(20), base);
    const medium = heroTitleFontSize("A".repeat(45), base);
    const long = heroTitleFontSize("A".repeat(65), base);
    const veryLong = heroTitleFontSize("A".repeat(90), base);
    expect(short).toBe("clamp(1.80rem, 5.00vw, 3.40rem)");
    expect(medium).toBe("clamp(1.26rem, 3.50vw, 2.38rem)");
    expect(long).toBe("clamp(0.99rem, 2.75vw, 1.87rem)");
    expect(veryLong).toBe("clamp(0.76rem, 2.10vw, 1.43rem)");
  });

  it("handles a real 100+ character prod title without a huge result", () => {
    const realTitle =
      "異世界グルメで成り上がり無双～山に追放されたので、のんびりキャンプを楽しんでいたらいつの間にか強くなっていて、王侯貴族や実力者たちが俺を放っておいてくれません。一方、俺を追放した貴族たちは破滅が始まる～";
    expect(realTitle.length).toBeGreaterThan(80);
    expect(heroTitleFontSize(realTitle, base)).toBe(
      "clamp(0.76rem, 2.10vw, 1.43rem)",
    );
  });
});
