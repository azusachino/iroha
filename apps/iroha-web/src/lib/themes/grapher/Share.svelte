<script lang="ts">
  import type { Summary, SummaryBucket } from "$lib/api";
  import { formatDistance, formatDuration, formatSport } from "$lib/format";

  type MonthSlot = { label: string; key: string; bucket?: SummaryBucket };

  let {
    summary,
    fullSummary,
    selectedYear,
    selectedYearTotals,
    years,
    monthSlots,
    monthMetric,
    monthMax,
    sportFilter,
  }: {
    summary: Summary;
    fullSummary: Summary | null;
    selectedYear: string | null;
    selectedYearTotals: SummaryBucket | null;
    years: string[];
    monthSlots: MonthSlot[];
    monthMetric: "distance_m" | "activity_count";
    monthMax: number;
    sportFilter: string | null;
  } = $props();

  function metricValue(slot: MonthSlot): number {
    return slot.bucket?.[monthMetric] ?? 0;
  }
</script>

<section class="grapher-share" aria-labelledby="grapher-share-title">
  <header class="share-intro">
    <p class="grapher-kicker">
      Public data explorer / {selectedYear ?? "all years"}
    </p>
    <h1 id="grapher-share-title">A readable record of movement.</h1>
    <p>
      This is a public projection of selected activity data. Private daily,
      sleep, and media details remain outside this view.
    </p>
  </header>

  <div class="share-provenance">
    <span>Projection</span><strong>public/v1</strong>
    <span>Selection</span><strong
      >{sportFilter ? formatSport(sportFilter) : "all sports"}</strong
    >
    <span>Source</span><strong>sanitized activity records</strong>
  </div>

  <section class="share-totals" aria-label="Selected year totals">
    <div>
      <span>Distance</span><strong
        >{formatDistance(selectedYearTotals?.distance_m ?? 0)}</strong
      >
    </div>
    <div>
      <span>Activities</span><strong
        >{(selectedYearTotals?.activity_count ?? 0).toLocaleString()}</strong
      >
    </div>
    <div>
      <span>Time</span><strong
        >{formatDuration(selectedYearTotals?.duration_s ?? 0)}</strong
      >
    </div>
  </section>

  <section class="year-index" aria-label="Available years">
    <span class="grapher-kicker">Compare year</span>
    {#each years as year}
      <span class:active={year === selectedYear}>{year}</span>
    {/each}
  </section>

  <section class="share-chart" aria-labelledby="share-chart-title">
    <div class="panel-heading">
      <div>
        <p class="grapher-kicker">Monthly series</p>
        <h2 id="share-chart-title">
          {monthMetric === "distance_m" ? "Distance" : "Activities"} by month
        </h2>
      </div>
      <span>{selectedYear ?? "all years"}</span>
    </div>
    <div
      class="month-chart"
      role="img"
      aria-label="Monthly activity comparison"
    >
      {#each monthSlots as slot}
        <div class="month-column">
          <div class="month-bar-wrap">
            <i
              style={`height: ${Math.max(3, (metricValue(slot) / monthMax) * 100)}%`}
            ></i>
          </div>
          <strong>{slot.label}</strong>
          <small
            >{monthMetric === "distance_m"
              ? formatDistance(slot.bucket?.distance_m)
              : `${metricValue(slot)} activities`}</small
          >
        </div>
      {/each}
    </div>
  </section>

  <section class="source-panel" aria-label="Data source note">
    <strong>About this view</strong>
    <p>
      The public projection contains activity summaries and sanitized routes.
      {#if fullSummary && selectedYear}
        The comparison baseline includes the surrounding years for context.
      {/if}
    </p>
  </section>
</section>

<style>
  .grapher-share {
    display: grid;
    gap: 1rem;
  }
  .share-intro {
    max-width: 52rem;
    padding-bottom: 2rem;
    border-bottom: 3px solid var(--text);
  }
  .grapher-kicker {
    margin: 0 0 0.45rem;
    color: var(--accent);
    font-size: 0.68rem;
    font-weight: 750;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  h1,
  h2 {
    margin: 0;
    letter-spacing: -0.06em;
  }
  h1 {
    font-size: clamp(2.8rem, 7vw, 6.5rem);
    line-height: 0.9;
  }
  h2 {
    font-size: 1.25rem;
  }
  .share-intro > p:not(.grapher-kicker) {
    max-width: 38rem;
    margin: 1rem 0 0;
    color: var(--text-muted);
    line-height: 1.55;
  }
  .share-provenance {
    display: flex;
    flex-wrap: wrap;
    gap: 0.3rem 0.6rem;
    padding: 0.8rem 0;
    border-bottom: 1px solid var(--border);
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .share-provenance strong {
    margin-right: 1rem;
    color: var(--text);
  }
  .share-totals {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    border-block: 1px solid var(--border);
  }
  .share-totals div {
    display: grid;
    gap: 0.45rem;
    padding: 1.2rem;
    border-right: 1px solid var(--border);
  }
  .share-totals div:last-child {
    border: 0;
  }
  .share-totals span {
    color: var(--text-muted);
    font-size: 0.68rem;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }
  .share-totals strong {
    font-size: clamp(1.5rem, 4vw, 3rem);
    letter-spacing: -0.08em;
  }
  .year-index {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 0.6rem;
    padding: 0.5rem 0;
  }
  .year-index .grapher-kicker {
    margin: 0 0.8rem 0 0;
  }
  .year-index > span:not(.grapher-kicker) {
    padding: 0.35rem 0.5rem;
    border-bottom: 2px solid transparent;
    color: var(--text-muted);
    font-size: 0.78rem;
  }
  .year-index > span.active {
    border-color: var(--accent);
    color: var(--text);
  }
  .share-chart {
    padding: 1.25rem;
    border: 1px solid var(--border);
    background: var(--surface);
  }
  .panel-heading {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .panel-heading > span {
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .month-chart {
    display: grid;
    grid-template-columns: repeat(12, minmax(0, 1fr));
    gap: 0.55rem;
    height: 18rem;
    margin-top: 2rem;
    border-bottom: 1px solid var(--border);
  }
  .month-column {
    display: grid;
    grid-template-rows: 1fr auto auto;
    gap: 0.45rem;
    min-width: 0;
    text-align: center;
  }
  .month-bar-wrap {
    display: flex;
    align-items: end;
    justify-content: center;
    min-height: 0;
    background: repeating-linear-gradient(
      to top,
      transparent 0 3rem,
      color-mix(in srgb, var(--border) 65%, transparent) 3rem 3.05rem
    );
  }
  .month-bar-wrap i {
    display: block;
    width: 70%;
    min-height: 0.2rem;
    background: var(--accent);
  }
  .month-column strong {
    font-size: 0.68rem;
  }
  .month-column small {
    overflow: hidden;
    color: var(--text-muted);
    font-size: 0.58rem;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .source-panel {
    padding: 1rem 0;
    border-top: 1px solid var(--border);
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  .source-panel p {
    max-width: 40rem;
    margin: 0.4rem 0 0;
    line-height: 1.5;
  }
  @media (max-width: 680px) {
    .share-totals div {
      padding: 0.85rem 0.5rem;
    }
    .share-totals strong {
      font-size: 1.35rem;
    }
    .month-chart {
      gap: 0.2rem;
      overflow-x: auto;
    }
    .month-column {
      min-width: 2.8rem;
    }
  }
</style>
