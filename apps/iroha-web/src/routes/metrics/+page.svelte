<script lang="ts">
  import { onMount } from "svelte";
  import { replaceState } from "$app/navigation";
  import { page } from "$app/state";
  import { Activity, Download } from "@lucide/svelte";
  import {
    getMetricCatalog,
    getMetricSeries,
    type MetricDefinition,
    type MetricSeriesResponse,
  } from "$lib/api";
  import DailySmallMultiples, {
    type SmallMultiple,
  } from "$lib/components/DailySmallMultiples.svelte";
  import ThemeMonthNavigator from "$lib/themes/ThemeMonthNavigator.svelte";
  import ThemeRouteRenderer from "$lib/themes/ThemeRouteRenderer.svelte";
  import MetricMetadata from "@iroha/shared/MetricMetadata.svelte";
  import MetricTable from "@iroha/shared/MetricTable.svelte";
  import { metricSeriesCsv, pointValue } from "@iroha/shared/metric-series";
  import {
    canonicalMonth,
    currentMonth,
    monthBounds,
    shiftMonth,
  } from "@iroha/shared/month";

  let catalog = $state<MetricDefinition[]>([]);
  let metricId = $state(page.url.searchParams.get("metric") ?? "");
  let month = $state(
    canonicalMonth(page.url.searchParams.get("month"), currentMonth()),
  );
  let dimensions = $state<Record<string, string>>({});
  let series = $state<MetricSeriesResponse | null>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);
  let requestVersion = 0;

  const definition = $derived(
    catalog.find((metric) => metric.id === metricId) ?? null,
  );
  const labels = $derived(series?.series[0]?.points.map((p) => p.period) ?? []);
  const charts = $derived<SmallMultiple[]>(
    series?.series.map((item) => ({
      label:
        Object.values(item.dimensions).join(" · ") || series?.label || metricId,
      values: item.points.map((point) => pointValue(point) ?? Number.NaN),
      color: "var(--accent)",
      unit: series?.unit,
    })) ?? [],
  );

  onMount(async () => {
    try {
      const response = await getMetricCatalog();
      catalog = response.metrics;
      if (!catalog.some((metric) => metric.id === metricId)) {
        metricId = catalog[0]?.id ?? "";
      }
      resetDimensions();
      await loadSeries();
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause);
      loading = false;
    }
  });

  function resetDimensions() {
    dimensions = Object.fromEntries(
      (definition?.dimensions ?? [])
        .filter((dimension) => dimension.required)
        .map((dimension) => [dimension.id, dimension.values[0] ?? ""]),
    );
  }

  async function loadSeries() {
    if (!definition) return;
    const version = ++requestVersion;
    loading = true;
    error = null;
    const from = monthBounds(shiftMonth(month, -11)).from;
    const to = monthBounds(month).to;
    try {
      const next = await getMetricSeries(metricId, {
        from,
        to,
        grain: "month",
        dimensions: Object.entries(dimensions)
          .filter(([, value]) => value)
          .map(([key, value]) => `${key}:${value}`),
      });
      if (version === requestVersion) series = next;
    } catch (cause) {
      if (version !== requestVersion) return;
      series = null;
      error = cause instanceof Error ? cause.message : String(cause);
    } finally {
      if (version === requestVersion) loading = false;
    }
  }

  function selectMetric(value: string) {
    metricId = value;
    resetDimensions();
    syncUrl();
    void loadSeries();
  }

  function selectMonth(value: string) {
    month = value;
    syncUrl();
    void loadSeries();
  }

  function selectDimension(id: string, value: string) {
    dimensions = { ...dimensions, [id]: value };
    void loadSeries();
  }

  function syncUrl() {
    const url = new URL(page.url);
    url.searchParams.set("metric", metricId);
    url.searchParams.set("month", month);
    if (url.search !== page.url.search) replaceState(url, page.state);
  }

  function downloadCsv() {
    if (!series) return;
    const blob = new Blob(
      [metricSeriesCsv(series.metric_id, series.unit, series.series)],
      { type: "text/csv;charset=utf-8" },
    );
    const link = document.createElement("a");
    link.href = URL.createObjectURL(blob);
    link.download = `${series.metric_id}-${month}.csv`;
    link.click();
    URL.revokeObjectURL(link.href);
  }
