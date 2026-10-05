<script lang="ts">
  import RingGauge from "@iroha/shared/theme-ui/components/RingGauge.svelte";
  import DailySmallMultiples from "@iroha/shared/theme-ui/components/DailySmallMultiples.svelte";
  import PeriodSelector from "$lib/components/PeriodSelector.svelte";
  import PeriodToolbar from "$lib/components/PeriodToolbar.svelte";
  import {
    formatDateOnly,
    formatMonth as formatCanonicalMonth,
  } from "$lib/format";
  import LoadingBoundary from "$lib/components/LoadingBoundary.svelte";
  import RouteIntro from "$lib/components/RouteIntro.svelte";
  import { useTheme } from "$lib/themes/context.svelte";
  import ThemeRouteRenderer from "@iroha/shared/theme-ui/ThemeRouteRenderer.svelte";
  import { hasThemeRoute } from "$lib/themes/registry";
  import { createPatternsState } from "./patterns-state.svelte";
  import FallbackPatternsTable from "./FallbackPatternsTable.svelte";
  import RetryNotice from "@iroha/shared/theme-ui/components/RetryNotice.svelte";

  type Gran = "day" | "month" | "year";
  const theme = useTheme();
  const t = createPatternsState();
  const themeProps = $derived({
    seriesAvailable: t.seriesAvailable,
    chrono: t.chrono,
    gran: t.gran,
    onGran: (value: Gran) => void t.changeGranularity(value),
    onDrillIndex: t.drillIntoIndex,
    onDrillPeriod: t.drillIntoPeriod,
    ringData: t.ringData,
    latestRingDay: t.latestRingDay,
  });
  let seriesTarget = $state<HTMLElement>();
  let latestTarget = $state<HTMLElement>();
  let boundsTarget = $state<HTMLElement>();
</script>

<svelte:head>
  <title>Patterns · iroha</title>
</svelte:head>

