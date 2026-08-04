<script lang="ts">
  import type { Activity, RouteFeatureCollection, Summary } from "$lib/api";
  import { formatDistance, formatDuration, formatDate } from "$lib/format";

  export type DashboardVariant =
    "atlas" | "field-journal" | "phenology" | "sound-map" | "archive";
  let {
    variant,
    summary,
    activities,
    routes,
    streak,
    loading,
    error,
  }: {
    variant: DashboardVariant;
    summary: Summary | null;
    activities: Activity[];
    routes: RouteFeatureCollection | null;
    streak: string;
    loading: boolean;
    error: string | null;
  } = $props();
</script>

<section
  class={`theme-dashboard theme-dashboard-${variant}`}
  aria-labelledby="theme-dashboard-title"
>
  <header class="dashboard-head">
    <div>
      <p class="theme-kicker">Observatory / long view</p>
      <h1 id="theme-dashboard-title">See the footprint.</h1>
      <p>
        The archive at a glance: distance, sessions, routes, and the quiet
        continuity between them.
      </p>
    </div>
    <strong>{streak}<small> streak</small></strong>
  </header>
  {#if loading}<p class="dashboard-status">
      Loading the long view…
    </p>{:else if error}<p class="dashboard-status error">{error}</p>{:else}
    <div class="dashboard-stats">
      <div>
        <span>Distance</span><strong
          >{formatDistance(summary?.totals.distance_m)}</strong
        >
      </div>
      <div>
        <span>Activities</span><strong
          >{summary?.totals.activity_count ?? "—"}</strong
        >
      </div>
      <div>
        <span>Total time</span><strong
          >{formatDuration(
            summary?.totals.moving_time_s || summary?.totals.duration_s,
          )}</strong
        >
      </div>
      <div>
        <span>Routes</span><strong>{routes?.features.length ?? "—"}</strong>
      </div>
    </div>
    <div class="dashboard-grid">
      <section class="recent-panel">
        <header>
          <div>
            <p class="theme-kicker">Recent entries</p>
            <h2>Movement, lately.</h2>
          </div>
          <span
            >{Math.min(8, activities.length)} of {activities.length} loaded</span
          >
        </header>
        <ol>
          {#each activities.slice(0, 8) as activity}<li>
              <b>{formatDate(activity.started_at)}</b><strong
                >{activity.title || activity.sport_type}</strong
              ><span
                >{formatDistance(activity.distance_m)} · {formatDuration(
                  activity.duration_s ?? activity.moving_time_s,
                )}</span
              >
            </li>{/each}
        </ol>
      </section>
      <section class="route-panel">
        <p class="theme-kicker">Geography</p>
        <h2>{routes?.features.length ?? 0} route traces</h2>
        <div class="route-signal" aria-hidden="true">
          <i></i><i></i><i></i><i></i><i></i>
        </div>
        <p>
          Routes stay linked to their source activity and remain inspectable
          from the archive.
        </p>
      </section>
    </div>
  {/if}
  <footer class="dashboard-source">
    Source: public activity summary · long-view presentation only
  </footer>
</section>

<style>
  .theme-dashboard {
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
    max-width: 11ch;
    font-size: clamp(2.8rem, 7vw, 6rem);
    line-height: 0.9;
  }
  h2 {
    font-size: 1.7rem;
  }
  .dashboard-head {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    border-bottom: 1px solid var(--border);
    padding-bottom: 2rem;
  }
  .dashboard-head p:last-child {
    color: var(--text-muted);
    line-height: 1.6;
  }
  .dashboard-head > strong {
    color: var(--accent);
    font-family: Georgia, serif;
    font-size: 3.4rem;
    font-weight: 400;
    white-space: nowrap;
  }
  .dashboard-head > strong small {
    color: var(--text-muted);
    font-family: inherit;
    font-size: 0.8rem;
  }
  .dashboard-stats {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    border: 1px solid var(--border);
  }
  .dashboard-stats div {
    display: grid;
    gap: 0.4rem;
    border-right: 1px solid var(--border);
    padding: 1.1rem;
  }
  .dashboard-stats div:last-child {
    border-right: 0;
  }
  .dashboard-stats span {
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  .dashboard-stats strong {
    font-family: Georgia, serif;
    font-size: 1.5rem;
    font-weight: 400;
  }
  .dashboard-grid {
    display: grid;
    grid-template-columns: 1.4fr 1fr;
    gap: 1.25rem;
  }
  .recent-panel,
  .route-panel {
    border: 1px solid var(--border);
    padding: 1.5rem;
    background: color-mix(in srgb, var(--surface-1) 88%, transparent);
  }
  .recent-panel header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .recent-panel header > span {
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  ol {
    margin: 1rem 0 0;
    padding: 0;
    list-style: none;
  }
  li {
    display: grid;
    grid-template-columns: 7rem minmax(0, 1fr) auto;
    gap: 1rem;
    border-top: 1px solid var(--border);
    padding: 0.8rem 0;
    align-items: baseline;
    font-size: 0.75rem;
  }
  li b {
    color: var(--accent);
    font-weight: 400;
  }
  li strong {
    font-family: Georgia, serif;
    font-size: 0.95rem;
    font-weight: 400;
  }
  li span,
  .route-panel p:last-child {
    color: var(--text-muted);
  }
  .route-panel h2 {
    margin-top: 1rem;
  }
  .route-signal {
    display: flex;
    align-items: center;
    gap: 0.35rem;
    height: 10rem;
    margin: 1.5rem 0;
    border-bottom: 1px solid var(--border);
  }
  .route-signal i {
    flex: 1;
    background: var(--accent);
    transform: skewY(-25deg);
  }
  .route-signal i:nth-child(1) {
    height: 25%;
  }
  .route-signal i:nth-child(2) {
    height: 65%;
  }
  .route-signal i:nth-child(3) {
    height: 40%;
  }
  .route-signal i:nth-child(4) {
    height: 90%;
  }
  .route-signal i:nth-child(5) {
    height: 55%;
  }
  .dashboard-status {
    border: 1px dashed var(--border);
    padding: 2rem;
    color: var(--text-muted);
  }
  .error {
    color: var(--sport-run);
  }
  .dashboard-source {
    border-top: 1px solid var(--border);
    padding-top: 0.8rem;
    color: var(--text-muted);
    font-size: 0.68rem;
  }
  .theme-dashboard-atlas {
    font-family: "Avenir Next", sans-serif;
  }
  .theme-dashboard-atlas .route-panel {
    border-top: 0.35rem solid var(--accent);
  }
  .theme-dashboard-phenology h1,
  .theme-dashboard-phenology h2 {
    font-style: italic;
  }
  .theme-dashboard-sound-map {
    font-family: "IBM Plex Mono", monospace;
  }
  .theme-dashboard-sound-map h1,
  .theme-dashboard-sound-map h2 {
    font-family: inherit;
  }
  .theme-dashboard-archive .recent-panel {
    box-shadow: 0.45rem 0.45rem 0
      color-mix(in srgb, var(--accent) 18%, transparent);
  }
  @media (max-width: 760px) {
    .dashboard-head,
    .dashboard-grid {
      display: block;
    }
    .dashboard-head > strong {
      display: block;
      margin-top: 1.5rem;
      font-size: 2.6rem;
    }
    .dashboard-stats {
      grid-template-columns: repeat(2, 1fr);
    }
    .dashboard-stats div:nth-child(2) {
      border-right: 0;
    }
    .dashboard-stats div:nth-child(-n + 2) {
      border-bottom: 1px solid var(--border);
    }
    .recent-panel {
      margin-bottom: 1.25rem;
    }
    li {
      grid-template-columns: 1fr;
      gap: 0.25rem;
    }
  }
</style>
