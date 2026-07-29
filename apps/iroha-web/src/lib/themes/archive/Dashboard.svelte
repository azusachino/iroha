<script lang="ts">
  import type { Activity, RouteFeatureCollection, Summary } from "$lib/api";
  import { formatDistance, formatDuration, formatDate } from "$lib/format";
  import { sportColor, sportLabel } from "$lib/sport";
  // The geography panel reuses the shared RoutesMap (maplibre) component
  // rather than a bespoke re-implementation -- routes are real geography,
  // and re-drawing a basemap + tile renderer per theme would be pure
  // duplication for no visual gain. Same documented exception atlas,
  // field-journal, phenology, and sound-map took for their Dashboards.
  import RoutesMap from "$lib/components/RoutesMap.svelte";

  let {
    summary,
    activities,
    routes,
    streak,
    loading,
    error,
  }: {
    summary: Summary | null;
    activities: Activity[];
    routes: RouteFeatureCollection | null;
    streak: string;
    loading: boolean;
    error: string | null;
  } = $props();

  const hasRoutes = $derived((routes?.features.length ?? 0) > 0);

  // The sport breakdown becomes a core log: each recorded sport is a
  // stratum, thickness real session share, tone the sport's own semantic
  // hue -- specimen bands drawn from the actual collection, not a legend.
  const sportRows = $derived.by(() => {
    const buckets = (summary?.by_sport ?? [])
      .filter((bucket) => bucket.activity_count > 0)
      .sort((a, b) => b.activity_count - a.activity_count)
      .slice(0, 8);
    const max = Math.max(1, ...buckets.map((bucket) => bucket.activity_count));
    return buckets.map((bucket) => ({
      key: bucket.key,
      count: bucket.activity_count,
      magnitude: Math.max(bucket.activity_count / max, 0.08),
      color: sportColor(bucket.key),
    }));
  });
</script>

