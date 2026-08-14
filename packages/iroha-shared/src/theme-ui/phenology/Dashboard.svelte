<script lang="ts">
  import type { DashboardThemeProps } from "../../dashboard-view";
  import { formatDistance, formatDuration, formatDate } from "../../format";
  import RetryNotice from "../components/RetryNotice.svelte";

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
    onOpenActivity,
    onOpenSport,
    sleepSummary,
    mediaAggregates,
    theme,
    children,
  }: DashboardThemeProps = $props();

  // Same meteorological tagging as the field record, folded into a season
  // wheel over the loaded activity sweep -- a long view read as a turning
  // year rather than a flat total.
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
  type Season = (typeof SEASONS)[number];

  const seasonCounts = $derived.by(() => {
    const counts: Record<Season, number> = {
      spring: 0,
      summer: 0,
      autumn: 0,
      winter: 0,
    };
    for (const activity of activities) {
      const month = new Date(activity.started_at).getMonth();
      if (!Number.isNaN(month)) counts[SEASONS[month]] += 1;
    }
    return counts;
  });
  const seasonTotal = $derived(
    Object.values(seasonCounts).reduce((sum, value) => sum + value, 0),
  );
  const seasonWheel = $derived.by(() => {
    const order: Season[] = ["spring", "summer", "autumn", "winter"];
    let cursor = 0;
    const stops: string[] = [];
    for (const season of order) {
      const count = seasonCounts[season];
      const span = seasonTotal > 0 ? (count / seasonTotal) * 360 : 0;
      stops.push(
        `var(--season-${season}) ${cursor.toFixed(1)}deg ${(cursor + span).toFixed(1)}deg`,
      );
      cursor += span;
    }
    return stops.length ? stops.join(", ") : "var(--border) 0deg 360deg";
  });
</script>

<section
  class="bloom-view"
  data-theme={theme}
  aria-labelledby="bloom-view-title"