<section class="daily">
  {#if hasThemeRoute(theme.definition(), "daily")}
    <ThemeRouteRenderer
      route="daily"
      props={{ ...themeProps, section: "controls" }}
    >
      {#snippet children()}
        <section
          aria-label="Pattern scope selection"
          tabindex="-1"
          bind:this={boundsTarget}
        >
          <PeriodToolbar title="Daily pattern scope" ariaLabel="Daily period">
            <PeriodSelector
              years={t.periodYears}
              months={t.periodMonths}
              year={t.gran === "year" ? t.selectedYear : t.activeYear}
              month={t.gran === "day" ? t.activeMonth : t.selectedMonth}
              bounds={t.dailyBounds}
              showAllYears={t.gran === "year"}
              surface="inline"
              onYear={t.selectYear}
              onMonth={t.selectMonth}
            />
          </PeriodToolbar>
          <p>Selected pattern scope: {t.periodLabel}</p>
          {#if t.boundsResource.error}
            <RetryNotice
              message={`Pattern range unavailable. ${t.boundsResource.error}`}
              retryLabel="Retry pattern range"
              onRetry={t.loadBounds}
              focusTarget={boundsTarget}
            />
            <p>Calendar choices are not an observed data inventory.</p>
          {/if}
        </section>
      {/snippet}
    </ThemeRouteRenderer>
    <section
      aria-label="Latest pattern reading"
      tabindex="-1"
      bind:this={latestTarget}
    >
      {#if t.latestDayResource.error}
        <RetryNotice
          message={`Latest reading unavailable. ${t.latestDayResource.error}`}
          retryLabel="Retry latest reading"
          onRetry={t.loadLatestDay}
          focusTarget={latestTarget}
        />
      {/if}
      <LoadingBoundary
        resource={t.latestDayResource}
        preserveLayout
        label="Loading latest reading…"
      >
        {#if t.latestDayResource.ready}
          <ThemeRouteRenderer
            route="daily"
            props={{ ...themeProps, section: "rings" }}
          />
        {/if}
      </LoadingBoundary>
    </section>
    <section
      aria-label="Pattern periods"
      tabindex="-1"
      bind:this={seriesTarget}
    >
      {#if t.seriesResource.error}
        <RetryNotice
          message={`Pattern periods unavailable. ${t.seriesResource.error}`}
          retryLabel={t.seriesRetryLabel}
          onRetry={t.retrySeries}
          focusTarget={seriesTarget}
        />
      {/if}
      <LoadingBoundary
        resource={t.seriesResource}
        preserveLayout
        label="Loading time-series data…"
      >
        {#if t.seriesAvailable}
          <p>Observed pattern scope: {t.observedScope}</p>
          <ThemeRouteRenderer
            route="daily"
            props={{ ...themeProps, section: "series" }}
          />
        {/if}
      </LoadingBoundary>
    </section>
  {:else}
    <RouteIntro
      eyebrow="Patterns / personal history"
      title="Patterns & Vitals"
      description="Rings, movement, and body signals across your history. Start with the latest day, then zoom out to see the pattern."
      actionHref="/"
      actionLabel="Today"
    />

    <PeriodToolbar title="Daily pattern scope" ariaLabel="Daily period">
      <PeriodSelector
        years={t.periodYears}
        months={t.periodMonths}
        year={t.gran === "year" ? t.selectedYear : t.activeYear}
        month={t.gran === "day" ? t.activeMonth : t.selectedMonth}
        bounds={t.dailyBounds}
        showAllYears={t.gran === "year"}
        surface="inline"
        onYear={t.selectYear}
        onMonth={t.selectMonth}
      />
    </PeriodToolbar>

    {#if t.activeResources.some((r) => r.loading) && t.dayRows.length === 0}
      <p class="muted status">Loading daily history…</p>
    {:else if t.error}
      <p class="error status">Could not load daily data: {t.error}</p>
    {:else if t.dayRows.length === 0}
      <p class="muted status">No daily data imported yet.</p>
    {:else}
      <div class="hero tile glow">
        <div class="hero-head">
          <span class="hero-kicker">Latest rings</span>
          {#if t.latestRingDay}<span class="hero-date"
              >{formatDateOnly(t.latestRingDay.day)}</span
            >{/if}
        </div>
        <RingGauge rings={t.ringData} />
      </div>

      <div class="controls">
        <div class="seg" role="tablist" aria-label="Aggregation granularity">
          {#each ["day", "month", "year"] as const as g}
            <button
              role="tab"
              aria-selected={t.gran === g}
              class:active={t.gran === g}
              onclick={() => void t.changeGranularity(g)}
            >
              {g[0].toUpperCase() + g.slice(1)}
            </button>
          {/each}
        </div>
        <span class="muted small">
          {#if t.aggregated}
            {t.chrono.length}
            {t.gran}s in {t.gran === "year" ? "view" : t.activeYear} · per-day averages
          {:else}
            {t.chrono.length} days in {formatCanonicalMonth(t.activeMonth)}
          {/if}
        </span>
        {#if t.rangeFrom && t.rangeTo}<span class="muted small"
            >Range: {t.rangeFrom} → before {t.rangeTo}</span
          >{/if}
      </div>
      <div class="trend-panel tile">
        <div class="trend-heading">
          <div>
            <span class="t-label">Daily signals</span>
            <h2>Small multiples</h2>
          </div>
          <span class="muted small"
            >Move the crosshair to compare one period</span
          >
        </div>
        <DailySmallMultiples
          labels={t.chrono.map((d) => d.label)}
          charts={t.trendCharts}
        />
      </div>

      <section class="atlas-note tile" aria-label="Pattern reading guide">
        <div>
          <span class="t-label">Reading the atlas</span>
          <h2>{t.gran[0].toUpperCase() + t.gran.slice(1)} view</h2>
        </div>
        <p>
          Compare movement and body signals at this scale, then use the table as
          the precise record. Missing values remain unfilled rather than being
          treated as zero.
        </p>
        <span class="muted small">{t.chrono.length} periods in view</span>
      </section>

      <div class="table-wrap tile">
        <FallbackPatternsTable
          gran={t.gran}
          aggregated={t.aggregated}
          rows={t.table}
          format={t.fmt}
          onDrill={t.drillIntoPeriod}
        />
      </div>
    {/if}
  {/if}
</section>

<style>
  .daily {
    display: grid;
    gap: 1.25rem;
  }
  .status {
    padding: 2rem 0;
  }
  .error {
    color: var(--danger);
  }
  .small {
    font-size: 0.78rem;
  }

  .hero {
    padding: 1.25rem;
    width: fit-content;
  }
  .glow {
    border: 1px solid color-mix(in srgb, var(--accent) 32%, var(--border));
    box-shadow: var(--accent-glow), var(--tile-shadow);
  }
  .hero-head {
    display: flex;
    align-items: baseline;
    gap: 1rem;
    justify-content: space-between;
    margin-bottom: 0.9rem;
  }
  .hero-kicker {
    color: var(--text-muted);
    font-size: 0.72rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .hero-date {
    color: var(--text);
    font-weight: 650;
  }

  .controls {
    display: flex;
    align-items: center;
    gap: 1rem;
    flex-wrap: wrap;
  }
  .seg {
    display: inline-flex;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    overflow: hidden;
  }
  .seg button {
    appearance: none;
    border: 0;
    background: var(--surface);
    color: var(--text-muted);
    padding: 0.45rem 0.9rem;
    font-size: 0.85rem;
    cursor: pointer;
  }
  .seg button + button {
    border-left: 1px solid var(--border);
  }
  .seg button.active {
    background: color-mix(in srgb, var(--accent) 18%, var(--surface));
    color: var(--accent);
    font-weight: 650;
  }

  .trend-panel {
    padding: 1rem;
  }

  .atlas-note {
    display: grid;
    grid-template-columns: minmax(12rem, 0.8fr) minmax(0, 1.6fr) auto;
    align-items: center;
    gap: 1rem;
    padding: 1rem 1.15rem;
    border-left: 3px solid var(--accent);
  }

  .atlas-note h2 {
    margin: 0.2rem 0 0;
    font-size: 1.05rem;
  }

  .atlas-note p {
    margin: 0;
    color: var(--text-muted);
    font-size: 0.86rem;
    line-height: 1.5;
  }
  .trend-heading {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    gap: 1rem;
  }
  .trend-heading h2 {
    margin: 0.2rem 0 0;
    font-size: 1rem;
  }
  .t-label {
    font-size: 0.78rem;
    color: var(--text-muted);
  }

  .table-wrap {
    padding: 0.4rem 0.4rem 0.6rem;
    overflow-x: auto;
  }
  @media (max-width: 768px) {
    .atlas-note {
      grid-template-columns: 1fr;
    }

    .trend-heading {
      align-items: flex-start;
      flex-direction: column;
    }
  }
</style>
