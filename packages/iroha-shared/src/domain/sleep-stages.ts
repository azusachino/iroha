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
  core: { label: "Core", color: "var(--sleep-core)" },
  deep: { label: "Deep", color: "var(--sleep-deep)" },
  rem: { label: "REM", color: "var(--sleep-rem)" },
  awake: { label: "Awake", color: "var(--sleep-awake)" },
  in_bed: { label: "In bed", color: "var(--sleep-in-bed)" },
  asleep_unspecified: { label: "Asleep (unspecified)", color: "var(--sleep-unspecified)" },
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
