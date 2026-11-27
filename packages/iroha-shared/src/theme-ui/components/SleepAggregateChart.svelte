<script lang="ts">
  import type { DesignLanguage } from "../../theme/themes";
  import type { SleepAggregateBucket } from "../../domain/sleep";
  import EmptyNote from "../../components/EmptyNote.svelte";
  import MetricTable from "../../components/MetricTable.svelte";
  import PanelFrame from "../../components/PanelFrame.svelte";
  import BarChart from "./BarChart.svelte";
  import { formatDuration, formatCanonicalMonth } from "../../format/format";

  let {
    buckets,
    granularity,
    scope = "",
    theme,
  }: {
    buckets: SleepAggregateBucket[];
    granularity: "month" | "year";
    scope?: string;
    theme: DesignLanguage;
  } = $props();

  const ROLLUP_CHART_MIN_POINTS = 3;

  function labelFor(period: string): string {
    return granularity === "year"
      ? period.slice(0, 4)
      : formatCanonicalMonth(period);
  }

  const categories = $derived(buckets.map((bucket) => labelFor(bucket.period)));
  const averageAsleep = $derived(
    buckets.map((bucket) =>
      bucket.main_sleep_count > 0 ? bucket.average_asleep_s : null,
    ),
  );
  const mainSleepCount = $derived(
    buckets.map((bucket) => bucket.main_sleep_count),
  );
  const napCount = $derived(buckets.map((bucket) => bucket.nap_count));
</script>

<div class="sleep-aggregate-chart" data-theme={theme}>
  <PanelFrame title="Sleep rhythm over time">
    {#snippet meta()}
      <span>
        {buckets.length} {buckets.length === 1 ? "period" : "periods"}
        {scope ? ` · ${scope}` : ""}
      </span>
    {/snippet}

    {#if buckets.length >= ROLLUP_CHART_MIN_POINTS}
      <div class="chart-grid">
        <div>
          <p class="chart-label">Average main sleep</p>
          <BarChart
            {categories}
            primary={{
              name: "Average main sleep",
              values: averageAsleep,
              color: "var(--accent)",
              formatter: (value) => formatDuration(value),
            }}
            primaryType="line"
            height={290}
          />
        </div>
        <div>
          <p class="chart-label">Main sleep and naps</p>
          <BarChart
            {categories}
            primary={{
              name: "Main sleep",
              values: mainSleepCount,
              color: "var(--accent)",
            }}
            secondary={{
              name: "Naps",
              values: napCount,
              color: "var(--accent-2)",
            }}
            height={290}
          />
        </div>
      </div>
    {:else}
      <EmptyNote
        text={`Only ${buckets.length} ${granularity} ${buckets.length === 1 ? "period" : "periods"} in this scope — chart requires at least ${ROLLUP_CHART_MIN_POINTS} periods; exact values are below.`}
      />
    {/if}

    <MetricTable
      ariaLabel="Exact rollup values"
      caption="Exact rollup values"
      minWidth="36rem"
      nowrap
      numericAlign
      dense
    >
      <thead>
        <tr>
          <th>Period</th>
          <th>Sessions</th>
          <th>Main sleep</th>
          <th>Naps</th>
          <th>Wake dates</th>
          <th>Avg main sleep</th>
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
    </MetricTable>
  </PanelFrame>
</div>

<style>
  .sleep-aggregate-chart {
    min-width: 0;
  }

  .chart-label {
    margin: 0 0 var(--space-1);
    color: var(--accent);
    font-size: var(--type-caption);
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .chart-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--gap-grid);
  }

  @media (max-width: 768px) {
    .chart-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
