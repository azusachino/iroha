<script lang="ts">
  import type { DesignLanguage } from "../../theme/themes";
  import type { MetricSeriesResponse } from "../../components/metric-series";
  import BarChart from "./BarChart.svelte";
  import {
    formatCanonicalMonth,
    formatDistance,
    formatSport,
  } from "../../format/format";

  let {
    series,
    durationSeries = null,
    loading = false,
    error = null,
    scope = "",
    theme,
  }: {
    series?: MetricSeriesResponse | null;
    durationSeries?: MetricSeriesResponse | null;
    loading?: boolean;
    error?: string | null;
    scope?: string;
    theme: DesignLanguage;
  } = $props();

  type Point = { period: string; value: number | null; observed_days: number };

  function pointsFor(item: MetricSeriesResponse["series"][number]): Point[] {
    return item.points.map((point) => ({
      period: point.period,
      value: "value" in point ? (point.value ?? null) : null,
      observed_days: point.observed_days,
    }));
  }

  function aggregateValues(
    response: MetricSeriesResponse | null | undefined,
    periodsToRead: string[],
    divisor = 1,
  ): (number | null)[] {
    return periodsToRead.map((period) => {
      let total = 0;
      let observed = false;
      for (const item of response?.series ?? []) {
        const point = pointsFor(item).find(
          (candidate) => candidate.period === period,
        );
        if (point?.value != null) {
          total += point.value;
          observed = true;
        }
      }
      return observed ? total / divisor : null;
    });
  }

  function labelForPeriod(period: string): string {
    if (/^\d{4}-\d{2}$/.test(period)) return formatCanonicalMonth(period);
    return period;
  }

  const periods = $derived(
    series?.series[0]?.points.map((point) => point.period) ?? [],
  );
  const observedPeriodCount = $derived(
    new Set(
      (series?.series ?? []).flatMap((item) =>
        pointsFor(item)
          .filter((point) => point.value != null)
          .map((point) => point.period),
      ),
    ).size,
  );
  const expectedPeriodCount = $derived(
    Math.max(
      0,
      ...(series?.series ?? []).map((item) => item.coverage.expected_periods),
    ),
  );
  const totalDistance = $derived(aggregateValues(series, periods, 1000));
  const totalDuration = $derived(aggregateValues(durationSeries, periods));
  function observedDaysFor(
    response: MetricSeriesResponse | null | undefined,
    period: string,
  ): number {
    return (response?.series ?? []).reduce(
      (max, item) =>
        Math.max(
          max,
          pointsFor(item).find((point) => point.period === period)
            ?.observed_days ?? 0,
        ),
      0,
    );
  }
  const sportBreakdown = $derived(
    (series?.series ?? [])
      .map((item) => {
        const sport = item.dimensions.sport ?? "all";
        const total = pointsFor(item).reduce(
          (sum, point) => sum + (point.value ?? 0),
          0,
        );
        return { sport, total: total / 1000 };
      })
      .filter((item) => item.total > 0)
      .sort((left, right) => right.total - left.total),
  );
  const tableRows = $derived(
    periods.map((period, index) => ({
      period,
      value: totalDistance[index],
      duration: totalDuration[index],
      observedDays: observedDaysFor(series, period),
    })),
  );

  function formatKm(value: number): string {
    return formatDistance(value * 1000);
  }

  function formatTime(value: number): string {
    const seconds = Math.round(value);
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    return hours ? `${hours}h ${minutes}m` : `${minutes}m`;
  }
</script>

<section
  class="activity-metric-chart"
  data-theme={theme}
  aria-labelledby="activity-trend-title"
