<script lang="ts">
  import type { ActivityThemeProps } from "../../view-contracts/activity-view";
  import ActivityMetricChart from "../components/ActivityMetricChart.svelte";
  import {
    formatDateOnly,
    formatDistance,
    formatDuration,
    formatPace,
  } from "../../format/format";
  import { sportLabel } from "../../domain/sport";

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

<section class="mix-activities" aria-labelledby="mix-activities-title">
  <header class="mix-head">
    <div>
      <p class="mix-kicker">Track list / indexed record</p>
      <h1 id="mix-activities-title">Every session, a track.</h1>
      <p>
        Read movement as a set list of places, durations, and repeated gestures.
      </p>
    </div>
    <div class="mix-readout">
      <strong>{displaySummary.activity_count}</strong><span>sessions</span>
    </div>
  </header>

  {@render children?.()}

  <div class="mix-filters" aria-label="Activity filters">
    <label
      >Sport<select
        value={sportType}
        onchange={(event) =>
          onSportType((event.currentTarget as HTMLSelectElement).value)}
        ><option value="">All sports</option
        >{#each sportOptions as sport (sport)}<option value={sport}
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
    <p class="mix-status">Loading the track list…</p>
  {:else if error}
    <p class="mix-status error">{error}</p>
  {:else if activities.length === 0}
    <p class="mix-status">No sessions match this view.</p>
  {:else}
    <div class="mix-table-wrap">
      <table>
        <thead
          ><tr
            ><th>Tr.</th><th>Date</th><th>Session</th><th>Sport</th><th
              >Distance</th
            ><th>Duration</th><th>Pace</th></tr
          ></thead
        ><tbody>
          {#each activities as activity, index (activity.id)}
            <tr
              class="activity-row"
              role="link"
              tabindex="0"
              onclick={(event) => openActivity(event, activity.id)}
              onkeydown={(event) =>
                openActivityFromKeyboard(event, activity.id)}
              ><td class="track-index">{String(index + 1).padStart(3, "0")}</td
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
    {#if hasMore}
      <button
        class="mix-load-more"
        type="button"
        disabled={loadingMore}
        onclick={onLoadMore}>{loadingMore ? "Loading…" : "Load more"}</button
      >
    {/if}
  {/if}
</section>

<style>
  .mix-activities {
    display: grid;
    gap: 1.35rem;
    min-width: 0;
  }
  .mix-activities > * {
    min-width: 0;
  }
  .mix-kicker {
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-size: 0.66rem;
    letter-spacing: 0.14em;
    text-transform: uppercase;
  }
  h1 {
    max-width: 14ch;
    margin: 0;
    font-size: clamp(2.3rem, 6vw, 4.6rem);
    font-weight: 700;
    letter-spacing: -0.03em;
    line-height: 0.98;
    text-transform: uppercase;
  }
  .mix-head {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    border-bottom: 1px solid var(--border);
    padding-bottom: 1.75rem;
  }
  .mix-head p:last-child {
    max-width: 36rem;
    color: var(--text-muted);
    line-height: 1.6;
  }
  .mix-readout {
    display: grid;
    justify-items: end;
    padding: 0.6rem 0.9rem;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    color: var(--text-muted);
  }
  .mix-readout strong {
    color: var(--accent);
    font-size: 2.6rem;
    font-weight: 700;
    line-height: 1;
    font-variant-numeric: tabular-nums;
  }
  .mix-readout span {
    margin-top: 0.4rem;
    font-size: 0.62rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .mix-filters {
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
    font-size: 0.64rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  select {
    min-width: 9rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 0.5rem;
    background: var(--surface);
    color: var(--text);
    font: inherit;
    font-size: 0.8rem;
  }
  .mix-table-wrap {
    overflow-x: auto;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface) 90%, transparent);
  }
  table {
    width: 100%;
    min-width: 48rem;
    border-collapse: collapse;
    font-size: 0.78rem;
  }
  th {
    color: var(--text-muted);
    font-size: 0.64rem;
    font-weight: 400;
    letter-spacing: 0.08em;
    text-align: left;
    text-transform: uppercase;
  }
  th,
  td {
    border-top: 1px solid var(--border);
    padding: 0.85rem 0.75rem;
    text-align: left;
    white-space: nowrap;
    font-variant-numeric: tabular-nums;
  }
  .track-index {
    color: var(--accent);
  }
  td a {
    color: var(--text);
    text-decoration: none;
  }
  td a:hover {
    color: var(--accent);
    text-decoration: underline;
  }
  .mix-status {
    border: 1px dashed var(--border);
    border-radius: var(--radius);
    padding: 2rem;
    color: var(--text-muted);
  }
  .error {
    color: var(--sport-run);
  }
  .mix-load-more {
    justify-self: center;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    padding: 0.55rem 1.1rem;
    background: transparent;
    color: var(--accent);
    font: inherit;
    font-size: 0.75rem;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    cursor: pointer;
  }
  .mix-load-more:disabled {
    opacity: 0.5;
  }
  @media (max-width: 768px) {
    .mix-head {
      display: block;
    }
    .mix-readout {
      display: flex;
      align-items: baseline;
      justify-items: initial;
      gap: 0.5rem;
      margin-top: 1.5rem;
    }
    .mix-readout strong {
      font-size: 2.2rem;
    }
  }
</style>
