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

<section class="folio-activities" aria-labelledby="folio-activities-title">
  <header class="folio-head">
    <div>
      <p class="folio-kicker">Accession register / motion</p>
      <h1 id="folio-activities-title">Every session, catalogued.</h1>
      <p>
        Read movement as a collection: each session accessioned in the order it
        entered the record.
      </p>
    </div>
    <div class="folio-readout">
      <strong>{displaySummary.activity_count}</strong><span>sessions held</span>
    </div>
  </header>

  {@render children?.()}

  <div class="folio-filters" aria-label="Activity filters">
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
    <p class="folio-status">Retrieving the register…</p>
  {:else if error}
    <p class="folio-status error">{error}</p>
  {:else if activities.length === 0}
    <p class="folio-status">No accessioned sessions match this view.</p>
  {:else}
    <div class="folio-table-wrap">
      <table>
        <thead
          ><tr
            ><th>Acc. no.</th><th>Date</th><th>Session</th><th>Sport</th><th
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
              ><td class="folio-index"
                >ARC-{String(index + 1).padStart(4, "0")}</td
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
        class="folio-load-more"
        type="button"
        disabled={loadingMore}
        onclick={onLoadMore}>{loadingMore ? "Loading…" : "Load more"}</button
      >
    {/if}
  {/if}
</section>

<style>
  .folio-activities {
    display: grid;
    gap: 1.3rem;
    min-width: 0;
  }
  .folio-activities > * {
    min-width: 0;
  }
  .folio-kicker {
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.64rem;
    letter-spacing: 0.15em;
    text-transform: uppercase;
  }
  h1 {
    max-width: 13ch;
    margin: 0;
    font-family: var(--font-serif);
    font-size: clamp(2.5rem, 6.5vw, 5.2rem);
    font-weight: 700;
    letter-spacing: -0.01em;
    line-height: 0.95;
  }
  .folio-head {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    border-bottom: 1px solid var(--border);
    padding-bottom: 1.75rem;
  }
  .folio-head p:last-child {
    max-width: 36rem;
    color: var(--text-muted);
    line-height: 1.6;
  }
  .folio-readout {
    display: grid;
    justify-items: end;
    color: var(--text-muted);
  }
  .folio-readout strong {
    color: var(--accent);
    font-family: var(--font-serif);
    font-size: 3.3rem;
    font-weight: 700;
    white-space: nowrap;
  }
  .folio-readout span {
    margin-top: 0.5rem;
    font-family: var(--font-mono);
    font-size: 0.64rem;
    text-transform: uppercase;
  }
  .folio-filters {
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
  .folio-table-wrap {
    overflow-x: auto;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface) 90%, transparent);
  }
  table {
    width: 100%;
    min-width: 50rem;
    border-collapse: collapse;
    font-family: var(--font-mono);
    font-size: 0.76rem;
  }
  th {
    color: var(--text-muted);
    font-size: 0.62rem;
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
  }
  .folio-index {
    color: var(--accent);
  }
  td a {
    color: var(--text);
    font-family: var(--font-serif);
    text-decoration: none;
  }
  td a:hover {
    color: var(--accent);
    text-decoration: underline;
  }
  .folio-status {
    border: 1px dashed var(--border);
    border-radius: var(--radius);
    padding: 2rem;
    color: var(--text-muted);
  }
  .error {
    color: var(--sport-run);
  }
  .folio-load-more {
    justify-self: center;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    padding: 0.55rem 1.1rem;
    background: transparent;
    color: var(--accent);
    font: inherit;
    font-family: var(--font-mono);
    font-size: 0.75rem;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    cursor: pointer;
  }
  .folio-load-more:disabled {
    opacity: 0.5;
  }
  @media (max-width: 768px) {
    .folio-head {
      display: block;
    }
    .folio-readout {
      display: block;
      margin-top: 1.5rem;
    }
    .folio-readout strong {
      font-size: 2.6rem;
    }
  }
</style>
