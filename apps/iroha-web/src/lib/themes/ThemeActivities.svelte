<script lang="ts">
  import type { Activity } from "$lib/api";
  import {
    formatDateOnly,
    formatDistance,
    formatDuration,
    formatPace,
  } from "$lib/format";
  import { sportLabel } from "$lib/sport";

  export type ActivitiesVariant =
    "atlas" | "field-journal" | "phenology" | "sound-map" | "archive";
  type Summary = {
    activity_count: number;
    distance_m: number;
    duration_s: number;
  };

  let {
    variant,
    activities,
    displaySummary,
    sportType,
    selectedYear,
    selectedMonth,
    years,
    sportOptions,
    months,
    loading,
    error,
    hasMore,
    loadingMore,
    onSportType,
    onYear,
    onMonth,
    onLoadMore,
  }: {
    variant: ActivitiesVariant;
    activities: Activity[];
    displaySummary: Summary;
    sportType: string;
    selectedYear: string;
    selectedMonth: string;
    years: string[];
    sportOptions: string[];
    months: { value: string; label: string }[];
    loading: boolean;
    error: string | null;
    hasMore: boolean;
    loadingMore: boolean;
    onSportType: (value: string) => void;
    onYear: (value: string) => void;
    onMonth: (value: string) => void;
    onLoadMore: () => void;
  } = $props();
</script>

<section
  class={`theme-activities theme-activities-${variant}`}
  aria-labelledby="theme-activities-title"
>
  <header class="activities-head">
    <div>
      <p class="theme-kicker">Motion / indexed record</p>
      <h1 id="theme-activities-title">Every session leaves a trace.</h1>
      <p>
        Read movement as a collection of places, durations, and repeated
        gestures.
      </p>
    </div>
    <strong>{displaySummary.activity_count}<small> sessions</small></strong>
  </header>
  <div class="activity-filters" aria-label="Activity filters">
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
    <label
      >Year<select
        value={selectedYear}
        onchange={(event) =>
          onYear((event.currentTarget as HTMLSelectElement).value)}
        ><option value="">All years</option>{#each years as year}<option
            value={year}>{year}</option
          >{/each}</select
      ></label
    >
    <label
      >Month<select
        value={selectedMonth}
        onchange={(event) =>
          onMonth((event.currentTarget as HTMLSelectElement).value)}
        ><option value="">All months</option>{#each months as month}<option
            value={month.value}>{month.label}</option
          >{/each}</select
      ></label
    >
  </div>
  {#if loading && activities.length === 0}<p class="activity-status">
      Loading the movement record…
    </p>{:else if error}<p class="activity-status error">
      {error}
    </p>{:else if activities.length === 0}<p class="activity-status">
      No activity sessions match this view.
    </p>{:else}
    <div class="activity-table-wrap">
      <table>
        <thead
          ><tr
            ><th>Date</th><th>Session</th><th>Sport</th><th>Distance</th><th
              >Duration</th
            ><th>Pace</th></tr
          ></thead
        ><tbody>
          {#each activities as activity}
            <tr
              ><td>{formatDateOnly(activity.started_at)}</td><td
                ><a href={`/activities/${activity.id}`}
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
    {#if hasMore}<button
        class="load-more"
        type="button"
        disabled={loadingMore}
        onclick={onLoadMore}>{loadingMore ? "Loading…" : "Load more"}</button
      >{/if}
  {/if}
</section>

<style>
  .theme-activities {
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
  h1 {
    max-width: 13ch;
    margin: 0;
    font-family: Georgia, serif;
    font-size: clamp(2.7rem, 7vw, 5.8rem);
    font-weight: 400;
    letter-spacing: -0.06em;
    line-height: 0.9;
  }
  .activities-head {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    border-bottom: 1px solid var(--border);
    padding-bottom: 2rem;
  }
  .activities-head p:last-child {
    color: var(--text-muted);
    line-height: 1.6;
  }
  .activities-head > strong {
    color: var(--accent);
    font-family: Georgia, serif;
    font-size: 3.5rem;
    font-weight: 400;
    white-space: nowrap;
  }
  .activities-head > strong small {
    color: var(--text-muted);
    font-family: inherit;
    font-size: 0.8rem;
  }
  .activity-filters {
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
    font-size: 0.65rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  select {
    min-width: 9rem;
    border: 1px solid var(--border);
    padding: 0.5rem;
    background: var(--surface-1);
    color: var(--text);
    font: inherit;
    font-size: 0.8rem;
  }
  .activity-table-wrap {
    overflow-x: auto;
    border: 1px solid var(--border);
    background: color-mix(in srgb, var(--surface-1) 88%, transparent);
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
  }
  td:first-child {
    color: var(--accent);
    font-family: Georgia, serif;
  }
  td a {
    color: var(--text);
    text-decoration: none;
  }
  td a:hover {
    color: var(--accent);
    text-decoration: underline;
  }
  .activity-status {
    border: 1px dashed var(--border);
    padding: 2rem;
    color: var(--text-muted);
  }
  .error {
    color: var(--sport-run);
  }
  .load-more {
    justify-self: center;
    border: 1px solid var(--accent);
    padding: 0.55rem 1rem;
    background: transparent;
    color: var(--accent);
    font: inherit;
    cursor: pointer;
  }
  .load-more:disabled {
    opacity: 0.5;
  }
  .theme-activities-atlas {
    font-family: "Avenir Next", sans-serif;
  }
  .theme-activities-atlas .activity-table-wrap {
    border-top: 0.35rem solid var(--accent);
  }
  .theme-activities-phenology h1 {
    font-style: italic;
  }
  .theme-activities-phenology .activity-table-wrap {
    border-radius: 1rem 0.25rem;
  }
  .theme-activities-sound-map {
    font-family: "IBM Plex Mono", monospace;
  }
  .theme-activities-sound-map h1 {
    font-family: inherit;
    letter-spacing: -0.08em;
  }
  .theme-activities-sound-map td a {
    color: var(--accent);
  }
  .theme-activities-archive .activity-table-wrap {
    box-shadow: 0.45rem 0.45rem 0
      color-mix(in srgb, var(--accent) 18%, transparent);
  }
  .theme-activities-archive h1 {
    text-decoration: underline;
    text-decoration-thickness: 1px;
    text-underline-offset: 0.2em;
  }
  @media (max-width: 680px) {
    .activities-head {
      display: block;
    }
    .activities-head > strong {
      display: block;
      margin-top: 1.5rem;
      font-size: 2.6rem;
    }
  }
</style>
