<script lang="ts">
  import type { ActivityThemeProps } from "../../view-contracts/activity-view";
  import type { MetricSeriesResponse } from "../../components/metric-series";
  import EmptyNote from "../../components/EmptyNote.svelte";
  import FocalBand, {
    type FocalPrimaryMetric,
    type FocalSecondaryItem,
  } from "../../components/FocalBand.svelte";
  import MetricTable from "../../components/MetricTable.svelte";
  import PanelFrame from "../../components/PanelFrame.svelte";
  import RouteHeader from "../../components/RouteHeader.svelte";
  import ActivityMetricChart from "../components/ActivityMetricChart.svelte";
  import RetryNotice from "../components/RetryNotice.svelte";
  import {
    formatCanonicalMonth,
    formatDateOnly,
    formatDistance,
    formatDuration,
    formatPace,
  } from "../../format/format";
  import { sportLabel } from "../../domain/sport";
  import { sportIcon } from "../../domain/sport-icons";

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

  const MOVEMENT_CHART_MIN_POINTS = 3;

  let root = $state<HTMLElement>();
  let summaryPanel = $state<HTMLElement>();
  let recordsPanel = $state<HTMLElement>();

  function aggregatePeriodValue(
    response: MetricSeriesResponse | null | undefined,
    period: string,
  ): number | null {
    let total = 0;
    let observed = false;
    for (const item of response?.series ?? []) {
      const point = item.points.find((candidate) => candidate.period === period);
      if (point && "value" in point && point.value != null) {
        total += point.value;
        observed = true;
      }
    }
    return observed ? total : null;
  }

  function periodObservedDays(
    response: MetricSeriesResponse | null | undefined,
    period: string,
  ): number {
    return (response?.series ?? []).reduce(
      (max, item) =>
        Math.max(
          max,
          item.points.find((point) => point.period === period)?.observed_days ??
            0,
        ),
      0,
    );
  }

  function formatSeriesPeriod(period: string): string {
    if (/^\d{4}-\d{2}$/.test(period)) return formatCanonicalMonth(period);
    return period;
  }

  const movementPeriods = $derived(
    activitySeries?.series[0]?.points.map((point) => point.period) ?? [],
  );

  const expectedMovementPeriods = $derived(
    Math.max(
      0,
      ...(activitySeries?.series ?? []).map(
        (item) => item.coverage.expected_periods,
      ),
    ),
  );

  const movementRows = $derived(
    movementPeriods
      .map((period) => ({
        period,
        distance_m: aggregatePeriodValue(activitySeries, period),
        duration_s: aggregatePeriodValue(activityDurationSeries, period),
        observedDays: periodObservedDays(activitySeries, period),
      }))
      .toReversed(),
  );

  const observedMovementRows = $derived(
    movementRows.filter(
      (row) => row.distance_m != null || row.duration_s != null,
    ),
  );

  const useShortMovementSummary = $derived(
    !!activitySeries &&
      !activitySeriesError &&
      movementPeriods.length > 0 &&
      observedMovementRows.length > 0 &&
      observedMovementRows.length < MOVEMENT_CHART_MIN_POINTS,
  );

  const shortMovementSummary = $derived.by(() => {
    const grain = activitySeries?.period.grain ?? "period";
    const chronological = observedMovementRows.toReversed();
    if (chronological.length === 1) {
      const only = chronological[0];
      return `Single ${grain} observed: ${formatSeriesPeriod(only.period)} · ${formatDistance(only.distance_m)} · ${formatDuration(only.duration_s)}.`;
    }
    if (chronological.length === 2) {
      const [first, second] = chronological;
      return `Two ${grain}s observed: ${formatSeriesPeriod(first.period)} (${formatDistance(first.distance_m)}, ${formatDuration(first.duration_s)}) and ${formatSeriesPeriod(second.period)} (${formatDistance(second.distance_m)}, ${formatDuration(second.duration_s)}).`;
    }
    return "";
  });

  const primaryMetric = $derived.by<FocalPrimaryMetric>(() => {
    if (!displaySummary) {
      return {
        label: "Distance",
        value: "—",
        detail: summaryScope || undefined,
      };
    }
    if (displaySummary.distance_m != null) {
      return {
        label: "Distance",
        value: formatDistance(displaySummary.distance_m),
        detail: summaryScope || undefined,
      };
    }
    return {
      label: "Moving time",
      value: formatDuration(displaySummary.duration_s),
      detail: summaryScope || undefined,
    };
  });

  const secondaryItems = $derived.by<FocalSecondaryItem[]>(() => {
    const items: FocalSecondaryItem[] = [];
    if (displaySummary) {
      items.push({
        label: "Sessions",
        value: displaySummary.activity_count.toLocaleString(),
      });
      if (displaySummary.distance_m != null && displaySummary.duration_s > 0) {
        items.push({
          label: "Moving time",
          value: formatDuration(displaySummary.duration_s),
        });
      }
    }
    if (sportOptions.length > 0) {
      items.push({
        label: "Sports",
        value: sportOptions.length.toLocaleString(),
        detail: sportType ? `Filtered: ${sportLabel(sportType)}` : undefined,
      });
    }
    return items;
  });

  const emptyNote = $derived.by(() => {
    const notes: string[] = [];
    if (summaryDistanceUnknownCount > 0) {
      notes.push(
        `${summaryDistanceUnknownCount.toLocaleString()} ${summaryDistanceUnknownCount === 1 ? "session has" : "sessions have"} no observed distance.`,
      );
    }
    if (displaySummary && displaySummary.activity_count === 0) {
      notes.push("No activity sessions in this observed scope.");
    }
    return notes.length ? notes.join(" · ") : undefined;
  });

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
  aria-label="Motion"
  tabindex="-1"
  bind:this={root}
