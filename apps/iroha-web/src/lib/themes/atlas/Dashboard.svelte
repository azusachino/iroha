<script lang="ts">
  import type {
    Activity,
    MediaAggregates,
    RouteFeatureCollection,
    Summary,
  } from "$lib/api";
  import { formatDistance, formatDuration, formatDate } from "$lib/format";
  // The geography panel reuses the shared RoutesMap (maplibre) component
  // rather than a bespoke re-implementation: atlas is the theme built around
  // real routes and places, and re-drawing a basemap + tile renderer per
  // theme would be pure duplication for no visual gain. Same documented
  // exception field-journal took for its Dashboard.
  import RouteFootprint from "$lib/components/RouteFootprint.svelte";
  import RetryNotice from "$lib/components/RetryNotice.svelte";

  let {
    summary,
    activities,
    routes,
    streak,
    loading,
    error,
    onRetry,
    routesLoading,
    routesError,
    onLoadRoutes,
    sleepSummary,
    mediaAggregates,
  }: {
    summary: Summary | null;
    activities: Activity[];
    routes: RouteFeatureCollection | null;
    streak: string;
    loading: boolean;
    error: string | null;
    onRetry: () => void;
    routesLoading: boolean;
    routesError: string | null;
    onLoadRoutes: () => void;
    sleepSummary: {
      averageAsleepS: number;
      averageEfficiency: number;
      nightCount: number;
    };
    mediaAggregates: MediaAggregates | null;
  } = $props();
</script>

