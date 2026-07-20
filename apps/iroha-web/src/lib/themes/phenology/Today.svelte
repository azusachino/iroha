<script lang="ts">
  import type { Activity, DailyRow, SleepSession } from "$lib/api";
  import ThemeToday from "$lib/themes/ThemeToday.svelte";

  let {
    dayLabel,
    day,
    dRow,
    mainNight,
    acts,
  }: {
    dayLabel: string;
    day: string;
    dRow: DailyRow | undefined;
    mainNight: SleepSession | undefined;
    acts: Activity[];
  } = $props();
</script>

<ThemeToday variant="phenology" {dayLabel} {day} {dRow} {mainNight} {acts} />