</script>

<svelte:head><title>Metrics · iroha</title></svelte:head>

<section class="metrics-shell">
  <header>
    <div>
      <p class="eyebrow"><Activity size={14} /> Metric explorer</p>
      <h1>Metrics</h1>
    </div>
    <ThemeMonthNavigator {month} onMonth={selectMonth} disabled={loading} />
  </header>

  <ThemeRouteRenderer route="metrics" props={{}}>
    <div class="controls panel">
      <label
        >Metric<select
          value={metricId}
          onchange={(event) => selectMetric(event.currentTarget.value)}
        >
          {#each catalog as metric}<option value={metric.id}
              >{metric.domain} · {metric.label}</option
            >{/each}
        </select></label
      >
      {#each definition?.dimensions ?? [] as dimension}
        <label
          >{dimension.label}<select
            value={dimensions[dimension.id] ?? ""}
            onchange={(event) =>
              selectDimension(dimension.id, event.currentTarget.value)}
          >
            {#if !dimension.required}<option value="">All</option>{/if}
            {#each dimension.values as value}<option {value}>{value}</option
              >{/each}
          </select></label
        >
      {/each}
    </div>

    {#if loading}<p class="status">Loading metric series…</p>
    {:else if error}<p class="error" role="alert">{error}</p>
    {:else if series && definition}
      <section class="chart panel">
        <div class="section-head">
          <div>
            <p class="eyebrow">{definition.domain}</p>
            <h2>{definition.label}</h2>
            <p>{definition.description}</p>
          </div>
          <button type="button" onclick={downloadCsv}
            ><Download size={14} /> CSV</button
          >
        </div>
        <DailySmallMultiples {labels} {charts} />
      </section>
      <section class="details panel">
        <MetricMetadata
          unit={series.unit}
          method={series.series[0]?.source.method ??
            definition.aggregation_version}
          coverage={series.series[0]?.coverage}
          sourceKinds={series.series[0]?.source.source_kinds ?? []}
        />
        <MetricTable series={series.series} unit={series.unit} />
      </section>
    {/if}
  </ThemeRouteRenderer>
</section>

<style>
  .metrics-shell {
    display: grid;
    gap: 1.25rem;
  }
  header,
  .section-head,
  .controls {
    display: flex;
    align-items: end;
    justify-content: space-between;
    gap: 1rem;
    flex-wrap: wrap;
  }
  h1,
  h2,
  p {
    margin: 0;
  }
  h1 {
    font-size: clamp(2.7rem, 7vw, 5.8rem);
    letter-spacing: -0.09em;
    line-height: 0.9;
  }
  h2 {
    font-size: 1.5rem;
  }
  .eyebrow {
    display: flex;
    align-items: center;
    gap: 0.35rem;
    color: var(--accent);
    font-size: 0.68rem;
    font-weight: 750;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .panel {
    padding: 1rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
  }
  .controls {
    justify-content: start;
    align-items: center;
  }
  label {
    display: grid;
    gap: 0.3rem;
    color: var(--text-muted);
    font-size: 0.7rem;
    font-weight: 700;
  }
  select,
  button {
    min-height: 2.4rem;
    padding: 0.45rem 0.7rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 4px);
    background: var(--surface-2);
    color: var(--text);
    font: inherit;
  }
  button {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    cursor: pointer;
  }
  .chart {
    display: grid;
    gap: 1rem;
  }
  .section-head p:last-child,
  .status {
    color: var(--text-muted);
  }
  .details {
    display: grid;
    gap: 1rem;
  }
  .error {
    color: var(--danger);
  }
</style>
