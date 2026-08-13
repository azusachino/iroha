<script lang="ts">
  import type { SleepAggregateBucket } from "$lib/api";
  import BarChart from "$lib/components/BarChart.svelte";
  import { formatDuration, formatMonth } from "$lib/format";
  import { useTheme } from "$lib/themes/context.svelte";

  let {
    buckets,
    granularity,
    scope = "",
  }: {
    buckets: SleepAggregateBucket[];
    granularity: "month" | "year";
    scope?: string;
  } = $props();

  const theme = useTheme();

  function labelFor(period: string): string {
    return granularity === "year" ? period.slice(0, 4) : formatMonth(period);
  }

  const categories = $derived(buckets.map((bucket) => labelFor(bucket.period)));
  const averageAsleep = $derived(
    buckets.map((bucket) =>
      bucket.main_sleep_count > 0 ? bucket.average_asleep_s : null,
    ),
  );
  const sessionCount = $derived(buckets.map((bucket) => bucket.session_count));
</script>

<section
  class="sleep-aggregate-chart"
  data-theme={theme.definition().identity.id}
  aria-labelledby="sleep-rollup-title"
>
  <header class="chart-header">
    <div>
      <p class="eyebrow">Complete {granularity} rollup</p>
      <h2 id="sleep-rollup-title">Sleep rhythm over time</h2>
      <p class="description">
        Average main-sleep duration and total recorded sessions from the full
        canonical dataset{scope ? ` · ${scope}` : ""}.
      </p>
    </div>
    <span class="coverage">{buckets.length} periods</span>
  </header>

  <div class="chart-grid">
    <div>
      <p class="chart-label">Average asleep</p>
      <BarChart
        {categories}
        primary={{
          name: "Average asleep",
          values: averageAsleep,
          color: "var(--accent)",
          formatter: (value) => formatDuration(value),
        }}
        primaryType="line"
        height={290}
      />
    </div>
    <div>
      <p class="chart-label">Sessions recorded</p>
      <BarChart
        {categories}
        primary={{
          name: "Sessions",
          values: sessionCount,
          color: "var(--accent-2)",
        }}
        categorical
        height={290}
      />
    </div>
  </div>

  <div class="table-scroll">
    <table>
      <caption>Exact rollup values</caption>
      <thead>
        <tr>
          <th>Period</th>
          <th>Sessions</th>
          <th>Main sleep</th>
          <th>Naps</th>
          <th>Wake dates</th>
          <th>Avg asleep</th>
        </tr>
      </thead>
      <tbody>
        {#each buckets as bucket (bucket.period)}
          <tr>
            <td>{labelFor(bucket.period)}</td>
            <td>{bucket.session_count}</td>
            <td>{bucket.main_sleep_count}</td>
            <td>{bucket.nap_count}</td>
            <td>{bucket.observed_wake_dates}</td>
            <td>
              {bucket.main_sleep_count > 0
                ? formatDuration(bucket.average_asleep_s)
                : "—"}
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</section>

<style>
  .sleep-aggregate-chart {
    display: grid;
    gap: 1rem;
    padding: clamp(1rem, 2.5vw, 1.5rem);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
  }

  .sleep-aggregate-chart[data-theme="atlas"] {
    border-width: 2px;
    border-radius: 2px;
    background-image:
      linear-gradient(
        color-mix(in srgb, var(--accent) 8%, transparent) 1px,
        transparent 1px
      ),
      linear-gradient(
        90deg,
        color-mix(in srgb, var(--accent) 8%, transparent) 1px,
        transparent 1px
      );
    background-size: 14px 14px;
  }

  .sleep-aggregate-chart[data-theme="field-journal"] {
    border-style: dashed;
    border-radius: 0;
  }

  .sleep-aggregate-chart[data-theme="phenology"] {
    border-radius: 1.2rem;
  }

  .sleep-aggregate-chart[data-theme="sound-map"] {
    border-inline-width: 3px;
  }

  .sleep-aggregate-chart[data-theme="archive"] {
    border-width: 3px;
    border-radius: 0;
  }

  .sleep-aggregate-chart[data-theme="grapher"] {
    border-radius: 2px;
    border-bottom-width: 3px;
  }

  .chart-header,
  .chart-header h2,
  .chart-header p {
    margin: 0;
  }

  .chart-header {
    display: flex;
    align-items: start;
    justify-content: space-between;
    gap: 1rem;
  }

  .eyebrow,
  .chart-label {
    color: var(--accent);
    font-size: 0.66rem;
    font-weight: 750;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  .eyebrow {
    margin-bottom: 0.35rem !important;
  }

  .chart-header h2 {
    font-size: clamp(1.25rem, 2.5vw, 1.8rem);
  }

  .description,
  .coverage {
    color: var(--text-muted);
    font-size: 0.78rem;
    line-height: 1.5;
  }

  .coverage {
    white-space: nowrap;
  }

  .chart-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 1rem;
  }

  .chart-label {
    margin: 0 0 0.25rem;
  }

  .table-scroll {
    max-height: 17rem;
    overflow: auto;
    border-top: 1px solid var(--border);
  }

  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.76rem;
  }

  caption {
    padding: 0.7rem 0 0.45rem;
    color: var(--text-muted);
    text-align: left;
  }

  th,
  td {
    padding: 0.5rem 0.4rem;
    border-bottom: 1px solid var(--border);
    text-align: right;
    white-space: nowrap;
  }

  th:first-child,
  td:first-child {
    text-align: left;
  }

  th {
    color: var(--text-muted);
    font-size: 0.64rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  @media (max-width: 760px) {
    .chart-grid {
      grid-template-columns: 1fr;
    }

    .chart-header {
      display: grid;
      gap: 0.25rem;
    }
  }
</style>
