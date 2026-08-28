// Daily-health metrics (packages/iroha-shared/src/domain/daily.ts) are keyed
// by their raw wearable-import field name -- fine as a data key, unreadable
// as a chart/table label. This is the one place that mapping lives so every
// theme's Reports.svelte shows the same human label instead of re-deriving
// or hand-writing its own.
const HEALTH_METRIC_LABELS: Record<string, string> = {
  steps: "Steps",
  distance_km: "Walking distance",
  flights: "Flights climbed",
  resting_hr: "Resting HR",
  walking_hr_avg: "Walking HR",
  hrv_sdnn: "HRV",
  spo2_avg: "SpO₂ avg",
  spo2_min: "SpO₂ min",
  respiratory_rate: "Respiratory rate",
  vo2max: "VO₂ max",
  body_mass_kg: "Body mass",
};

export function healthMetricLabel(metric: string): string {
  return HEALTH_METRIC_LABELS[metric] ?? metric;
}
