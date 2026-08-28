import type { Component } from "svelte";
import { Bike, Footprints, Mountain, Waves, Zap, Activity } from "@lucide/svelte";
import { canonicalSport } from "./sport-labels";

const SPORT_ICONS: Record<string, Component<any>> = {
  run: Zap,
  walk: Footprints,
  hike: Mountain,
  ride: Bike,
  swim: Waves,
  other: Activity,
};

export function sportIcon(sport: string): Component<any> | undefined {
  return SPORT_ICONS[canonicalSport(sport)];
}
