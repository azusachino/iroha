<script lang="ts">
  import type { Snippet } from "svelte";
  import ThemeToday from "$lib/themes/ThemeToday.svelte";
  import type { Activity, DailyRow, SleepSession } from "$lib/api";

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

<ThemeToday variant="atlas" {dayLabel} {day} {dRow} {mainNight} {acts} />
