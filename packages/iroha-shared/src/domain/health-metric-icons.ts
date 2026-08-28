import type { LucideIcon } from "@lucide/svelte";
import {
  ChevronsUp,
  Droplet,
  Footprints,
  Gauge,
  Heart,
  HeartPulse,
  Route,
  Scale,
  Waves,
  Wind,
} from "@lucide/svelte";

// Same key space as health-metric-labels.ts. Kept as a separate module so a
// consumer that only needs the label (e.g. CSV export) never pulls in
// @lucide/svelte.
const HEALTH_METRIC_ICONS: Record<string, LucideIcon> = {
  steps: Footprints,
  distance_km: Route,
  flights: ChevronsUp,
  resting_hr: Heart,
  walking_hr_avg: HeartPulse,
  hrv_sdnn: Waves,
  spo2_avg: Droplet,
  spo2_min: Droplet,
  respiratory_rate: Wind,
  vo2max: Gauge,
  body_mass_kg: Scale,
};

export function healthMetricIcon(metric: string): LucideIcon | undefined {
  return HEALTH_METRIC_ICONS[metric];
}
