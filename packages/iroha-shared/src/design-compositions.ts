export const DESIGN_COMPOSITIONS = [
  {
    id: "editorial",
    index: "A",
    label: "Editorial",
    intent: "Read the day as a composed story.",
  },
  {
    id: "command",
    index: "B",
    label: "Command center",
    intent: "Keep the important signals visible and actionable.",
  },
  {
    id: "chronicle",
    index: "C",
    label: "Chronicle",
    intent: "Follow the day as a lived sequence of events.",
  },
  {
    id: "cover",
    index: "D",
    label: "Cover page",
    intent: "Give one day a strong, legible visual identity.",
  },
  {
    id: "workspace",
    index: "E",
    label: "Personal OS",
    intent: "Arrange the day as a calm working surface.",
  },
  {
    id: "journal",
    index: "F",
    label: "Field journal",
    intent: "Preserve observations with continuity and provenance.",
  },
  {
    id: "quiet",
    index: "G",
    label: "Quiet",
    intent: "Notice a few meaningful signals without dashboard noise.",
  },
] as const;

export type DesignCompositionId = (typeof DESIGN_COMPOSITIONS)[number]["id"];

export type DesignRing = {
  move_kcal?: number;
  move_goal_kcal?: number;
  exercise_min?: number;
  exercise_goal_min?: number;
  stand_hours?: number;
  stand_goal_hours?: number;
};

export type DesignDaily = {
  ring?: DesignRing | null;
  steps?: number;
  distance_km?: number;
  resting_hr?: number;
  hrv_sdnn?: number;
  spo2_avg?: number;
  vo2max?: number;
};

export type DesignSleep = {
  asleep_s: number;
  efficiency: number;
  deep_s: number;
};

export type DesignActivity = {
  id: string;
  sport_type: string;
  title: string;
  started_at: string;
  distance_m?: number;
  moving_time_s?: number;
  duration_s?: number;
  avg_hr?: number;
  avg_pace_s_per_km?: number;
};

export type DesignMediaEvent = {
  id: string;
  title: string;
  progress_percent?: number;
};

export type DesignTodayData = {
  date: string;
  daily?: DesignDaily;
  sleep?: DesignSleep;
  activities: DesignActivity[];
  media: DesignMediaEvent[];
};

export type DesignCompositionLinks = {
  motion: string;
  night: string;
  patterns: string;
  library: string;
  activity: (id: string) => string;
};

export type DesignCompositionProps = {
  today: DesignTodayData;
  readiness: number;
  links: DesignCompositionLinks;
};

export function designDateLabel(value: string): string {
  return value.slice(0, 10);
}

export function designTimeLabel(value: string): string {
  return value.slice(11, 16);
}

export function designDuration(seconds?: number | null): string {
  if (seconds == null || !Number.isFinite(seconds)) return "—";
  const minutes = Math.round(seconds / 60);
  return `${Math.floor(minutes / 60)}h ${String(minutes % 60).padStart(2, "0")}m`;
}

export function designDistance(meters?: number | null): string {
  if (meters == null || !Number.isFinite(meters)) return "—";
  return meters >= 1000
    ? `${(meters / 1000).toFixed(1)} km`
    : `${Math.round(meters)} m`;
}

export function designPercent(value?: number | null): string {
  if (value == null || !Number.isFinite(value)) return "—";
  return `${Math.round(Math.max(0, Math.min(100, value)))}%`;
}

export function designSportLabel(value: string): string {
  return value.replaceAll("_", " ").replace(/^./, (char) => char.toUpperCase());
}

export function designActivitySummary(activity: DesignActivity): string {
  return [
    designDistance(activity.distance_m),
    designDuration(activity.moving_time_s ?? activity.duration_s),
    activity.avg_pace_s_per_km != null
      ? `${Math.floor(activity.avg_pace_s_per_km / 60)}:${String(
          activity.avg_pace_s_per_km % 60,
        ).padStart(2, "0")} /km`
      : null,
  ]
    .filter(Boolean)
    .join(" · ");
}

export function isDesignComposition(
  value: string | null | undefined,
): value is DesignCompositionId {
  return DESIGN_COMPOSITIONS.some((composition) => composition.id === value);
}

export function designComposition(
  value: string | null | undefined,
): DesignCompositionId {
  return isDesignComposition(value) ? value : "editorial";
}
