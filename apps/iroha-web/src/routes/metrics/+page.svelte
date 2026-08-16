<script lang="ts">
  import { onMount } from "svelte";
  import { replaceState } from "$app/navigation";
  import { page } from "$app/state";
  import { Activity } from "@lucide/svelte";
  import {
    getDailyBounds,
    getMetricCatalog,
    getMetricSeries,
    type MetricDefinition,
    type MetricSeriesResponse,
  } from "$lib/api";
  import DailySmallMultiples, {
    type SmallMultiple,
  } from "@iroha/shared/theme-ui/components/DailySmallMultiples.svelte";
  import { formatMetricValue } from "$lib/format";
  import PeriodSelector from "$lib/components/PeriodSelector.svelte";
  import PeriodToolbar from "$lib/components/PeriodToolbar.svelte";
  import ThemeRouteRenderer from "@iroha/shared/theme-ui/ThemeRouteRenderer.svelte";
  import MetricPanel from "@iroha/shared/components/MetricPanel.svelte";
  import { seriesPanelRows } from "@iroha/shared/components/metric-panel";
  import { pointValue } from "@iroha/shared/components/metric-series";
  import {
    currentMonth,
    monthBounds,
    monthOptionsInRange,
    shiftMonth,
    yearOptionsInRange,
  } from "@iroha/shared/format/month";
  import {
    currentCalendarScope,
    readCalendarScope,
    serializeCalendarScope,
    writeCalendarScope,
    type DateBounds,
  } from "@iroha/shared/format/scope";
  import { IROHA_TIMEZONE } from "$lib/config";

  let catalog = $state<MetricDefinition[]>([]);
  let metricId = $state(page.url.searchParams.get("metric") ?? "");
  const defaultMonthScope = currentCalendarScope(
    "month",
    new Date(),
    IROHA_TIMEZONE,
  );
  const requestedMonthScope = readCalendarScope(page.url.searchParams, {
    fallback: defaultMonthScope,
    allowDay: false,
  });
  let month = $state(
    requestedMonthScope.kind === "month"
      ? (serializeCalendarScope(requestedMonthScope) as string)
      : currentMonth(new Date(), IROHA_TIMEZONE),
  );
  let dimensions = $state<Record<string, string>>({});
  let series = $state<MetricSeriesResponse | null>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);
  let requestVersion = 0;
  // The real cross-domain data range (fetched once, independent of the
  // current selection) -- not a hardcoded 2015 guess, and not every month.
  let bounds = $state<DateBounds>({});
  const periodYears = $derived(yearOptionsInRange(bounds));
  const periodYear = $derived(month.slice(0, 4));
  const periodMonth = $derived(String(Number(month.slice(5, 7))));
  const periodMonths = $derived(monthOptionsInRange(periodYear, bounds));

  async function loadBounds() {
    try {
      bounds = await getDailyBounds();
    } catch {
      bounds = {};
    }
    if (!bounds.min || !bounds.max) return;
    if (month < bounds.min.slice(0, 7)) month = bounds.min.slice(0, 7);
    else if (month > bounds.max.slice(0, 7)) month = bounds.max.slice(0, 7);
    else return;
    syncUrl();
    void loadSeries();
  }

  const definition = $derived(
    catalog.find((metric) => metric.id === metricId) ?? null,
  );
  const panelRows = $derived(
    seriesPanelRows(series?.series ?? [], (value) =>
      formatMetricValue(value, series?.unit),
    ),
  );
  const labels = $derived(series?.series[0]?.points.map((p) => p.period) ?? []);
  const charts = $derived<SmallMultiple[]>(
    series?.series.map((item) => ({
      label:
        Object.values(item.dimensions).join(" · ") || series?.label || metricId,
      values: item.points.map((point) => pointValue(point)),
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
      void loadBounds();
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause);
      loading = false;
    }
  });

  function resetDimensions() {
    const fromUrl = new Map(
      page.url.searchParams.getAll("dimension").map((value) => {
        const separator = value.indexOf(":");
        return separator > 0
          ? [value.slice(0, separator), value.slice(separator + 1)]
          : ["", ""];
      }),
    );
    dimensions = Object.fromEntries(
      (definition?.dimensions ?? [])
        .map((dimension) => [
          dimension.id,
          dimension.values.includes(fromUrl.get(dimension.id) ?? "")
            ? fromUrl.get(dimension.id)
            : dimension.required
              ? (dimension.values[0] ?? "")
              : "",
        ])
        .filter(([, value]) => value),
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

  function selectPeriodYear(value: string) {
    if (!/^\d{4}$/.test(value)) return;
    selectMonth(`${value}-${month.slice(5, 7)}`);
  }

  function selectPeriodMonth(value: string) {
    if (!/^(?:[1-9]|1[0-2])$/.test(value)) return;
    selectMonth(`${periodYear}-${value.padStart(2, "0")}`);
  }

  function selectDimension(id: string, value: string) {
    dimensions = { ...dimensions, [id]: value };
    syncUrl();
    void loadSeries();
  }

  function syncUrl() {
    const url = new URL(window.location.href);
    url.searchParams.set("metric", metricId);
    writeCalendarScope(url.searchParams, {
      kind: "month",
      year: Number(month.slice(0, 4)),
      month: Number(month.slice(5, 7)),
    });
    url.searchParams.delete("dimension");
    for (const [id, value] of Object.entries(dimensions)) {
      if (value) url.searchParams.append("dimension", `${id}:${value}`);
    }
    if (url.search !== window.location.search) replaceState(url, page.state);
  }
</script>

<svelte:head><title>Metrics · iroha</title></svelte:head>

<section class="metrics-shell">
  <header>
    <div>
      <p class="eyebrow"><Activity size={14} /> Metric explorer</p>
      <h1>Metrics</h1>
    </div>
  </header>

  <PeriodToolbar title="Monthly metric window" ariaLabel="Metric period">
    <PeriodSelector
      year={periodYear}
      month={periodMonth}
      years={periodYears}
      months={periodMonths}
      {bounds}
      showAllYears={false}
      showAllMonths={false}
      surface="inline"
      onYear={selectPeriodYear}
      onMonth={selectPeriodMonth}
    />
  </PeriodToolbar>

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
        </div>
        <MetricPanel
          metricId={series.metric_id}
          label={definition.label}
          unit={series.unit}
          method={series.series[0]?.source.method ??
            definition.aggregation_version}
          coverage={series.series[0]?.coverage}
          sourceKinds={series.series[0]?.source.source_kinds ?? []}
          rows={panelRows}
          period={month}
        >
          <DailySmallMultiples {labels} {charts} />
        </MetricPanel>
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
  select {
    min-height: 2.4rem;
    padding: 0.45rem 0.7rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 4px);
    background: var(--surface-2);
    color: var(--text);
    font: inherit;
  }
  .chart {
    display: grid;
    gap: 1rem;
  }
  .section-head p:last-child,
  .status {
    color: var(--text-muted);
  }
  .error {
    color: var(--danger);
  }
</style>
