import type { Snippet } from "svelte";
import type { DailyRow } from "./daily";
import type { DesignLanguage } from "./themes";

export interface ActivityRing {
  label: string;
  value: number;
  goal: number;
  unit: string;
  color: string;
}

export interface DailyPeriod {
  label: string;
  period: string;
  days: number | null;
  move: number | null;
  exercise: number | null;
  stand: number | null;
  moveClosedPct: number | null;
  steps: number | null;
  distance: number | null;
  resting_hr: number | null;
  hrv_sdnn: number | null;
  spo2_avg?: number | null;
  respiratory_rate?: number | null;
  vo2max?: number | null;
  body_mass_kg?: number | null;
}

export type DailyThemeProps = {
  chrono: DailyPeriod[];
  gran: "day" | "month" | "year";
  onGran: (value: "day" | "month" | "year") => void;
  onDrillIndex: (index: number) => void;
  onDrillPeriod: (period: string) => void;
  ringData: ActivityRing[];
  latestRingDay: DailyRow | null;
  theme: DesignLanguage;
  children?: Snippet;
};
