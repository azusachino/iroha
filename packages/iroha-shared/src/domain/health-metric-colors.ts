// Same key space as health-metric-labels.ts/health-metric-icons.ts. Assigns
// each metric a stable shared palette slot, keyed by metric identity rather
// than API order. Data colors do not borrow a host's interactive accent.
const HEALTH_METRIC_COLOR_VARS: Record<string, string> = {
  resting_hr: "--ring-move",
  walking_hr_avg: "--mark-amber",
  hrv_sdnn: "--ring-stand",
  spo2_avg: "--ring-stand",
  spo2_min: "--sport-swim",
  respiratory_rate: "--ring-move",
  vo2max: "--ring-exercise",
  steps: "--ring-move",
  distance_km: "--mark-amber",
  flights: "--ring-stand",
  body_mass_kg: "--ring-stand",
};

export function healthMetricColorVar(metric: string): string {
  return HEALTH_METRIC_COLOR_VARS[metric] ?? "--text-muted";
}
