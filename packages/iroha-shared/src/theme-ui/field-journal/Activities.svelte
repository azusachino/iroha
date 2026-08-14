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

<section class="journal-log" aria-labelledby="journal-log-title">
  <header class="log-opening">
    <div>
      <p class="journal-kicker">Field log · indexed record</p>
      <h1 id="journal-log-title">Every session, entered.</h1>
      <p>
        A quiet ledger of movement: where it happened, how long it lasted, and
        what it cost.
      </p>
    </div>
    <div class="log-stamp" aria-label="Total recorded sessions">
      <strong>{displaySummary.activity_count}</strong>
      <span>entries</span>
    </div>
  </header>

  {@render children?.()}

  <div class="journal-rule"><span>the index</span></div>

  <div class="log-filters" aria-label="Log filters">
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
    <p class="log-status">No entries match this view.</p>
  {:else}
    <section class="log-ledger">
      <div class="ledger-scroll">
        <table>
          <thead>
            <tr>
              <th>No.</th>
              <th>Date</th>
              <th>Session</th>
              <th>Sport</th>
              <th>Distance</th>
              <th>Duration</th>
              <th>Pace</th>
            </tr>
          </thead>
          <tbody>
            {#each activities as activity, index (activity.id)}
              <tr
                class="activity-row"
                role="link"
                tabindex="0"
                onclick={(event) => openActivity(event, activity.id)}
                onkeydown={(event) =>
                  openActivityFromKeyboard(event, activity.id)}
              >
                <td class="log-index">{String(index + 1).padStart(3, "0")}</td>
                <td>{formatDateOnly(activity.started_at)}</td>
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
    </section>
    {#if hasMore}
      <button
        class="log-continue"
        type="button"
        disabled={loadingMore}
        onclick={onLoadMore}
      >
        {loadingMore ? "Turning the page…" : "Turn the page for more entries"}
      </button>
    {/if}
  {/if}

  <footer class="journal-source">
    <span>Source: imported activity records</span>
    <span>Presentation only · no readiness score inferred</span>
  </footer>
</section>

<style>
  .journal-log {
    display: grid;
    gap: 1.5rem;
    min-width: 0;
  }
  .journal-log > * {
    min-width: 0;
  }
  .journal-kicker {
    margin: 0 0 0.55rem;
    color: var(--accent);
    font-size: 0.68rem;
    letter-spacing: 0.16em;
    text-transform: uppercase;
  }
  h1 {
    max-width: 12ch;
    margin: 0;
    font-family: var(--font-serif);
    font-weight: 400;
    letter-spacing: -0.05em;
    font-size: clamp(2.6rem, 6vw, 4.6rem);
    line-height: 0.94;
  }
  .log-opening {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
  }
  .log-opening p:last-child {
    max-width: 38rem;
    color: var(--text-muted);
    line-height: 1.7;
  }
  .log-stamp {
    display: grid;
    justify-items: end;
    color: var(--text-muted);
  }
  .log-stamp strong {
    color: var(--accent);
    font-family: var(--font-serif);
    font-size: 3.2rem;
    font-weight: 400;
    line-height: 0.85;
  }
  .log-stamp span {
    margin-top: 0.5rem;
    font-size: 0.68rem;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }
  .journal-rule {
    display: flex;
    align-items: center;
    gap: 1rem;
    color: var(--text-muted);
    font-family: var(--font-serif);
    font-size: 0.8rem;
    font-style: italic;
  }
  .journal-rule::after {
    content: "";
    height: 1px;
    flex: 1;
    background: var(--border);
  }
  .log-filters {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
    border-bottom: 1px solid var(--border);
    padding-bottom: 1rem;
  }
  .log-filters label {
    display: grid;
    gap: 0.3rem;
  }
  .log-filters span {
    color: var(--text-muted);
    font-size: 0.62rem;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }
  .log-filters select {
    min-width: 9rem;
    border: 1px solid var(--border);
    padding: 0.5rem;
    background: color-mix(in srgb, var(--surface-1) 84%, transparent);
    color: var(--text);
    font-family: var(--font-serif);
    font-size: 0.85rem;
  }
  .log-status {
    border: 1px dashed var(--border);
    padding: 2rem;
    color: var(--text-muted);
  }
  .log-status.error {
    color: var(--sport-run);
  }
  .log-ledger {
    border: 1px solid var(--border);
    background: color-mix(in srgb, var(--surface-1) 84%, transparent);
    padding: 1.5rem;
  }
  .ledger-scroll {
    overflow-x: auto;
  }
  table {
    width: 100%;
    min-width: 44rem;
    border-collapse: collapse;
    font-size: 0.8rem;
  }
  th {
    color: var(--text-muted);
    font-size: 0.65rem;
    font-weight: 400;
    letter-spacing: 0.08em;
    text-align: left;
    text-transform: uppercase;
  }
  th,
  td {
    border-top: 1px solid var(--border);
    padding: 0.8rem 0.6rem;
    text-align: left;
    white-space: nowrap;
  }
  .log-index {
    color: var(--accent);
    font-family: var(--font-serif);
  }
  td:nth-child(3) {
    font-family: var(--font-serif);
  }
  td a {
    color: var(--text);
    text-decoration: none;
  }
  td a:hover {
    color: var(--accent);
    text-decoration: underline;
  }
  .log-continue {
    justify-self: center;
    border: 1px solid var(--border);
    padding: 0.6rem 1.2rem;
    background: transparent;
    color: var(--text-muted);
    font-family: var(--font-serif);
    font-style: italic;
    font-size: 0.85rem;
    cursor: pointer;
  }
  .log-continue:hover:not(:disabled) {
    border-color: var(--accent);
    color: var(--accent);
  }
  .log-continue:disabled {
    opacity: 0.5;
    cursor: default;
  }
  .journal-source {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    border-top: 1px solid var(--border);
    padding-top: 1rem;
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  @media (max-width: 768px) {
    .log-opening,
    .journal-source {
      display: block;
    }
    .log-stamp {
      align-items: start;
      justify-items: start;
      margin-top: 1.5rem;
    }
  }
</style>
