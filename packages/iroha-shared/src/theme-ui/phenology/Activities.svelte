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

  // Meteorological (northern-hemisphere) season, purely a presentation tag
  // over the real started_at date -- phenology is literally the study of
  // seasonal timing, so the ledger keeps that framing visible.
  const SEASONS = [
    "winter",
    "winter",
    "spring",
    "spring",
    "spring",
    "summer",
    "summer",
    "summer",
    "autumn",
    "autumn",
    "autumn",
    "winter",
  ] as const;

  function seasonOf(iso: string): (typeof SEASONS)[number] | null {
    const month = new Date(iso).getMonth();
    return Number.isNaN(month) ? null : SEASONS[month];
  }
</script>

<section class="bloom-log" aria-labelledby="bloom-log-title">
  <header class="log-opening">
    <div>
      <p class="bloom-kicker">◕ Field record · indexed</p>
      <h1 id="bloom-log-title">Every session, its season.</h1>
      <p>
        Movement read as a cycle of entries, each one placed back into the turn
        of the year it happened in.
      </p>
    </div>
    <div class="log-count">
      <strong>{displaySummary.activity_count}</strong>
      <span>sessions</span>
    </div>
  </header>

  {@render children?.()}

  <div class="log-filters" aria-label="Activity filters">
    <label>
      <span>Sport</span>
      <select
        value={sportType}
        onchange={(event) =>
          onSportType((event.currentTarget as HTMLSelectElement).value)}
      >
        <option value="">All sports</option>
        {#each sportOptions as sport (sport)}
          <option value={sport}>{sportLabel(sport)}</option>
        {/each}
      </select>
    </label>
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
    <p class="log-status">Gathering the record…</p>
  {:else if error}
    <p class="log-status error">{error}</p>
  {:else if activities.length === 0}
    <p class="log-status">No sessions match this view.</p>
  {:else}
    <div class="log-wrap">
      <table>
        <thead>
          <tr>
            <th>Date</th>
            <th>Season</th>
            <th>Session</th>
            <th>Sport</th>
            <th>Distance</th>
            <th>Duration</th>
            <th>Pace</th>
          </tr>
        </thead>
        <tbody>
          {#each activities as activity (activity.id)}
            <tr
              class="activity-row"
              role="link"
              tabindex="0"
              onclick={(event) => openActivity(event, activity.id)}
              onkeydown={(event) =>
                openActivityFromKeyboard(event, activity.id)}
            >
              <td>{formatDateOnly(activity.started_at)}</td>
              <td>
                {#if seasonOf(activity.started_at)}
                  <span
                    class={`season-tag season-${seasonOf(activity.started_at)}`}
                    >{seasonOf(activity.started_at)}</span
                  >
                {:else}
                  <span class="season-tag">—</span>
                {/if}
              </td>
              <td>
                <a href={`/motion/${activity.id}`}
                  >{activity.title || sportLabel(activity.sport_type)}</a
                >
              </td>
              <td>{sportLabel(activity.sport_type)}</td>
              <td>{formatDistance(activity.distance_m)}</td>
              <td
                >{formatDuration(
                  activity.duration_s ?? activity.moving_time_s,
                )}</td
              >
              <td>{formatPace(activity.avg_pace_s_per_km)}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
    {#if hasMore}
      <button
        class="load-more"
        type="button"
        disabled={loadingMore}
        onclick={onLoadMore}>{loadingMore ? "Gathering…" : "Load more"}</button
      >
    {/if}
  {/if}
</section>

<style>
  .bloom-log {
    display: grid;
    gap: 1.5rem;
    font-family: var(--font-serif);
  }
  .bloom-kicker {
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-size: 0.66rem;
    letter-spacing: 0.15em;
    text-transform: uppercase;
  }
  h1 {
    max-width: 13ch;
    margin: 0;
    font-size: clamp(2.4rem, 6vw, 4.6rem);
    font-weight: 400;
    letter-spacing: -0.02em;
    line-height: 0.95;
  }
  .log-opening {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    border-bottom: 1px solid var(--border);
    padding-bottom: 1.75rem;
  }
  .log-opening p:last-child {
    max-width: 36rem;
    color: var(--text-muted);
    line-height: 1.65;
  }
  .log-count {
    display: grid;
    justify-items: end;
    color: var(--text-muted);
  }
  .log-count strong {
    color: var(--accent);
    font-style: italic;
    font-size: 3.2rem;
    font-weight: 400;
    white-space: nowrap;
  }
  .log-count span {
    margin-top: 0.4rem;
    font-size: 0.68rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .log-filters {
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
    border-radius: calc(var(--radius) * 0.4);
    padding: 0.5rem;
    background: var(--surface);
    color: var(--text);
    font: inherit;
    font-size: 0.8rem;
  }
  .log-status {
    border: 1px dashed var(--border);
    border-radius: var(--radius);
    padding: 2rem;
    color: var(--text-muted);
  }
  .log-status.error {
    color: var(--sport-run);
  }
  .log-wrap {
    overflow-x: auto;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface) 90%, transparent);
  }
  table {
    width: 100%;
    min-width: 52rem;
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
    font-style: italic;
  }
  td a {
    color: var(--text);
    text-decoration: none;
  }
  td a:hover {
    color: var(--accent);
    text-decoration: underline;
  }
  .season-tag {
    display: inline-block;
    border-radius: 999px;
    padding: 0.2rem 0.6rem;
    color: var(--text-muted);
    font-size: 0.66rem;
    text-transform: capitalize;
    background: color-mix(in srgb, var(--border) 60%, transparent);
  }
  .season-spring {
    background: color-mix(in srgb, var(--accent-2) 22%, transparent);
    color: color-mix(in srgb, var(--accent-2) 75%, var(--text));
  }
  .season-summer {
    background: color-mix(in srgb, #e7b65a 25%, transparent);
    color: color-mix(in srgb, #e7b65a 70%, var(--text));
  }
  .season-autumn {
    background: color-mix(in srgb, var(--accent) 22%, transparent);
    color: color-mix(in srgb, var(--accent) 75%, var(--text));
  }
  .season-winter {
    background: color-mix(in srgb, var(--text-muted) 22%, transparent);
    color: var(--text-muted);
  }
  .load-more {
    justify-self: center;
    border: 1px solid var(--accent);
    border-radius: 999px;
    padding: 0.55rem 1.2rem;
    background: transparent;
    color: var(--accent);
    font: inherit;
    cursor: pointer;
  }
  .load-more:disabled {
    opacity: 0.5;
  }
  @media (max-width: 680px) {
    .log-opening {
      display: block;
    }
    .log-count {
      display: block;
      margin-top: 1.5rem;
    }
    .log-count strong {
      font-size: 2.4rem;
    }
  }
</style>
