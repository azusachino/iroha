<script lang="ts">
  import type { Activity, RouteFeatureCollection, Summary } from "$lib/api";
  import { formatDistance, formatDuration, formatDate } from "$lib/format";
  import { sportColor, sportLabel } from "$lib/sport";
  // The geography panel reuses the shared RoutesMap (maplibre) component
  // rather than a bespoke re-implementation -- routes are real geography,
  // and re-drawing a basemap + tile renderer per theme would be pure
  // duplication for no visual gain. Same documented exception atlas,
  // field-journal, and phenology took for their Dashboards.
  import RoutesMap from "$lib/components/RoutesMap.svelte";
  import RetryNotice from "$lib/components/RetryNotice.svelte";

  let {
    summary,
    activities,
    routes,
    streak,
    loading,
    error,
    onRetry,
  }: {
    summary: Summary | null;
    activities: Activity[];
    routes: RouteFeatureCollection | null;
    streak: string;
    loading: boolean;
    error: string | null;
    onRetry: () => void;
  } = $props();

  const hasRoutes = $derived((routes?.features.length ?? 0) > 0);

  // A per-sport spectrum: each recorded sport becomes one band, height set
  // by its share of loaded sessions, colored with the same sport hues used
  // on activity cards elsewhere -- a multi-band level meter built from a
  // real aggregate breakdown rather than a decorative equalizer graphic.
  const spectrum = $derived.by(() => {
    const buckets = (summary?.by_sport ?? [])
      .filter((bucket) => bucket.activity_count > 0)
      .sort((a, b) => b.activity_count - a.activity_count)
      .slice(0, 8);
    const max = Math.max(1, ...buckets.map((bucket) => bucket.activity_count));
    return buckets.map((bucket) => ({
      key: bucket.key,
      count: bucket.activity_count,
      pct: (bucket.activity_count / max) * 100,
      color: sportColor(bucket.key),
    }));
  });
</script>

