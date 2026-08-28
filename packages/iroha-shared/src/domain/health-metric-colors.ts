// Same key space as health-metric-labels.ts/health-metric-icons.ts. Assigns
// each metric a *stable* slot from the existing per-theme categorical
// palette (the same set BarChart.svelte's own categoricalColors fallback
// draws from) -- keyed by metric identity, not array position, so a
// metric's color doesn't change just because the API returned it in a
// different order. Reuses each theme's own --accent/--ring-*/--mark-*
// tokens rather than inventing a new, theme-invariant health palette, so
// the coverage view stays in the active language's own color voice.
const HEALTH_METRIC_COLOR_VARS: Record<string, string> = {
  resting_hr: "--ring-move",
  walking_hr_avg: "--mark-amber",
  hrv_sdnn: "--accent",
  spo2_avg: "--ring-stand",
  spo2_min: "--sport-swim",
  respiratory_rate: "--accent-2",
  vo2max: "--ring-exercise",
  steps: "--ring-move",
  distance_km: "--mark-amber",
  flights: "--accent",
  body_mass_kg: "--ring-stand",
};

export function healthMetricColorVar(metric: string): string {
  return HEALTH_METRIC_COLOR_VARS[metric] ?? "--accent";
}