>
  <header class="chart-header">
    <div>
      <p class="eyebrow">Canonical movement series</p>
      <h2 id="activity-trend-title">Movement over time</h2>
      <p class="chart-description">
        Server-aggregated distance, grouped by {series?.period.grain ??
          "period"}
        {scope ? ` · ${scope}` : ""}. Missing periods remain empty.
      </p>
    </div>
    {#if series}
      <span class="coverage"
        >{observedPeriodCount} / {expectedPeriodCount} periods observed</span
      >
    {/if}
  </header>

  {#if loading && !series}
    <p class="chart-status">Building the canonical movement series…</p>
  {:else if error}
    <p class="chart-status error">{error}</p>
  {:else if !series || periods.length === 0}
    <p class="chart-status">No movement series is available for this scope.</p>
  {:else}
    <div class="chart-grid">
      <div class="trend-charts">
        <div class="trend-chart">
          <p class="eyebrow">Distance</p>
          <BarChart
            categories={periods.map(labelForPeriod)}
            primary={{
              name: "Distance",
              values: totalDistance,
              color: "var(--accent)",
              formatter: formatKm,
            }}
            primaryType="line"
            height={300}
            showDataTable={false}
          />
        </div>
        {#if durationSeries}
          <div class="trend-chart">
            <p class="eyebrow">Duration</p>
            <BarChart
              categories={periods.map(labelForPeriod)}
              primary={{
                name: "Duration",
                values: totalDuration,
                color: "var(--accent-2)",
                formatter: formatTime,
              }}
              primaryType="line"
              height={240}
              showDataTable={false}
            />
          </div>
        {/if}
      </div>
      {#if sportBreakdown.length > 1}
        <div class="sport-chart">
          <header>
            <p class="eyebrow">Sport composition</p>
            <h3>Where the distance came from</h3>
          </header>
          <BarChart
            categories={sportBreakdown.map((item) => formatSport(item.sport))}
            primary={{
              name: "Distance",
              values: sportBreakdown.map((item) => item.total),
              formatter: formatKm,
            }}
            orientation="horizontal"
            categorical
            height={Math.max(220, sportBreakdown.length * 42)}
            showDataTable={false}
          />
        </div>
      {/if}
    </div>

    <div class="series-table-wrap">
      <table>
        <caption>Exact movement series</caption>
        <thead
          ><tr
            ><th>Period</th><th>Distance</th><th>Duration</th><th
              >Observed days</th
            ></tr
          ></thead
        >
        <tbody>
          {#each tableRows as row (row.period)}
            <tr>
              <td>{labelForPeriod(row.period)}</td>
              <td>{row.value == null ? "—" : formatKm(row.value)}</td>
              <td>{row.duration == null ? "—" : formatTime(row.duration)}</td>
              <td>{row.observedDays || "—"}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</section>

<style>
  .activity-metric-chart {
    display: grid;
    gap: 1rem;
    min-width: 0;
    padding: clamp(1rem, 2.5vw, 1.5rem);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
  }

  .activity-metric-chart[data-theme="atlas"] {
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

  .activity-metric-chart[data-theme="field-journal"] {
    border-style: dashed;
    border-radius: 0;
  }

  .activity-metric-chart[data-theme="phenology"] {
    border-radius: 1.2rem;
  }

  .activity-metric-chart[data-theme="cadence"] {
    border-inline-width: 3px;
  }

  .activity-metric-chart[data-theme="archive"] {
    border-width: 3px;
    border-radius: 0;
  }

  .activity-metric-chart[data-theme="grapher"] {
    border-radius: 2px;
    border-bottom-width: 3px;
  }

  .chart-header,
  .sport-chart > header {
    display: flex;
    align-items: start;
    justify-content: space-between;
    gap: 1rem;
  }

  .chart-header h2,
  .sport-chart h3,
  .chart-header p,
  .sport-chart p {
    margin: 0;
  }

  .chart-header h2 {
    font-size: clamp(1.25rem, 2.5vw, 1.8rem);
  }

  .sport-chart h3 {
    font-size: 1rem;
  }

  .eyebrow {
    margin: 0 0 0.35rem;
    color: var(--accent);
    font-size: 0.66rem;
    font-weight: 750;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  .chart-description,
  .coverage,
  .chart-status {
    color: var(--text-muted);
    font-size: 0.78rem;
    line-height: 1.5;
  }

  .coverage {
    white-space: nowrap;
  }

  .chart-grid {
    display: grid;
    grid-template-columns: minmax(0, 1.6fr) minmax(16rem, 0.9fr);
    gap: 1rem;
  }

  .trend-chart,
  .trend-charts,
  .sport-chart {
    min-width: 0;
  }

  .trend-charts {
    display: grid;
    gap: 0.75rem;
  }

  .trend-charts .eyebrow {
    margin-bottom: -0.25rem;
  }

  .sport-chart {
    display: grid;
    align-content: start;
    gap: 0.35rem;
  }

  .series-table-wrap {
    max-height: 15rem;
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
    text-align: left;
  }

  th {
    color: var(--text-muted);
    font-size: 0.64rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  @media (max-width: 768px) {
    .chart-grid {
      grid-template-columns: 1fr;
    }

    .chart-header {
      display: grid;
    }
  }
</style>
