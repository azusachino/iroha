import type { LucideIcon } from "@lucide/svelte";
import { Bike, Footprints, Mountain, Waves, Zap, Activity } from "@lucide/svelte";
import { canonicalSport } from "./sport";

const SPORT_ICONS: Record<string, LucideIcon> = {
  run: Zap,
  walk: Footprints,
  hike: Mountain,
  ride: Bike,
  swim: Waves,
  other: Activity,
};

export function sportIcon(sport?: string | null): LucideIcon | undefined {
  return SPORT_ICONS[canonicalSport(sport)];
}
