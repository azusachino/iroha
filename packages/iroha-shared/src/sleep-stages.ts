export interface SleepStageDefinition {
  label: string;
  color: string;
}

// Apple Health uses a stable vocabulary for classified sleep stages. Iroha
// keeps the source stage unchanged and gives unclassified sleep its own
// readable color instead of treating it as a nap or silently dropping it.
export const SLEEP_STAGE_DEFINITIONS: Readonly<
  Record<string, SleepStageDefinition>
> = {
  core: { label: "Core", color: "#4ca6a8" },
  deep: { label: "Deep", color: "#3d5fa8" },
  rem: { label: "REM", color: "#8b6fd1" },
  awake: { label: "Awake", color: "#ed9947" },
  in_bed: { label: "In bed", color: "#8792a3" },
  asleep_unspecified: { label: "Asleep (unspecified)", color: "#5c91c2" },
};

export function sleepStageLabel(stage: string): string {
  return SLEEP_STAGE_DEFINITIONS[stage]?.label ?? stage.replaceAll("_", " ");
}

export function sleepStageColor(stage: string): string {
  return (
    SLEEP_STAGE_DEFINITIONS[stage]?.color ??
    SLEEP_STAGE_DEFINITIONS.asleep_unspecified.color
  );
}