>
  <RouteHeader
    title="Motion"
    context="Movement sessions and activity routes"
  />

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
    <p class="status-note" role="status">
      Loading the available activity date range…
    </p>
  {/if}

  <div
    class="summary-region"
    aria-label="Activity summary read state"
    tabindex="-1"
    bind:this={summaryPanel}
  >
    {#if summaryError}
      <RetryNotice
        message={`Could not load activity summary: ${summaryError}`}
        onRetry={onRetrySummary}
        retryLabel="Retry activity summary"
        focusTarget={summaryPanel}
      />
    {:else if summaryLoading}
      <p class="status-note" role="status">
        {displaySummary
          ? "Updating activity summary…"
          : "Loading activity summary…"}
      </p>
    {/if}

    <FocalBand
      heading="Motion summary"
      controls={children}
      primary={primaryMetric}
      secondary={secondaryItems}
      empty={emptyNote}
    />
  </div>

  <div class="filters" aria-label="Activity filters">
    <label>
      <span>Sport</span>
      <select
        value={sportType}
        onchange={(event) =>
          onSportType((event.currentTarget as HTMLSelectElement).value)}
      >
        <option value="">All sports</option>
        {#each sportOptions as sport}
          <option value={sport}>{sportLabel(sport)}</option>
        {/each}
      </select>
    </label>
  </div>

  {#if useShortMovementSummary}
    <PanelFrame title="Movement over time">
      {#snippet meta()}
        <span>
          {observedMovementRows.length} / {expectedMovementPeriods} periods observed
        </span>
      {/snippet}

      {#if activitySeriesLoading}
        <p class="status-note" role="status">Updating movement series…</p>
      {/if}

      <EmptyNote text={shortMovementSummary} />

      <MetricTable
        ariaLabel="Exact movement series"
        scrollLabel="Exact movement series scroll area"
        caption="Exact movement series"
        minWidth="36rem"
        nowrap
        numericAlign
        dense
      >
        <thead>
          <tr>
            <th>Period</th>
            <th>Distance</th>
            <th>Duration</th>
            <th>Observed days</th>
          </tr>
        </thead>
        <tbody>
          {#each movementRows as row (row.period)}
            <tr>
              <td>{formatSeriesPeriod(row.period)}</td>
              <td>{formatDistance(row.distance_m)}</td>
              <td>{formatDuration(row.duration_s)}</td>
              <td>{row.observedDays || "—"}</td>
            </tr>
          {/each}
        </tbody>
      </MetricTable>
    </PanelFrame>
  {:else}
    <ActivityMetricChart
      series={activitySeries}
      durationSeries={activityDurationSeries}
      loading={activitySeriesLoading}
      error={activitySeriesError}
      scope={activitySeriesScope}
      onRetry={onRetrySeries}
      {theme}
    />
  {/if}

  <div
    class="read-state"
    aria-label="Activity records"
    tabindex="-1"
    bind:this={recordsPanel}
  >
    <PanelFrame title="Activity records">
      {#snippet meta()}
        {#if recordsScope}<span>{recordsScope}</span>{/if}
      {/snippet}

      {#if error}
        <RetryNotice
          message={`Could not load activity data: ${error}`}
          onRetry={onRetryRecords}
          retryLabel="Retry activities"
          focusTarget={recordsPanel}
        />
      {/if}

      {#if loading && !recordsReady}
        <p class="status-note" role="status">Loading activity data…</p>
      {:else if recordsReady && activities.length === 0}
        <EmptyNote text="No activity sessions in this observed scope." />
      {:else if activities.length}
        <MetricTable
          ariaLabel="Motion exact activity records"
          scrollLabel="Motion exact activity records"
          caption="Motion exact activity records"
          minWidth="44rem"
          nowrap
          dense
        >
          <thead>
            <tr>
              <th>Date</th>
              <th>Activity</th>
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
                <td>{formatDateOnly(activity.started_at, activity.timezone)}</td>
                <td>
                  <a href={`/motion/${activity.id}`}>
                    {activity.title || sportLabel(activity.sport_type)}
                  </a>
                  <small class="sport-small">
                    {#if sportIcon(activity.sport_type)}
                      {@const Icon = sportIcon(activity.sport_type)}
                      <Icon size={11} aria-hidden="true" />
                    {/if}
                    {sportLabel(activity.sport_type)}
                  </small>
                </td>
                <td>{formatDistance(activity.distance_m)}</td>
                <td>
                  {formatDuration(
                    activity.duration_s ?? activity.moving_time_s,
                  )}
                </td>
                <td>{formatPace(activity.avg_pace_s_per_km)}</td>
              </tr>
            {/each}
          </tbody>
        </MetricTable>

        {#if loadMoreError}
          <RetryNotice
            message={`Could not load the next activity page: ${loadMoreError}. Observed rows and cursor are retained.`}
            retryLabel="Retry more activities"
            onRetry={onLoadMore}
            focusTarget={recordsPanel}
          />
        {/if}

        {#if hasMore}
          <button
            class="load-more"
            type="button"
            onclick={onLoadMore}
            disabled={loadingMore}
          >
            {loadingMore ? "Loading…" : "Load more rows"}
          </button>
        {/if}
      {/if}
    </PanelFrame>
  </div>
</section>

<style>
  .grapher-activities {
    display: grid;
    gap: var(--gap-section);
    min-width: 0;
  }

  .grapher-activities > *,
  .summary-region > *,
  .read-state > * {
    min-width: 0;
  }

  .summary-region,
  .read-state {
    display: grid;
    gap: var(--space-3);
    min-width: 0;
  }

  .status-note {
    margin: 0;
    color: var(--text-muted);
    font-size: var(--type-label);
  }

  .filters {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-3);
    padding-bottom: var(--space-3);
    border-bottom: 1px solid var(--border);
  }

  .filters label {
    display: grid;
    gap: var(--space-1);
    color: var(--text-muted);
    font-size: var(--type-caption);
    font-weight: 700;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }

  .filters select {
    min-width: 9rem;
    min-height: var(--control-height);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    background: var(--surface);
    color: var(--text);
    padding: var(--space-1) var(--space-3);
    font-size: var(--type-label);
  }

  .filters select:focus-visible {
    outline: 2px solid var(--color-focus);
    outline-offset: 3px;
  }

  .activity-row {
    cursor: pointer;
    transition: background-color var(--motion-micro);
  }

  .activity-row:hover {
    background: color-mix(in srgb, var(--accent) 8%, var(--surface));
  }

  .activity-row:focus-visible {
    outline: 2px solid var(--color-focus);
    outline-offset: -2px;
  }

  .activity-row a {
    display: inline-flex;
    align-items: center;
    min-height: var(--control-target-min);
    color: var(--text);
    font-weight: 700;
    text-decoration: none;
  }

  .activity-row a:hover {
    color: var(--accent);
  }

  .activity-row a:focus-visible {
    outline: 2px solid var(--color-focus);
    outline-offset: 3px;
    border-radius: var(--radius-sm);
  }

  .sport-small {
    display: flex;
    align-items: center;
    gap: var(--space-1);
    margin-top: var(--space-1);
    color: var(--text-muted);
    font-size: var(--type-caption);
  }

  .load-more {
    justify-self: start;
    min-height: var(--control-height);
    padding: var(--space-2) var(--space-4);
    border: 1px solid var(--border);
    border-radius: var(--radius-pill);
    background: var(--surface);
    color: var(--text);
    font-size: var(--type-label);
    cursor: pointer;
  }

  .load-more:focus-visible {
    outline: 2px solid var(--color-focus);
    outline-offset: 3px;
  }
</style>
