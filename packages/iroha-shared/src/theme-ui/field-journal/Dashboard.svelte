<script lang="ts">
  import type { DashboardThemeProps } from "../../view-contracts/dashboard-view";
  import { formatDistance, formatDuration, formatDate } from "../../format/format";
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
</script>

<section
  class="journal-long-view"
  data-theme={theme}
  aria-labelledby="journal-long-view-title"
>
  <header class="view-opening">
    <div>
      <p class="journal-kicker">Long view · standing entry</p>
      <h1 id="journal-long-view-title">The days, kept in view.</h1>
      <p>
        Distance, sessions, and routes read together as a continuing record,
        without turning the day into a verdict.
      </p>
    </div>
    <div class="view-stamp" aria-label="Current streak">
      <strong>{streak}</strong>
      <span>streak</span>
    </div>
  </header>

  <div class="journal-rule"><span>the standing totals</span></div>

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
            <p class="journal-kicker">03 · movement notes</p>
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
                <span class="entry-index"
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
          <p class="journal-empty">No movement sessions recorded yet.</p>
        {/if}
      </section>

      <section class="map-card">
        <p class="journal-kicker">Geography</p>
        <h2>
          {routes
            ? `${routes.features.length} route traces`
            : "Route footprint"}
        </h2>
        <div class="map-frame">
          {@render children?.()}
        </div>
        <p>
          Routes stay linked to their source activity and remain inspectable
          from the journal.
        </p>
      </section>
    </div>
  {/if}

  <footer class="journal-source">
    Source: activity summary and routes · long-view presentation only
  </footer>
</section>

<style>
  .journal-long-view {
    display: grid;
    gap: 1.5rem;
    min-width: 0;
  }
  .journal-long-view > * {
    min-width: 0;
  }
  .journal-kicker {
    margin: 0 0 0.55rem;
    color: var(--accent);
    font-size: 0.68rem;
    letter-spacing: 0.16em;
    text-transform: uppercase;
  }
  h1,
  h2 {
    margin: 0;
    font-family: var(--font-serif);
    font-weight: 400;
    letter-spacing: -0.04em;
  }
  h1 {
    max-width: 11ch;
    font-size: clamp(2.6rem, 6vw, 4.8rem);
    line-height: 0.92;
  }
  h2 {
    margin: 0.25rem 0 0.5rem;
    font-size: 1.5rem;
  }
  .view-opening {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    align-items: end;
  }
  .view-opening p:last-child {
    max-width: 38rem;
    color: var(--text-muted);
    line-height: 1.7;
  }
  .view-stamp {
    display: grid;
    justify-items: end;
    color: var(--text-muted);
  }
  .view-stamp strong {
    color: var(--accent);
    font-family: var(--font-serif);
    font-size: 3.2rem;
    font-weight: 400;
    line-height: 0.85;
  }
  .view-stamp span {
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
  .view-status {
    border: 1px dashed var(--border);
    padding: 2rem;
    color: var(--text-muted);
  }
  .view-summary {
    display: grid;
    grid-template-columns: repeat(6, 1fr);
    margin: 0;
    border: 1px solid var(--border);
    background: color-mix(in srgb, var(--surface-1) 84%, transparent);
  }
  .view-summary div {
    display: grid;
    gap: 0.4rem;
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
    font-family: var(--font-serif);
    font-size: 1.4rem;
  }
  .view-grid {
    display: grid;
    grid-template-columns: 1.4fr 1fr;
    gap: 1.25rem;
  }
  .view-grid > * {
    min-width: 0;
  }
  .entries-card,
  .map-card {
    border: 1px solid var(--border);
    padding: 1.5rem;
    background: color-mix(in srgb, var(--surface-1) 84%, transparent);
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
  .entries-card ol {
    margin: 1rem 0 0;
    padding: 0;
    list-style: none;
  }
  .entries-card li {
    display: grid;
    grid-template-columns: 1.6rem 5rem minmax(0, 1fr) auto;
    gap: 1rem;
    align-items: baseline;
    border-top: 1px solid var(--border);
    padding: 0.8rem 0;
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  .entry-index {
    color: var(--accent);
    font-family: var(--font-serif);
  }
  .entry-date {
    color: var(--accent);
  }
  .entry-name {
    color: var(--text);
    font-family: var(--font-serif);
    font-size: 0.92rem;
  }
  .journal-empty {
    color: var(--text-muted);
  }
  .map-card h2 {
    margin-top: 1rem;
  }
  .map-frame {
    height: 12rem;
    margin: 1.25rem 0;
    border: 1px solid var(--border);
  }
  .map-frame :global(.map) {
    height: 100%;
  }
  .map-card > p:last-child {
    color: var(--text-muted);
  }
  .journal-source {
    border-top: 1px solid var(--border);
    padding-top: 0.8rem;
    color: var(--text-muted);
    font-size: 0.68rem;
  }
  @media (max-width: 768px) {
    .view-opening,
    .view-grid {
      display: block;
    }
    .view-stamp {
      align-items: start;
      justify-items: start;
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
    .entries-card li {
      grid-template-columns: 1fr;
      gap: 0.25rem;
    }
  }
</style>
