// Same key space as health-metric-labels.ts. Only metrics with a real,
// citable, general-adult-population reference range appear here --
// steps/distance_km/flights/walking_hr_avg/hrv_sdnn/vo2max/body_mass_kg are
// deliberately absent, not an oversight: there is no official step-count
// target (WHO's 2020 guidance is stated in activity-minutes, not steps),
// HRV and VO2max need age/sex percentile bands this app's data model
// doesn't carry (population reference distributions, not a clinical
// cutoff), body_mass_kg needs height (BMI) to mean anything, and
// distance_km/flights have no clinical guidance at all. Do not add a range
// for any of these without a real citation -- a fabricated "healthy range"
// is worse than none.
export interface HealthMetricRange {
  min: number;
  max: number;
  source: string;
  caveat?: string;
}

const HEALTH_METRIC_RANGES: Record<string, HealthMetricRange> = {
  resting_hr: {
    min: 60,
    max: 100,
    source: "American Heart Association — Target Heart Rates Chart",
    caveat: "Trained athletes may run lower.",
  },
  spo2_avg: {
    min: 95,
    max: 100,
    source: "Clinical consensus (pulse oximetry)",
    caveat:
      "This app shows the whole sub-95% band as below normal rather than distinguishing severity -- clinically, readings below 92% are more urgently concerning than 93-94%. Altitude above 3000m, age over 70, and consumer pulse-ox error (±2-3%) all shift this.",
  },
  spo2_min: {
    min: 95,
    max: 100,
    source: "Clinical consensus (pulse oximetry)",
    caveat:
      "This app shows the whole sub-95% band as below normal rather than distinguishing severity -- clinically, readings below 92% are more urgently concerning than 93-94%. Altitude above 3000m, age over 70, and consumer pulse-ox error (±2-3%) all shift this.",
  },
  respiratory_rate: {
    min: 12,
    max: 20,
    source: "American Lung Association",
    caveat: "Other clinical sources cite tighter bands (12-16 or 12-18).",
  },
};

export function healthMetricRange(metric: string): HealthMetricRange | undefined {
  return HEALTH_METRIC_RANGES[metric];
}
