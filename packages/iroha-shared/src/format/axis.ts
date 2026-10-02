const DEFAULT_INTERVALS = 3;
const NICE_FACTORS = [1, 2, 5, 10];

// Zero is the reference baseline; missing/non-finite values never become data.
export function axisRange(values: readonly (number | null | undefined)[], minimumInterval = 0) {
  let min = 0;
  let max = 0;
  for (const value of values) {
    if (value == null || !Number.isFinite(value)) continue;
    min = Math.min(min, value);
    max = Math.max(max, value);
  }
  if (min === max) max = Math.max(1, minimumInterval);
  const desired = Math.max(max / DEFAULT_INTERVALS - min / DEFAULT_INTERVALS, minimumInterval);
  const power = 10 ** Math.floor(Math.log10(desired)) || Number.MIN_VALUE;
  const factor = NICE_FACTORS.find((factor) => factor * power >= desired) ?? 10;
  const interval = Math.min(factor * power, Number.MAX_VALUE);
  const lower = Math.floor(min / interval) * interval;
  const upper = Math.ceil(max / interval) * interval;
  return {
    min: Number.isFinite(lower) ? lower : min,
    max: Number.isFinite(upper) ? upper : max,
    interval,
  };
}

// Axis-only precision. Exact tables/tooltips continue using their own formatter.
export function formatAxisTick(value: number, interval: number, scale = 1): string {
  const displayed = value * scale;
  const step = interval * scale;
  if (!Number.isFinite(displayed)) return "—";
  if (displayed === 0) return "0";
  const scientific = step < 0.000001 || Math.abs(displayed) >= 1e12;
  return new Intl.NumberFormat("en-US", scientific
    ? { notation: "scientific", maximumSignificantDigits: 15 }
    : { maximumFractionDigits: Math.min(20, Math.max(0, Math.ceil(-Math.log10(step)))) }
  ).format(Object.is(displayed, -0) ? 0 : displayed);
}
