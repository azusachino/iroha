import { describe, expect, it } from "vitest";
import { axisRange, formatAxisTick } from "@iroha/shared/format/axis";

describe("shared quantity axes", () => {
  for (const values of [
    [],
    [null, NaN, Infinity],
    [0],
    [7, 7],
    [-7, -7],
    [-12, 3],
    [0.00000003, 0.00000005],
    [1e12, 3e12],
    [12345, null, 0],
  ]) {
    it(`keeps finite bounds and distinct nice ticks for ${JSON.stringify(values)}`, () => {
      const range = axisRange(values);
      expect(Number.isFinite(range.min)).toBe(true);
      expect(Number.isFinite(range.max)).toBe(true);
      expect(Number.isFinite(range.interval)).toBe(true);
      expect(range.max).toBeGreaterThan(range.min);
      const mantissa =
        range.interval / 10 ** Math.floor(Math.log10(range.interval));
      expect([1, 2, 5].some((nice) => Math.abs(nice - mantissa) < 1e-8)).toBe(
        true,
      );
      for (const value of values)
        if (value != null && Number.isFinite(value)) {
          expect(value).toBeGreaterThanOrEqual(range.min);
          expect(value).toBeLessThanOrEqual(range.max);
        }
      const ticks = Array.from(
        { length: Math.round((range.max - range.min) / range.interval) + 1 },
        (_, index) => range.min + index * range.interval,
      );
      expect(ticks.every(Number.isFinite)).toBe(true);
      expect(
        ticks.every((tick, index) => index === 0 || tick > ticks[index - 1]),
      ).toBe(true);
      const labels = ticks.map((tick) => formatAxisTick(tick, range.interval));
      expect(new Set(labels).size).toBe(labels.length);
    });
  }
  it("keeps integer observations and explicit minor-unit presentation precise", () => {
    expect(axisRange([1], 1)).toEqual({ min: 0, max: 1, interval: 1 });
    expect(formatAxisTick(123400, 10000, 0.01)).toBe("1,234");
    expect(formatAxisTick(1, 1, 0.01)).toBe("0.01");
    expect(formatAxisTick(-0, 1)).toBe("0");
    expect(formatAxisTick(NaN, 1)).toBe("—");
    expect(axisRange([null, 12345]).max).toBeGreaterThanOrEqual(12345);
    expect(axisRange([Number.MAX_VALUE]).max).toBe(Number.MAX_VALUE);
    expect(axisRange([-Number.MAX_VALUE]).min).toBe(-Number.MAX_VALUE);
  });
});
