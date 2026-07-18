<script lang="ts">
  import type { Summary, SummaryBucket } from "$lib/api";
  import { formatDistance, formatDuration, formatSport } from "$lib/format";

  export type ShareVariant =
    "atlas" | "field-journal" | "phenology" | "sound-map" | "archive";
  type Slot = { label: string; key: string; bucket?: SummaryBucket };
  let {
    variant,
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
    variant: ShareVariant;
    summary: Summary;
    fullSummary: Summary | null;
    selectedYear: string | null;
    selectedYearTotals: SummaryBucket | null;
    years: string[];
    monthSlots: Slot[];
    monthMetric: "distance_m" | "activity_count";
    monthMax: number;
    sportFilter: string | null;
  } = $props();
  function value(slot: Slot): number {
    return slot.bucket?.[monthMetric] ?? 0;
  }
</script>

<section
  class={`theme-share theme-share-${variant}`}
  aria-labelledby="theme-share-title"
>
  <header class="share-head">
    <div>
      <p class="theme-kicker">
        Public projection / {selectedYear ?? "all years"}
      </p>
      <h1 id="theme-share-title">A record made readable.</h1>
      <p>
        Selected activity data, presented as a clear public window. Private
        daily, sleep, and media details remain outside this view.
      </p>
    </div>
    <span class="share-mark"
      >{variant === "atlas"
        ? "N"
        : variant === "phenology"
          ? "◒"
          : variant === "sound-map"
            ? "≈"
            : variant === "archive"
              ? "№"
              : "✽"}</span
    >
  </header>
  <div class="share-provenance">
    <span>Projection</span><strong>public/v1</strong><span>Selection</span
    ><strong>{sportFilter ? formatSport(sportFilter) : "all sports"}</strong
    ><span>Source</span><strong>sanitized records</strong>
  </div>
  <section class="share-totals">
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
      <span>Total time</span><strong
        >{formatDuration(selectedYearTotals?.duration_s ?? 0)}</strong
      >
    </div>
  </section>
  <nav class="year-index" aria-label="Available years">
    {#each years as year}<span class:active={year === selectedYear}>{year}</span
      >{/each}
  </nav>
  <section class="share-chart">
    <header>
      <div>
        <p class="theme-kicker">Monthly series</p>
        <h2>
          {monthMetric === "distance_m" ? "Distance" : "Activities"} by month
        </h2>
      </div>
      <span>{selectedYear ?? "all years"}</span>
    </header>
    <div class="month-bars" role="img" aria-label="Monthly activity comparison">
      {#each monthSlots as slot}<div>
          <i style={`height: ${Math.max(3, (value(slot) / monthMax) * 100)}%`}
          ></i><strong>{slot.label}</strong><small
            >{monthMetric === "distance_m"
              ? formatDistance(slot.bucket?.distance_m)
              : `${value(slot)} activities`}</small
          >
        </div>{/each}
    </div>
  </section>
  <footer class="share-source">
    <strong>About this view</strong><span
      >{fullSummary && selectedYear
        ? "Comparison includes surrounding years for context."
        : "Public projection of sanitized activity records."}</span
    >
  </footer>
</section>

<style>
  .theme-share {
    display: grid;
    gap: 1.25rem;
  }
  .theme-kicker {
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-size: 0.66rem;
    letter-spacing: 0.14em;
    text-transform: uppercase;
  }
  h1,
  h2 {
    margin: 0;
    font-family: Georgia, serif;
    font-weight: 400;
    letter-spacing: -0.05em;
  }
  h1 {
    max-width: 12ch;
    font-size: clamp(2.8rem, 7vw, 6rem);
    line-height: 0.9;
  }
  h2 {
    font-size: 1.6rem;
  }
  .share-head {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    border-bottom: 1px solid var(--border);
    padding-bottom: 2rem;
  }
  .share-head p:last-child {
    max-width: 42rem;
    color: var(--text-muted);
    line-height: 1.65;
  }
  .share-mark {
    display: grid;
    width: 5rem;
    height: 5rem;
    place-items: center;
    border: 1px solid var(--accent);
    color: var(--accent);
    font-family: Georgia, serif;
    font-size: 2rem;
  }
  .share-provenance {
    display: grid;
    grid-template-columns: auto 1fr auto 1fr auto 2fr;
    gap: 0.5rem 1rem;
    align-items: baseline;
    border-bottom: 1px solid var(--border);
    padding: 0.75rem 0;
    font-size: 0.72rem;
  }
  .share-provenance span {
    color: var(--text-muted);
  }
  .share-provenance strong {
    font-weight: 400;
  }
  .share-totals {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    border: 1px solid var(--border);
    background: color-mix(in srgb, var(--surface-1) 88%, transparent);
  }
  .share-totals div {
    display: grid;
    gap: 0.45rem;
    border-right: 1px solid var(--border);
    padding: 1.4rem;
  }
  .share-totals div:last-child {
    border-right: 0;
  }
  .share-totals span {
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  .share-totals strong {
    font-family: Georgia, serif;
    font-size: 1.6rem;
    font-weight: 400;
  }
  .year-index {
    display: flex;
    flex-wrap: wrap;
    gap: 0.45rem;
  }
  .year-index span {
    border: 1px solid var(--border);
    border-radius: 999px;
    padding: 0.4rem 0.7rem;
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  .year-index span.active {
    border-color: var(--accent);
    color: var(--accent);
  }
  .share-chart {
    border: 1px solid var(--border);
    padding: 1.5rem;
    background: color-mix(in srgb, var(--surface-1) 88%, transparent);
  }
  .share-chart header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .share-chart header > span {
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .month-bars {
    display: grid;
    grid-template-columns: repeat(12, 1fr);
    align-items: end;
    gap: 0.45rem;
    height: 16rem;
    margin-top: 1.5rem;
    border-bottom: 1px solid var(--border);
  }
  .month-bars > div {
    display: grid;
    grid-template-rows: 1fr auto auto;
    align-items: end;
    height: 100%;
    min-width: 0;
  }
  .month-bars i {
    display: block;
    width: 65%;
    min-height: 0.2rem;
    margin: 0 auto;
    background: var(--accent);
  }
  .month-bars strong,
  .month-bars small {
    overflow: hidden;
    color: var(--text-muted);
    font-size: 0.58rem;
    text-align: center;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .month-bars small {
    margin-top: 0.35rem;
    font-size: 0.5rem;
  }
  .share-source {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    border-top: 1px solid var(--border);
    padding-top: 0.9rem;
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  .share-source strong {
    color: var(--text);
    font-weight: 400;
  }
  .theme-share-atlas {
    font-family: "Avenir Next", sans-serif;
  }
  .theme-share-atlas .share-chart {
    border-top: 0.35rem solid var(--accent);
  }
  .theme-share-phenology h1,
  .theme-share-phenology h2 {
    font-style: italic;
  }
  .theme-share-phenology .share-totals {
    border-radius: 1rem 0.2rem;
  }
  .theme-share-sound-map {
    font-family: "IBM Plex Mono", monospace;
  }
  .theme-share-sound-map h1,
  .theme-share-sound-map h2 {
    font-family: inherit;
  }
  .theme-share-archive .share-chart {
    box-shadow: 0.45rem 0.45rem 0
      color-mix(in srgb, var(--accent) 18%, transparent);
  }
  @media (max-width: 680px) {
    .share-head,
    .share-chart header,
    .share-source {
      display: block;
    }
    .share-mark {
      margin-top: 1.5rem;
    }
    .share-head p:last-child {
      margin-bottom: 0;
    }
    .share-provenance {
      grid-template-columns: auto 1fr;
    }
    .share-totals {
      grid-template-columns: 1fr;
    }
    .share-totals div {
      border-right: 0;
      border-bottom: 1px solid var(--border);
    }
    .share-totals div:last-child {
      border-bottom: 0;
    }
    .share-chart header > span {
      display: block;
      margin-top: 0.7rem;
    }
    .share-source span {
      display: block;
      margin-top: 0.7rem;
    }
  }
</style>
