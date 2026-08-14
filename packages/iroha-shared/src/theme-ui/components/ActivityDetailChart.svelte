<script lang="ts">
  import { useTheme } from "../context.svelte";
  import FusedActivityChart from "./FusedActivityChart.svelte";

  let {
    xValues,
    xLabel,
    pace,
    heartRate,
    elevation,
    paceLabel,
    onHover,
  }: {
    xValues: number[];
    xLabel: string;
    pace: (number | null)[];
    heartRate: (number | null)[];
    elevation: (number | null)[];
    paceLabel: string;
    onHover?: (index: number | null) => void;
  } = $props();

  const theme = useTheme();
  const hasHeartRate = $derived(heartRate.some((value) => value != null));
  const hasMeasurements = $derived(
    [pace, heartRate, elevation].some((values) =>
      values.some((value) => value != null),
    ),
  );
</script>

{#if hasMeasurements}
  <section
    class="activity-detail-chart"
    data-theme={theme.definition().identity.id}
    aria-labelledby="activity-detail-chart-title"
  >
    <header>
      <div>
        <p class="chart-kicker">Recorded measurements</p>
        <h2 id="activity-detail-chart-title">
          {hasHeartRate
            ? "Heart rate across the record"
            : "Effort across the record"}
        </h2>
      </div>
      <span>{xLabel}</span>
    </header>
    <FusedActivityChart
      {xValues}
      {xLabel}
      {pace}
      {heartRate}
      {elevation}
      {paceLabel}
      {onHover}
    />
  </section>
{/if}

<style>
  .activity-detail-chart {
    display: grid;
    gap: 0.75rem;
    padding: 1rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
  }

  .activity-detail-chart > header {
    display: flex;
    align-items: end;
    justify-content: space-between;
    gap: 1rem;
  }

  .activity-detail-chart h2,
  .activity-detail-chart p {
    margin: 0;
  }

  .chart-kicker {
    color: var(--accent);
    font-size: 0.68rem;
    font-weight: 750;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  .activity-detail-chart h2 {
    margin-top: 0.3rem;
    font-size: 1.25rem;
  }

  .activity-detail-chart > header > span {
    color: var(--text-muted);
    font-size: 0.75rem;
  }

  .activity-detail-chart[data-theme="atlas"] {
    border-width: 2px;
    border-radius: 2px;
    background-image:
      linear-gradient(
        color-mix(in srgb, var(--accent) 7%, transparent) 1px,
        transparent 1px
      ),
      linear-gradient(
        90deg,
        color-mix(in srgb, var(--accent) 7%, transparent) 1px,
        transparent 1px
      );
    background-size: 12px 12px;
  }

  .activity-detail-chart[data-theme="field-journal"] {
    border-style: dashed;
  }

  .activity-detail-chart[data-theme="phenology"] {
    border-radius: 1.2rem;
  }

  .activity-detail-chart[data-theme="sound-map"] {
    border-inline-width: 3px;
  }

  .activity-detail-chart[data-theme="archive"] {
    border-width: 3px;
    border-style: double;
    border-radius: 0;
  }

  .activity-detail-chart[data-theme="grapher"] {
    border-radius: 2px;
    border-bottom-width: 3px;
  }

  @media (max-width: 560px) {
    .activity-detail-chart > header {
      align-items: flex-start;
      flex-direction: column;
    }
  }
</style>