>
  <header class="view-opening">
    <div>
      <p class="bloom-kicker">◑ Long view · standing entry</p>
      <h1 id="bloom-view-title">The whole cycle, at once.</h1>
      <p>
        Distance, sessions, and seasons read together, without turning the
        record into a verdict.
      </p>
    </div>
    <div class="view-count">
      <strong>{streak}</strong>
      <span>streak</span>
    </div>
  </header>

  {#if loading}
    <p class="view-status">Gathering the long view…</p>
  {:else if error}
    <RetryNotice message={error} {onRetry} />
  {:else}
    <dl class="view-summary">
      <div>
        <dt>Distance</dt>
        <dd>{formatDistance(summary?.totals.distance_m)}</dd>
      </div>
      <div>
        <dt>Activities</dt>
        <dd>{summary?.totals.activity_count ?? "—"}</dd>
      </div>
      <div>
        <dt>Total time</dt>
        <dd>
          {formatDuration(
            summary?.totals.moving_time_s || summary?.totals.duration_s,
          )}
        </dd>
      </div>
      <div>
        <dt>Routes</dt>
        <dd>{routes?.features.length ?? "—"}</dd>
      </div>
      <div>
        <dt>Sleep ({sleepSummary.nightCount} nights)</dt>
        <dd>
          {sleepSummary.nightCount
            ? formatDuration(sleepSummary.averageAsleepS)
            : "—"}
        </dd>
      </div>
      <div>
        <dt>Media ({mediaAggregates?.totals.item_count ?? "—"} items)</dt>
        <dd>{mediaAggregates?.totals.completed_count ?? "—"}</dd>
      </div>
    </dl>

    <div class="view-grid">
      <section class="entries-card">
        <div class="card-heading">
          <div>
            <p class="bloom-kicker">◕ Movement notes</p>
            <h2>Recent entries</h2>
          </div>
          <span
            >{Math.min(8, activities.length)} of {activities.length} loaded</span
          >
        </div>
        {#if activities.length}
          <ol>
            {#each activities.slice(0, 8) as activity, index (activity.id)}
              <li>
                <span class="entry-mark"
                  >{String(index + 1).padStart(2, "0")}</span
                >
                <span class="entry-date">{formatDate(activity.started_at)}</span
                >
                <span class="entry-name"
                  >{activity.title || activity.sport_type}</span
                >
                <span
                  >{formatDistance(activity.distance_m)} · {formatDuration(
                    activity.duration_s ?? activity.moving_time_s,
                  )}</span
                >
              </li>
            {/each}
          </ol>
        {:else}
          <p class="bloom-empty">No movement sessions recorded yet.</p>
        {/if}
      </section>

      <div class="side-stack">
        <section class="wheel-card">
          <p class="bloom-kicker">Season wheel</p>
          <h2>{seasonTotal} sessions, by season</h2>
          <div
            class="season-wheel"
            style={`background: conic-gradient(${seasonWheel});`}
            role="img"
            aria-label={`Sessions by season: spring ${seasonCounts.spring}, summer ${seasonCounts.summer}, autumn ${seasonCounts.autumn}, winter ${seasonCounts.winter}`}
          >
            <span>{seasonTotal}</span>
          </div>
          <ul class="season-legend">
            <li><i class="dot-spring"></i>Spring · {seasonCounts.spring}</li>
            <li><i class="dot-summer"></i>Summer · {seasonCounts.summer}</li>
            <li><i class="dot-autumn"></i>Autumn · {seasonCounts.autumn}</li>
            <li><i class="dot-winter"></i>Winter · {seasonCounts.winter}</li>
          </ul>
        </section>

        <section class="map-card">
          <p class="bloom-kicker">Geography</p>
          <h2>
            {routes
              ? `${routes.features.length} route traces`
              : "Route footprint"}
          </h2>
          <div class="map-frame">
            {@render children?.()}
          </div>
        </section>
      </div>
    </div>
  {/if}

  <footer class="bloom-source">
    Source: activity summary, sweep, and routes · long-view presentation only
  </footer>
</section>

<style>
  .bloom-view {
    display: grid;
    gap: 1.5rem;
    font-family: var(--font-serif);
    --season-spring: var(--accent-2);
    --season-summer: var(--season-summer-accent);
    --season-autumn: var(--accent);
    --season-winter: var(--text-muted);
  }
  .bloom-kicker {
    margin: 0 0 0.5rem;
    color: var(--accent);
    font-size: 0.66rem;
    letter-spacing: 0.15em;
    text-transform: uppercase;
  }
  h1,
  h2 {
    margin: 0;
    font-weight: 400;
    letter-spacing: -0.02em;
  }
  h1 {
    max-width: 12ch;
    font-size: clamp(2.4rem, 6vw, 4.6rem);
    line-height: 0.95;
  }
  h2 {
    font-size: 1.4rem;
  }
  .view-opening {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
    border-bottom: 1px solid var(--border);
    padding-bottom: 1.75rem;
  }
  .view-opening p:last-child {
    max-width: 36rem;
    color: var(--text-muted);
    line-height: 1.65;
  }
  .view-count {
    display: grid;
    justify-items: end;
    color: var(--text-muted);
  }
  .view-count strong {
    color: var(--accent);
    font-style: italic;
    font-size: 3rem;
    font-weight: 400;
  }
  .view-count span {
    margin-top: 0.4rem;
    font-size: 0.68rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .view-status {
    border: 1px dashed var(--border);
    border-radius: var(--radius);
    padding: 2rem;
    color: var(--text-muted);
  }
  .view-summary {
    display: grid;
    grid-template-columns: repeat(6, 1fr);
    margin: 0;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface) 90%, transparent);
  }
  .view-summary div {
    display: grid;
    gap: 0.35rem;
    border-right: 1px solid var(--border);
    padding: 1.1rem;
  }
  .view-summary div:last-child {
    border-right: 0;
  }
  dt {
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  dd {
    margin: 0;
    font-style: italic;
    font-size: 1.35rem;
  }
  .view-grid {
    display: grid;
    grid-template-columns: 1.35fr 1fr;
    gap: 1.25rem;
  }
  .side-stack {
    display: grid;
    gap: 1.25rem;
  }
  .entries-card,
  .wheel-card,
  .map-card {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 1.4rem;
    background: color-mix(in srgb, var(--surface) 90%, transparent);
  }
  .card-heading {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
  }
  .card-heading > span {
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
    grid-template-columns: 1.6rem 5rem minmax(0, 1fr) auto;
    gap: 1rem;
    align-items: baseline;
    border-top: 1px solid var(--border);
    padding: 0.8rem 0;
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  .entry-mark {
    color: var(--accent);
    font-style: italic;
  }
  .entry-date {
    color: var(--accent);
  }
  .entry-name {
    color: var(--text);
    font-style: italic;
    font-size: 0.92rem;
  }
  .bloom-empty {
    color: var(--text-muted);
  }
  .wheel-card h2,
  .map-card h2 {
    margin-top: 0.7rem;
    font-size: 1.1rem;
  }
  .season-wheel {
    position: relative;
    display: grid;
    place-items: center;
    width: 8.5rem;
    height: 8.5rem;
    margin: 1.5rem auto 0;
    border-radius: 50%;
  }
  .season-wheel::before {
    content: "";
    position: absolute;
    inset: 22%;
    border-radius: 50%;
    background: var(--surface);
    box-shadow: 0 0 0 1px var(--border);
  }
  .season-wheel span {
    position: relative;
    color: var(--text);
    font-style: italic;
    font-size: 1.6rem;
  }
  .season-legend {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 0.5rem;
    margin: 1.5rem 0 0;
    padding: 0;
    list-style: none;
    color: var(--text-muted);
    font-size: 0.7rem;
  }
  .season-legend li {
    display: flex;
    align-items: center;
    gap: 0.4rem;
  }
  .season-legend i {
    display: inline-block;
    width: 0.55rem;
    height: 0.55rem;
    border-radius: 50%;
  }
  .dot-spring {
    background: var(--season-spring);
  }
  .dot-summer {
    background: var(--season-summer);
  }
  .dot-autumn {
    background: var(--season-autumn);
  }
  .dot-winter {
    background: var(--season-winter);
  }
  .map-frame {
    height: 11rem;
    margin-top: 1.2rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) * 0.6);
    overflow: hidden;
  }
  .map-frame :global(.map) {
    height: 100%;
  }
  .bloom-source {
    border-top: 1px solid var(--border);
    padding-top: 0.85rem;
    color: var(--text-muted);
    font-size: 0.68rem;
  }
  @media (max-width: 760px) {
    .view-opening,
    .view-grid {
      display: block;
    }
    .view-count {
      display: block;
      margin-top: 1.5rem;
    }
    .view-summary {
      grid-template-columns: repeat(2, 1fr);
    }
    .view-summary div:nth-child(even) {
      border-right: 0;
    }
    .view-summary div:not(:nth-last-child(-n + 2)) {
      border-bottom: 1px solid var(--border);
    }
    .entries-card {
      margin-bottom: 1.25rem;
    }
    .side-stack {
      margin-top: 1.25rem;
    }
    li {
      grid-template-columns: 1fr;
      gap: 0.25rem;
    }
  }
</style>
