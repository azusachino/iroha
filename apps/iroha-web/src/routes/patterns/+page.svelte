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
  import PeriodDrill from "@iroha/shared/theme-ui/components/PeriodDrill.svelte";
  import { hasThemeRoute } from "$lib/themes/registry";
  import { createPatternsState } from "./patterns-state.svelte";

  type Gran = "day" | "month" | "year";
  const theme = useTheme();
  const t = createPatternsState();
</script>

<svelte:head>
  <title>Patterns · iroha</title>
</svelte:head>

<section class="daily">
  {#if hasThemeRoute(theme.definition(), "daily")}
    <LoadingBoundary
      resource={t.activeResources}
      preserveLayout
      label="Loading time-series data…"
    >
      {#if t.error}
        <p class="error status" role="alert">
          Could not load daily data: {t.error}
        </p>
      {/if}
      <ThemeRouteRenderer
        route="daily"
        props={{
          chrono: t.chrono,
          gran: t.gran,
          onGran: (value: Gran) => void t.changeGranularity(value),
          onDrillIndex: t.drillIntoIndex,
          onDrillPeriod: t.drillIntoPeriod,
          ringData: t.ringData,
          latestRingDay: t.latestRingDay,
        }}
      >
        {#snippet children()}
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
        {/snippet}
      </ThemeRouteRenderer>
    </LoadingBoundary>
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
        <table>
          <thead>
            <tr>
              <th class="l"
                >{t.gran === "day"
                  ? "Day"
                  : t.gran === "month"
                    ? "Month"
                    : "Year"}</th
              >
              {#if t.aggregated}<th>Days</th>{/if}
              <th>Move</th><th>Exer</th><th>Stand</th><th>Move ✓</th>
              <th>Steps{t.aggregated ? "/d" : ""}</th><th
                >Dist{t.aggregated ? "/d" : ""}</th
              >
              <th>rHR</th><th>HRV</th><th>SpO₂</th><th>Resp</th><th>VO₂</th><th
                >Mass</th
              >
            </tr>
          </thead>
          <tbody>
            {#each t.table as d}
              <tr>
                <td class="l">
                  {#if t.gran !== "day"}<PeriodDrill
                      label={d.label}
                      period={d.period}
                      onDrill={t.drillIntoPeriod}
                    />{:else}{d.label}{/if}
                </td>
                {#if t.aggregated}<td>{d.days}</td>{/if}
                <td>{t.fmt(d.move, 0)}</td>
                <td>{t.fmt(d.exercise, 0)}</td>
                <td>{t.fmt(d.stand, 0)}</td>
                <td
                  >{d.moveClosedPct == null
                    ? "—"
                    : `${Math.round(d.moveClosedPct)}%`}</td
                >
                <td>{t.fmt(d.steps, 0)}</td>
                <td>{t.fmt(d.distance, 1)}</td>
                <td>{t.fmt(d.resting_hr, 0)}</td>
                <td>{t.fmt(d.hrv_sdnn, 0)}</td>
                <td>{t.fmt(d.spo2_avg, 1)}</td>
                <td>{t.fmt(d.respiratory_rate, 1)}</td>
                <td>{t.fmt(d.vo2max, 1)}</td>
                <td>{t.fmt(d.body_mass_kg, 1)}</td>
              </tr>
            {/each}
          </tbody>
        </table>
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
  table {
    width: 100%;
    border-collapse: collapse;
    font-variant-numeric: tabular-nums;
    font-size: 0.84rem;
  }
  th,
  td {
    padding: 0.5rem 0.6rem;
    text-align: right;
    white-space: nowrap;
  }
  th {
    color: var(--text-muted);
    font-weight: 600;
    border-bottom: 1px solid var(--border);
    position: sticky;
    top: 0;
    background: var(--surface);
  }
  td {
    color: var(--text);
  }
  tbody tr + tr td {
    border-top: 1px solid color-mix(in srgb, var(--border) 55%, transparent);
  }
  .l {
    text-align: left;
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
