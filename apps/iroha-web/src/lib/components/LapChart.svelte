<script lang="ts">
  import type { Lap } from "$lib/api";
  import { formatDuration, formatPace } from "$lib/format";
  import BarChart from "@iroha/shared/theme-ui/components/BarChart.svelte";

  let {
    laps,
    swimming = false,
  }: {
    laps: Lap[];
    swimming?: boolean;
  } = $props();

  const paceValues = $derived(
    laps.map((lap) => {
      const value = lap.avg_pace_s_per_km;
      return value != null && Number.isFinite(value) && value > 0
        ? value
        : null;
    }),
  );
  const durationValues = $derived(
    laps.map((lap) => {
      const value = lap.duration_s;
      return value != null && Number.isFinite(value) && value > 0
        ? value
        : null;
    }),
  );
  const usePace = $derived(paceValues.some((value) => value != null));
  const values = $derived(usePace ? paceValues : durationValues);
  const label = $derived(
    usePace ? (swimming ? "Pace / 100m" : "Pace / km") : "Duration",
  );
  const categories = $derived(laps.map((lap) => `Lap ${lap.lap_no}`));

  function formatValue(value: number): string {
    if (usePace) {
      if (swimming) {
        const minutes = Math.floor(value / 60);
        return `${minutes}:${String(Math.round(value % 60)).padStart(2, "0")} /100m`;
      }
      return formatPace(value);
    }
    return formatDuration(value);
  }
</script>

<section class="lap-chart" aria-label={`${label} by lap`}>
  {#if values.some((value) => value != null)}
    <BarChart
      {categories}
      primary={{
        name: label,
        values,
        color: "var(--accent)",
        formatter: formatValue,
      }}
      categorical
      height={250}
    />
  {:else}
    <p class="empty">
      No measured pace or duration was recorded for these laps.
    </p>
  {/if}
</section>

<style>
  .lap-chart {
    min-width: 0;
    padding: 0.25rem 0;
  }

  .empty {
    margin: 0;
    color: var(--text-muted);
    font-size: 0.8rem;
  }
</style>
