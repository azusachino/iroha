<script lang="ts">
  import type { ActivityThemeProps } from "../../view-contracts/activity-view";
  import ActivityMetricChart from "../components/ActivityMetricChart.svelte";
  import RetryNotice from "../components/RetryNotice.svelte";
  import {
    formatDateOnly,
    formatDistance,
    formatDuration,
    formatPace,
  } from "../../format/format";
  import { sportLabel } from "../../domain/sport";
  import { sportIcon } from "../../domain/sport-icons";
  let recordsScroll = $state<HTMLDivElement>();
  let {
    activities,
    displaySummary,
    summaryLoading,
    summaryError,
    summaryDistanceUnknownCount,
    boundsError,
    boundsLoading,
    onRetryBounds,
    summaryScope,
    recordsScope,
    recordsReady,
    onRetrySummary,
    onRetryRecords,
    onRetrySeries,
    sportType,
    sportOptions,
    loading,
    error,
    hasMore,
    loadingMore,
    loadMoreError = null,
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

  let root = $state<HTMLElement>();
  let summaryPanel = $state<HTMLElement>();
  let recordsPanel = $state<HTMLElement>();

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

<section
  class="grapher-activities"
  aria-labelledby="activity-data-title"
  tabindex="-1"
  bind:this={root}
>
  <header class="page-intro">
    <p class="kicker">Activity data / public-style table</p>
    <h1 id="activity-data-title">The movement record.</h1>
    <p>
      Filter the imported sessions, then compare the same fields row by row.
    </p>
  </header>

  {@render children?.()}
  {#if boundsError}
    <RetryNotice
      message={`Could not load the available date range: ${boundsError}. Your selected period is unchanged.`}
      onRetry={onRetryBounds}
      retryLabel="Retry activity date range"
      focusTarget={root?.querySelector<HTMLElement>(
        '[aria-label="Filter by year"]',
      ) ?? undefined}
    />
  {:else if boundsLoading}
    <p class="muted" role="status">
      Loading the available activity date range…
    </p>
  {/if}

  <div class="filters" aria-label="Activity filters">
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
    onRetry={onRetrySeries}
    {theme}
  />

  <section
    class="read-state"
    aria-label="Activity summary read state"
    tabindex="-1"
    bind:this={summaryPanel}
  >
    {#if summaryScope}<p class="muted">Observed summary: {summaryScope}</p>{/if}
    {#if summaryError}
      <RetryNotice
        message={`Could not load activity summary: ${summaryError}`}
        onRetry={onRetrySummary}
        retryLabel="Retry activity summary"
        focusTarget={summaryPanel}
      />
    {:else if summaryLoading}
      <p class="muted" role="status">
        {displaySummary
          ? "Updating activity summary…"
          : "Loading activity summary…"}
      </p>
    {/if}
    <div
      class="summary-row"
      role="region"
      aria-label="Filtered activity summary"
    >
      <div>
        <span>Sessions</span><strong
          >{displaySummary
            ? displaySummary.activity_count.toLocaleString()
            : "—"}</strong
        >
      </div>
      <div>
        <span>Distance</span><strong
          >{displaySummary
            ? formatDistance(displaySummary.distance_m)
            : "—"}</strong
        >
      </div>
      <div>
        <span>Moving time</span><strong
          >{displaySummary
            ? formatDuration(displaySummary.duration_s)
            : "—"}</strong
        >
      </div>
    </div>

    {#if summaryDistanceUnknownCount > 0}
      <p class="muted">
        {summaryDistanceUnknownCount.toLocaleString()}
        {summaryDistanceUnknownCount === 1 ? "session has" : "sessions have"} no observed
        distance.
      </p>
    {/if}
  </section>

  <section
    class="read-state"
    aria-label="Activity records"
    tabindex="-1"
    bind:this={recordsPanel}
  >
    {#if recordsScope}<p class="muted">Observed records: {recordsScope}</p>{/if}
    {#if error}
      <RetryNotice
        message={`Could not load activity data: ${error}`}
        onRetry={onRetryRecords}
        retryLabel="Retry activities"
        focusTarget={recordsPanel}
      />
    {/if}
    {#if loading && !recordsReady}
      <p class="muted" role="status">Loading activity data…</p>
    {:else if recordsReady && activities.length === 0}
      <p class="muted">No activity sessions in this observed scope.</p>
    {:else if activities.length}
      <button
        class="scroll-hint"
        type="button"
        aria-controls="motion-records-scroll"
        onclick={() => recordsScroll?.focus({ preventScroll: true })}
        >Focus activity records to scroll all columns</button
      >
      <div
        class="table-frame"
        id="motion-records-scroll"
        role="region"
        aria-label="Motion exact activity records"
        tabindex="-1"
        bind:this={recordsScroll}
      >
        <table aria-label="Motion exact activity records">
          <thead
            ><tr
              ><th>Date</th><th>Activity</th><th>Distance</th><th>Duration</th
              ><th>Pace</th></tr
            ></thead
          >
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
                <td>{formatDateOnly(activity.started_at, activity.timezone)}</td
                >
                <td
                  ><a href={`/motion/${activity.id}`}
                    >{activity.title || sportLabel(activity.sport_type)}</a
                  ><small class="sport-small"
                    >{#if sportIcon(activity.sport_type)}
                      {@const Icon = sportIcon(activity.sport_type)}
                      <Icon size={11} aria-hidden="true" />
                    {/if}{sportLabel(activity.sport_type)}</small
                  ></td
                >
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
      {#if loadMoreError}
        <RetryNotice
          message={`Could not load the next activity page: ${loadMoreError}. Observed rows and cursor are retained.`}
          retryLabel="Retry more activities"
          onRetry={onLoadMore}
          focusTarget={recordsPanel}
        />
      {/if}
      {#if hasMore}<button
          class="load-more"
          onclick={onLoadMore}
          disabled={loadingMore}
          >{loadingMore ? "Loading…" : "Load more rows"}</button
        >{/if}
    {/if}
  </section>
</section>

<style>
  .grapher-activities {
    display: grid;
    gap: 1rem;
    min-width: 0;
  }
  .grapher-activities > *,
  .read-state > * {
    min-width: 0;
  }
  .read-state {
    display: grid;
    gap: 1rem;
    min-width: 0;
  }
  .page-intro {
    max-width: 48rem;
    padding-bottom: 2rem;
    border-bottom: 3px solid var(--text);
  }
  .kicker {
    margin: 0 0 0.45rem;
    color: var(--accent);
    font-size: 0.68rem;
    font-weight: 750;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  h1 {
    margin: 0;
    font-size: var(--grapher-utility-title-size);
    letter-spacing: -0.05em;
    line-height: 1;
  }
  .page-intro p:last-child {
    margin: 1rem 0 0;
    color: var(--text-muted);
  }
  .filters {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
    padding: 0.75rem 0;
    border-bottom: 1px solid var(--border);
  }
  .filters label {
    display: grid;
    gap: 0.35rem;
    color: var(--text-muted);
    font-size: 0.68rem;
    text-transform: uppercase;
  }
  .filters select {
    min-width: 9rem;
    border-radius: 0;
    font-size: 0.78rem;
  }
  .summary-row {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    border-block: 1px solid var(--border);
  }
  .summary-row div {
    display: grid;
    gap: 0.4rem;
    padding: 1rem;
    border-right: 1px solid var(--border);
  }
  .summary-row div:last-child {
    border: 0;
  }
  .summary-row span {
    color: var(--text-muted);
    font-size: 0.66rem;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }
  .summary-row strong {
    font-size: clamp(1.35rem, 3vw, 2.4rem);
    letter-spacing: -0.08em;
  }
  .scroll-hint {
    min-height: 44px;
  }
  .table-frame {
    overflow-x: auto;
    border-top: 2px solid var(--text);
  }
  table {
    width: 100%;
    min-width: 44rem;
    border-collapse: collapse;
    font-size: 0.78rem;
  }
  th,
  td {
    padding: 0.75rem 0.45rem;
    border-bottom: 1px solid var(--border);
    text-align: left;
    white-space: nowrap;
  }
  th {
    color: var(--text-muted);
    font-size: 0.64rem;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }
  td a {
    display: inline-flex;
    align-items: center;
    min-height: var(--control-target-min);
    color: var(--text);
    font-weight: 700;
    text-decoration: none;
  }
  td a:hover {
    color: var(--accent);
  }
  td small {
    display: block;
    margin-top: 0.2rem;
    color: var(--text-muted);
    font-size: 0.65rem;
  }
  .sport-small {
    display: flex;
    align-items: center;
    gap: 0.25rem;
  }
  .load-more {
    padding: 0.7rem 1rem;
    border: 1px solid var(--border);
    border-radius: 0;
    background: var(--surface);
    color: var(--text);
    cursor: pointer;
  }
  .muted {
    color: var(--text-muted);
  }
  @media (max-width: 640px) {
    .summary-row strong {
      font-size: 1.2rem;
    }
    .summary-row div {
      padding: 0.75rem 0.4rem;
    }
  }
</style>
