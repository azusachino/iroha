<script lang="ts">
  import type { ActivityThemeProps } from "../../activity-view";
  import ActivityMetricChart from "../components/ActivityMetricChart.svelte";
  import {
    formatDateOnly,
    formatDistance,
    formatDuration,
    formatPace,
  } from "../../format";
  import { sportLabel } from "../../sport";

  let {
    activities,
    displaySummary,
    sportType,
    sportOptions,
    loading,
    error,
    hasMore,
    loadingMore,
    onSportType,
    onLoadMore,
    onOpenDetail,
    activitySeries = null,
    activityDurationSeries = null,
    activitySeriesLoading = false,
    activitySeriesError = null,
    activitySeriesScope = "",
    children,
    theme,
  }: ActivityThemeProps = $props();

  function openActivity(event: MouseEvent, id: string): void {
    if ((event.target as HTMLElement).closest("a, button")) return;
    onOpenDetail(id);
  }

  function openActivityFromKeyboard(event: KeyboardEvent, id: string): void {
    if (event.key !== "Enter" && event.key !== " ") return;
    event.preventDefault();
    onOpenDetail(id);
  }
</script>

<section class="atlas-index" aria-labelledby="atlas-index-title">
  <header class="index-header">
    <div>
      <p class="atlas-kicker">Route index · gazetteer</p>
      <h1 id="atlas-index-title">Every route, logged.</h1>
      <p class="index-sub">
        A survey manifest of recorded ground: when it was covered, how far, and
        at what pace.
      </p>
    </div>
    <div class="grid-ref">
      <span>{displaySummary.activity_count}</span>
      <small>entries charted</small>
    </div>
  </header>

  {@render children?.()}

  <div class="index-filters" aria-label="Route filters">
    <label
      >Sport<select
        value={sportType}
        onchange={(event) =>
          onSportType((event.currentTarget as HTMLSelectElement).value)}
        ><option value="">All sports</option
        >{#each sportOptions as sport}<option value={sport}
            >{sportLabel(sport)}</option
          >{/each}</select
      ></label
    >
  </div>

  <ActivityMetricChart
    series={activitySeries}
    durationSeries={activityDurationSeries}
    loading={activitySeriesLoading}
    error={activitySeriesError}
    scope={activitySeriesScope}
    {theme}
  />

  {#if loading && activities.length === 0}
    <p class="index-status">Surveying the record…</p>
  {:else if error}
    <p class="index-status error">{error}</p>
  {:else if activities.length === 0}
    <p class="index-status">No routes match this filter.</p>
  {:else}
    <div class="atlas-plate manifest-plate">
      <div class="manifest-scroll">
        <table>
          <thead
            ><tr
              ><th>No.</th><th>Date</th><th>Route</th><th>Sport</th><th
                >Distance</th
              ><th>Duration</th><th>Pace</th></tr
            ></thead
          ><tbody>
            {#each activities as activity, index}
              <tr
                class="activity-row"
                role="link"
                tabindex="0"
                onclick={(event) => openActivity(event, activity.id)}
                onkeydown={(event) =>
                  openActivityFromKeyboard(event, activity.id)}
                ><td class="manifest-index"
                  >{String(index + 1).padStart(3, "0")}</td
                ><td>{formatDateOnly(activity.started_at)}</td><td
                  ><a href={`/motion/${activity.id}`}
                    >{activity.title || sportLabel(activity.sport_type)}</a
                  ></td
                ><td>{sportLabel(activity.sport_type)}</td><td
                  >{formatDistance(activity.distance_m)}</td
                ><td
                  >{formatDuration(
                    activity.duration_s ?? activity.moving_time_s,
                  )}</td
                ><td>{formatPace(activity.avg_pace_s_per_km)}</td></tr
              >
            {/each}
          </tbody>
        </table>
      </div>
    </div>
    {#if hasMore}<button
        class="load-more"
        type="button"
        disabled={loadingMore}
        onclick={onLoadMore}
        >{loadingMore ? "Extending the survey…" : "Extend the survey"}</button
      >{/if}
  {/if}

  <footer class="atlas-source">
    Source: imported activity records · presentation only
  </footer>
</section>

<style>
  .atlas-index {
    display: grid;
    gap: 1.5rem;
    font-family: var(--font-sans);
  }
  .atlas-kicker {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.66rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .atlas-kicker::before {
    content: "⌖";
  }
  h1 {
    max-width: 13ch;
    margin: 0;
    font-weight: 600;
    letter-spacing: -0.03em;
    font-size: clamp(2.4rem, 6vw, 4.2rem);
    line-height: 1;
  }
  .index-header {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    border-bottom: 1px solid var(--border);
    padding-bottom: 1.75rem;
  }
  .index-sub {
    max-width: 40rem;
    margin: 1rem 0 0;
    color: var(--text-muted);
    line-height: 1.65;
  }
  .grid-ref {
    display: grid;
    justify-items: end;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    padding: 0.6rem 0.9rem;
    color: var(--accent);
    font-family: var(--font-mono);
    text-align: right;
  }
  .grid-ref span {
    font-size: 1.6rem;
  }
  .grid-ref small {
    margin-top: 0.2rem;
    color: var(--text-muted);
    font-size: 0.6rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }
  .index-filters {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
    border-bottom: 1px solid var(--border);
    padding-bottom: 1rem;
  }
  label {
    display: grid;
    gap: 0.3rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.62rem;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }
  select {
    min-width: 9rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 0.5rem;
    background: var(--surface-1);
    color: var(--text);
    font: inherit;
    font-size: 0.8rem;
  }
  .index-status {
    border: 1px dashed var(--border);
    border-radius: var(--radius);
    padding: 2rem;
    color: var(--text-muted);
  }
  .index-status.error {
    color: var(--sport-run);
  }
  .atlas-plate {
    position: relative;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface-1) 88%, transparent);
  }
  .atlas-plate::before,
  .atlas-plate::after {
    content: "";
    position: absolute;
    width: 0.7rem;
    height: 0.7rem;
    opacity: 0.7;
  }
  .atlas-plate::before {
    top: -1px;
    left: -1px;
    border-top: 2px solid var(--accent);
    border-left: 2px solid var(--accent);
  }
  .atlas-plate::after {
    right: -1px;
    bottom: -1px;
    border-right: 2px solid var(--accent);
    border-bottom: 2px solid var(--accent);
  }
  .manifest-plate {
    padding: 1.5rem;
  }
  .manifest-scroll {
    overflow-x: auto;
  }
  table {
    width: 100%;
    min-width: 48rem;
    border-collapse: collapse;
    font-size: 0.78rem;
  }
  th {
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.62rem;
    font-weight: 400;
    letter-spacing: 0.06em;
    text-align: left;
    text-transform: uppercase;
  }
  th,
  td {
    border-top: 1px solid var(--border);
    padding: 0.85rem 0.75rem;
    text-align: left;
    white-space: nowrap;
  }
  td {
    font-family: var(--font-mono);
  }
  .manifest-index {
    color: var(--accent);
  }
  td:nth-child(3) {
    font-family: var(--font-sans);
    font-weight: 600;
  }
  td a {
    color: var(--text);
    text-decoration: none;
  }
  td a:hover {
    color: var(--accent);
    text-decoration: underline;
  }
  .load-more {
    justify-self: center;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    padding: 0.55rem 1.1rem;
    background: transparent;
    color: var(--accent);
    font: inherit;
    font-size: 0.82rem;
    cursor: pointer;
  }
  .load-more:disabled {
    opacity: 0.5;
  }
  .atlas-source {
    border-top: 1px solid var(--border);
    padding-top: 0.8rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.66rem;
  }
  @media (max-width: 768px) {
    .index-header {
      display: block;
    }
    .grid-ref {
      margin-top: 1.5rem;
      justify-items: start;
      text-align: left;
    }
  }
</style>