<section class="atlas-master" aria-labelledby="atlas-master-title">
  <header class="master-header">
    <div>
      <p class="atlas-kicker">Master sheet · standing survey</p>
      <h1 id="atlas-master-title">The whole territory.</h1>
      <p class="master-sub">
        Distance, sessions, and routes read together — the full atlas at reduced
        scale.
      </p>
    </div>
    <div class="grid-ref">
      <span>{streak}</span>
      <small>day streak</small>
    </div>
  </header>

  {#if loading}
    <p class="master-status">Compiling the master sheet…</p>
  {:else if error}
    <RetryNotice message={error} {onRetry} />
  {:else}
    <div class="master-stats">
      <div class="atlas-plate">
        <p class="atlas-kicker">Distance</p>
        <strong>{formatDistance(summary?.totals.distance_m)}</strong>
      </div>
      <div class="atlas-plate">
        <p class="atlas-kicker">Activities</p>
        <strong>{summary?.totals.activity_count ?? "—"}</strong>
      </div>
      <div class="atlas-plate">
        <p class="atlas-kicker">Total time</p>
        <strong
          >{formatDuration(
            summary?.totals.moving_time_s || summary?.totals.duration_s,
          )}</strong
        >
      </div>
      <div class="atlas-plate">
        <p class="atlas-kicker">Routes</p>
        <strong>{routes?.features.length ?? "—"}</strong>
      </div>
      <div class="atlas-plate">
        <p class="atlas-kicker">Sleep</p>
        <strong
          >{sleepSummary.nightCount
            ? formatDuration(sleepSummary.averageAsleepS)
            : "—"}</strong
        >
      </div>
      <div class="atlas-plate">
        <p class="atlas-kicker">Media</p>
        <strong>{mediaAggregates?.totals.completed_count ?? "—"}</strong>
      </div>
    </div>

    <div class="master-grid">
      <section class="atlas-plate entries-plate">
        <header class="entries-heading">
          <div>
            <p class="atlas-kicker">Route log</p>
            <h2>Movement, lately.</h2>
          </div>
          <span
            >{Math.min(8, activities.length)} of {activities.length} loaded</span
          >
        </header>
        <ol class="waypoint-list">
          {#each activities.slice(0, 8) as activity, index}<li>
              <span class="waypoint-index"
                >{String(index + 1).padStart(2, "0")}</span
              ><span>{formatDate(activity.started_at)}</span><strong
                >{activity.title || activity.sport_type}</strong
              ><span
                >{formatDistance(activity.distance_m)} · {formatDuration(
                  activity.duration_s ?? activity.moving_time_s,
                )}</span
              >
            </li>{/each}
        </ol>
        {#if activities.length === 0}
          <p class="atlas-empty">No movement sessions recorded yet.</p>
        {/if}
      </section>
      <section class="atlas-plate geo-plate">
        <p class="atlas-kicker">Geography</p>
        <h2>
          {routes
            ? `${routes.features.length} route traces`
            : "Route footprint"}
        </h2>
        <div class="map-frame">
          <RouteFootprint
            {routes}
            loading={routesLoading}
            error={routesError}
            onLoad={onLoadRoutes}
          />
        </div>
        <p>
          Routes stay linked to their source activity and remain inspectable
          from the record.
        </p>
      </section>
    </div>
  {/if}
  <footer class="atlas-source">
    Source: public activity summary · master-sheet presentation only
  </footer>
</section>

<style>
  .atlas-master {
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
  h1,
  h2 {
    margin: 0;
    font-weight: 600;
    letter-spacing: -0.03em;
  }
  h1 {
    max-width: 11ch;
    font-size: clamp(2.5rem, 6vw, 4.6rem);
    line-height: 1;
  }
  h2 {
    font-size: 1.45rem;
  }
  .master-header {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    border-bottom: 1px solid var(--border);
    padding-bottom: 1.75rem;
  }
  .master-sub {
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
  .master-status {
    border: 1px dashed var(--border);
    border-radius: var(--radius);
    padding: 2rem;
    color: var(--text-muted);
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
  .master-stats {
    display: grid;
    grid-template-columns: repeat(6, 1fr);
    gap: 1rem;
  }
  .master-stats .atlas-plate {
    padding: 1.1rem;
  }
  .master-stats strong {
    display: block;
    margin-top: 0.35rem;
    font-family: var(--font-mono);
    font-size: 1.4rem;
    font-weight: 600;
  }
  .master-grid {
    display: grid;
    grid-template-columns: 1.4fr 1fr;
    gap: 1.25rem;
  }
  .entries-plate,
  .geo-plate {
    padding: 1.5rem;
  }
  .entries-heading {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .entries-heading > span {
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.7rem;
  }
  .waypoint-list {
    margin: 1rem 0 0;
    padding: 0;
    list-style: none;
  }
  .waypoint-list li {
    display: grid;
    grid-template-columns: 2.1rem 7rem minmax(0, 1fr) auto;
    gap: 1rem;
    align-items: baseline;
    border-top: 1px solid var(--border);
    padding: 0.8rem 0;
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  .waypoint-index {
    display: inline-block;
    border: 1px solid var(--accent);
    border-radius: 50%;
    padding: 0.15rem 0;
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.62rem;
    text-align: center;
  }
  .waypoint-list strong {
    color: var(--text);
    font-size: 0.9rem;
    font-weight: 600;
  }
  .atlas-empty {
    margin-top: 1rem;
    color: var(--text-muted);
  }
  .geo-plate h2 {
    margin-top: 0.75rem;
  }
  .map-frame {
    height: 12rem;
    margin: 1.25rem 0;
    overflow: hidden;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) * 0.6);
  }
  .map-frame :global(.map) {
    height: 100%;
  }
  .geo-plate > p:last-child {
    color: var(--text-muted);
  }
  .atlas-source {
    border-top: 1px solid var(--border);
    padding-top: 0.8rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.66rem;
  }
  @media (max-width: 760px) {
    .master-header,
    .master-grid {
      display: block;
    }
    .grid-ref {
      margin-top: 1.5rem;
      justify-items: start;
      text-align: left;
    }
    .master-stats {
      grid-template-columns: repeat(2, 1fr);
    }
    .entries-plate {
      margin-bottom: 1.25rem;
    }
    .waypoint-list li {
      grid-template-columns: 1fr;
      gap: 0.25rem;
    }
  }
</style>