<section class="folio-dashboard" aria-labelledby="folio-dashboard-title">
  <header class="folio-head">
    <div>
      <p class="folio-kicker">Long view / vault record</p>
      <h1 id="folio-dashboard-title">See the whole collection.</h1>
      <p>
        Distance, sessions, routes, and the quiet continuity that holds them in
        the same archive.
      </p>
    </div>
    <div class="accession-tag" aria-label={`Streak ${streak}`}>
      {streak} · streak
    </div>
  </header>

  {#if loading}
    <p class="folio-status">Retrieving the long view…</p>
  {:else if error}
    <p class="folio-status error">{error}</p>
  {:else}
    <div class="folio-stats catalog-card">
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

    <div class="folio-grid">
      <section class="folio-panel">
        <header>
          <div>
            <p class="folio-kicker">Recent accessions</p>
            <h2>Movement, lately.</h2>
          </div>
          <span>{activities.length} loaded</span>
        </header>
        <ol>
          {#each activities.slice(0, 8) as activity, index (activity.id)}<li>
              <b>ARC-{String(index + 1).padStart(3, "0")}</b><span
                >{formatDate(activity.started_at)}</span
              ><strong>{activity.title || activity.sport_type}</strong><span
                >{formatDistance(activity.distance_m)} · {formatDuration(
                  activity.duration_s ?? activity.moving_time_s,
                )}</span
              >
            </li>{/each}
        </ol>
        {#if activities.length === 0}
          <p class="folio-empty">No movement sessions recorded yet.</p>
        {/if}
      </section>

      <div class="folio-side">
        <section class="folio-panel">
          <p class="folio-kicker">Specimen bands</p>
          <h2>Sessions by sport</h2>
          {#if sportRows.length}
            <div
              class="core-log"
              role="img"
              aria-label={`Sessions by sport: ${sportRows
                .map((band) => `${sportLabel(band.key)} ${band.count}`)
                .join(", ")}`}
            >
              <div class="core-strip">
                {#each sportRows as band (band.key)}
                  <div
                    class="core-band"
                    style={`flex-grow: ${band.magnitude}; background: ${band.color};`}
                  ></div>
                {/each}
              </div>
              <div class="core-legend">
                {#each sportRows as band (band.key)}
                  <div class="core-row" style={`flex-grow: ${band.magnitude};`}>
                    <strong>{sportLabel(band.key)}</strong>
                    <span>{band.count}</span>
                  </div>
                {/each}
              </div>
            </div>
          {:else}
            <p class="folio-empty">No sport data recorded yet.</p>
          {/if}
        </section>

        <section class="folio-panel">
          <p class="folio-kicker">Geography</p>
          <h2>{routes?.features.length ?? 0} route traces</h2>
          {#if hasRoutes && routes}
            <div class="map-frame"><RoutesMap data={routes} /></div>
          {:else}
            <p class="folio-empty">No routes recorded yet.</p>
          {/if}
        </section>
      </div>
    </div>
  {/if}
  <footer class="folio-source">
    Source: public activity summary · long-view presentation only
  </footer>
</section>

<style>
  .folio-dashboard {
    display: grid;
    gap: 1.3rem;
  }
  .folio-kicker {
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.64rem;
    letter-spacing: 0.15em;
    text-transform: uppercase;
  }
  h1,
  h2 {
    margin: 0;
    font-family: var(--font-serif);
    font-weight: 700;
    letter-spacing: -0.01em;
  }
  h1 {
    max-width: 11ch;
    font-size: clamp(2.5rem, 6.5vw, 5rem);
    line-height: 0.95;
  }
  h2 {
    font-size: 1.4rem;
  }
  .folio-head {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: start;
    border-bottom: 1px solid var(--border);
    padding-bottom: 1.75rem;
  }
  .folio-head p:last-child {
    color: var(--text-muted);
    line-height: 1.6;
  }
  .accession-tag {
    display: inline-flex;
    align-items: center;
    flex-shrink: 0;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    padding: 0.45rem 0.75rem;
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 0.78rem;
    letter-spacing: 0.04em;
    white-space: nowrap;
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
  .catalog-card {
    position: relative;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface) 90%, transparent);
    padding: 1.5rem 1.5rem 1.5rem 1.7rem;
  }
  .catalog-card::before {
    content: "";
    position: absolute;
    left: -1px;
    top: 1.15rem;
    width: 4px;
    height: 2.3rem;
    background: var(--accent-2);
    border-radius: 0 2px 2px 0;
  }
  .folio-stats {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    padding: 0;
  }
  .folio-stats::before {
    display: none;
  }
  .folio-stats div {
    display: grid;
    gap: 0.4rem;
    border-right: 1px solid var(--border);
    padding: 1.1rem;
  }
  .folio-stats div:last-child {
    border-right: 0;
  }
  .folio-stats span {
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.68rem;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  .folio-stats strong {
    font-family: var(--font-serif);
    font-size: 1.5rem;
    font-weight: 700;
  }
  .folio-grid {
    display: grid;
    grid-template-columns: 1.4fr 1fr;
    gap: 1.25rem;
  }
  .folio-side {
    display: grid;
    gap: 1.25rem;
  }
  .folio-panel {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 1.5rem;
    background: color-mix(in srgb, var(--surface) 90%, transparent);
  }
  .folio-panel > header {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .folio-panel > header > span {
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.7rem;
  }
  .folio-panel h2 {
    margin-top: 0.6rem;
  }
  ol {
    margin: 1rem 0 0;
    padding: 0;
    list-style: none;
  }
  li {
    display: grid;
    grid-template-columns: 4.5rem 7rem minmax(0, 1fr) auto;
    gap: 1rem;
    border-top: 1px solid var(--border);
    padding: 0.8rem 0;
    align-items: baseline;
    font-size: 0.75rem;
  }
  li b {
    color: var(--accent);
    font-family: var(--font-mono);
    font-weight: 400;
  }
  li strong {
    font-family: var(--font-serif);
    font-size: 0.92rem;
    font-weight: 700;
  }
  li span,
  .folio-empty {
    color: var(--text-muted);
    font-family: var(--font-mono);
  }
  .core-log {
    display: flex;
    gap: 0.9rem;
    height: 9rem;
    margin: 1.4rem 0 0;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    overflow: hidden;
  }
  .core-strip {
    display: flex;
    flex-direction: column;
    width: 1.4rem;
    flex-shrink: 0;
  }
  .core-band {
    flex-shrink: 0;
    border-top: 1px solid var(--bg);
  }
  .core-band:first-child {
    border-top: 0;
  }
  .core-legend {
    display: flex;
    flex: 1;
    min-width: 0;
    flex-direction: column;
  }
  .core-row {
    display: flex;
    flex-shrink: 0;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    min-height: 1.15rem;
    overflow: hidden;
    border-top: 1px solid var(--border);
    padding: 0 0.6rem 0 0.25rem;
    font-size: 0.68rem;
  }
  .core-row:first-child {
    border-top: 0;
  }
  .core-row strong {
    overflow: hidden;
    font-family: var(--font-serif);
    font-weight: 700;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .core-row span {
    color: var(--text-muted);
    font-family: var(--font-mono);
  }
  .map-frame {
    height: 11rem;
    margin-top: 1.2rem;
    overflow: hidden;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) * 0.7);
  }
  .map-frame :global(.map) {
    height: 100%;
  }
  .folio-source {
    border-top: 1px solid var(--border);
    padding-top: 0.8rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.66rem;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  @media (max-width: 760px) {
    .folio-head,
    .folio-grid {
      display: block;
    }
    .accession-tag {
      margin-top: 1.5rem;
    }
    .folio-stats {
      grid-template-columns: repeat(2, 1fr);
    }
    .folio-stats div:nth-child(2) {
      border-right: 0;
    }
    .folio-stats div:nth-child(-n + 2) {
      border-bottom: 1px solid var(--border);
    }
    .folio-panel + .folio-panel,
    .folio-side {
      margin-top: 1.25rem;
    }
    li {
      grid-template-columns: 1fr;
      gap: 0.25rem;
    }
  }
</style>