<section class="mix-dashboard" aria-labelledby="mix-dashboard-title">
  <header class="mix-head">
    <div>
      <p class="mix-kicker">Long view / mix session</p>
      <h1 id="mix-dashboard-title">See the whole session.</h1>
      <p>Distance, sessions, routes, and the quiet continuity between them.</p>
    </div>
    <div class="mix-readout">
      <strong>{streak}</strong><span>streak</span>
    </div>
  </header>

  {#if loading}
    <p class="mix-status">Loading the long view…</p>
  {:else if error}
    <RetryNotice message={error} {onRetry} />
  {:else}
    <div class="mix-stats">
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

    <div class="mix-grid">
      <section class="mix-panel">
        <header>
          <div>
            <p class="mix-kicker">Track list</p>
            <h2>Movement, lately.</h2>
          </div>
          <span
            >{Math.min(8, activities.length)} of {activities.length} loaded</span
          >
        </header>
        <ol>
          {#each activities.slice(0, 8) as activity, index (activity.id)}<li>
              <b>{String(index + 1).padStart(2, "0")}</b><span
                >{formatDate(activity.started_at)}</span
              ><strong>{activity.title || activity.sport_type}</strong><span
                >{formatDistance(activity.distance_m)} · {formatDuration(
                  activity.duration_s ?? activity.moving_time_s,
                )}</span
              >
            </li>{/each}
        </ol>
        {#if activities.length === 0}
          <p class="mix-empty">No movement sessions recorded yet.</p>
        {/if}
      </section>

      <div class="mix-side">
        <section class="mix-panel">
          <p class="mix-kicker">Sport spectrum</p>
          <h2>Sessions by band</h2>
          {#if spectrum.length}
            <div
              class="mix-spectrum"
              role="img"
              aria-label={`Sessions by sport: ${spectrum
                .map((band) => `${sportLabel(band.key)} ${band.count}`)
                .join(", ")}`}
            >
              {#each spectrum as band (band.key)}
                <div class="spectrum-band">
                  <i
                    style={`height: ${Math.max(4, band.pct)}%; background: ${band.color};`}
                  ></i>
                  <small>{sportLabel(band.key)}</small>
                </div>
              {/each}
            </div>
          {:else}
            <p class="mix-empty">No sport data recorded yet.</p>
          {/if}
        </section>

        <section class="mix-panel">
          <p class="mix-kicker">Geography</p>
          <h2>{routes?.features.length ?? 0} route traces</h2>
          {#if hasRoutes && routes}
            <div class="map-frame"><RoutesMap data={routes} /></div>
          {:else}
            <p class="mix-empty">No routes recorded yet.</p>
          {/if}
        </section>
      </div>
    </div>
  {/if}
  <footer class="mix-source">
    <span>Source: public activity summary</span>
    <span>Long-view presentation only</span>
  </footer>
</section>

<style>
  .mix-dashboard {
    display: grid;
    gap: 1.35rem;
  }
  .mix-kicker {
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-size: 0.66rem;
    letter-spacing: 0.14em;
    text-transform: uppercase;
  }
  h1,
  h2 {
    margin: 0;
    font-weight: 700;
    letter-spacing: -0.03em;
  }
  h1 {
    max-width: 11ch;
    font-size: clamp(2.3rem, 6vw, 4.2rem);
    line-height: 0.98;
    text-transform: uppercase;
  }
  h2 {
    font-size: 1.5rem;
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
    font-size: 2.4rem;
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
  .mix-status {
    border: 1px dashed var(--border);
    border-radius: var(--radius);
    padding: 2rem;
    color: var(--text-muted);
  }
  .mix-stats {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    border: 1px solid var(--border);
    border-radius: var(--radius);
  }
  .mix-stats div {
    display: grid;
    gap: 0.4rem;
    border-right: 1px solid var(--border);
    padding: 1.1rem;
  }
  .mix-stats div:last-child {
    border-right: 0;
  }
  .mix-stats span {
    color: var(--text-muted);
    font-size: 0.7rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  .mix-stats strong {
    font-size: 1.5rem;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
  }
  .mix-grid {
    display: grid;
    grid-template-columns: 1.4fr 1fr;
    gap: 1.25rem;
  }
  .mix-side {
    display: grid;
    gap: 1.25rem;
  }
  .mix-panel {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 1.4rem;
    background: color-mix(in srgb, var(--surface) 90%, transparent);
  }
  .mix-panel > header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .mix-panel > header > span {
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
    grid-template-columns: 2rem 7rem minmax(0, 1fr) auto;
    gap: 1rem;
    border-top: 1px solid var(--border);
    padding: 0.8rem 0;
    align-items: baseline;
    font-size: 0.75rem;
  }
  li b {
    color: var(--accent);
    font-weight: 700;
  }
  li strong {
    font-size: 0.9rem;
    font-weight: 600;
  }
  li span,
  .mix-empty {
    color: var(--text-muted);
  }
  .mix-panel h2 {
    margin-top: 0.6rem;
  }
  .mix-spectrum {
    display: flex;
    align-items: end;
    gap: 0.5rem;
    height: 8rem;
    margin: 1.4rem 0 0;
    border-bottom: 1px solid var(--border);
  }
  .spectrum-band {
    display: grid;
    flex: 1 1 0;
    grid-template-rows: 1fr auto;
    align-items: end;
    height: 100%;
    min-width: 0;
  }
  .spectrum-band i {
    display: block;
    width: 100%;
    min-height: 0.2rem;
    border-radius: 1px 1px 0 0;
  }
  .spectrum-band small {
    overflow: hidden;
    margin-top: 0.4rem;
    color: var(--text-muted);
    font-size: 0.55rem;
    text-align: center;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .map-frame {
    height: 11rem;
    margin-top: 1.2rem;
    overflow: hidden;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) * 0.6);
  }
  .map-frame :global(.map) {
    height: 100%;
  }
  .mix-source {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    border-top: 1px solid var(--border);
    padding-top: 0.8rem;
    color: var(--text-muted);
    font-size: 0.66rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  @media (max-width: 760px) {
    .mix-head,
    .mix-grid,
    .mix-source {
      display: block;
    }
    .mix-readout {
      display: flex;
      align-items: baseline;
      justify-items: initial;
      gap: 0.5rem;
      margin-top: 1.5rem;
    }
    .mix-stats {
      grid-template-columns: repeat(2, 1fr);
    }
    .mix-stats div:nth-child(2) {
      border-right: 0;
    }
    .mix-stats div:nth-child(-n + 2) {
      border-bottom: 1px solid var(--border);
    }
    .mix-panel + .mix-panel,
    .mix-side {
      margin-top: 1.25rem;
    }
    li {
      grid-template-columns: 1fr;
      gap: 0.25rem;
    }
  }
</style>
