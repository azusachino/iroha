<script lang="ts">
  import type { DashboardThemeProps } from "../../view-contracts/dashboard-view";
  import ActivityHeatmap from "../components/ActivityHeatmap.svelte";
  import FocalBand, {
    type FocalSecondaryItem,
  } from "../../components/FocalBand.svelte";
  import MetricTable from "../../components/MetricTable.svelte";
  import PanelFrame from "../../components/PanelFrame.svelte";
  import RouteHeader from "../../components/RouteHeader.svelte";
  import BarChart from "../components/BarChart.svelte";
  import {
    formatDate,
    formatDistance,
    formatDuration,
    formatElevation,
    formatHumanDuration,
    formatMetricValue,
  } from "../../format/format";
  import { formatMonth } from "../../format/month";
  import { buildMonthlyDistanceSeries } from "./monthly-distance";

  type Period = 6 | 12 | 24;
  const periods: Period[] = [6, 12, 24];

  let {
    summary,
    activeDays,
    heatmapEndDay,
    activities,
    routes,
    streak,
    loading,
    error,
    onRetry,
    onOpenActivity,
    onOpenSport,
    sleepSummary,
    sleepLoading,
    sleepError,
    mediaAggregates,
    mediaLoading,
    mediaError,
    theme,
    children,
  }: DashboardThemeProps = $props();

  let period = $state<Period>(12);

  const periodLabel = $derived(`Last ${period} months`);
  const monthly = $derived(
    buildMonthlyDistanceSeries(
      summary?.by_month ?? [],
      heatmapEndDay.slice(0, 7),
      period,
    ),
  );
  const activeMonths = $derived(
    monthly.filter((bucket) => bucket.distance_m > 0),
  );
  const periodBucketKeys = $derived(
    new Set(monthly.map((bucket) => bucket.key)),
  );
  const periodBuckets = $derived(
    (summary?.by_month ?? []).filter((bucket) =>
      periodBucketKeys.has(bucket.key),
    ),
  );
  const periodDistanceM = $derived(
    periodBuckets.reduce((sum, bucket) => sum + bucket.distance_m, 0),
  );
  const periodActivityCount = $derived(
    periodBuckets.reduce((sum, bucket) => sum + bucket.activity_count, 0),
  );
  const periodDurationS = $derived(
    periodBuckets.reduce(
      (sum, bucket) => sum + (bucket.moving_time_s || bucket.duration_s),
      0,
    ),
  );
  const primaryDistanceValue = $derived.by(() => {
    if (!summary) return "—";
    if (
      summary.totals.distance_unknown_count > 0 &&
      summary.totals.distance_known_count === 0
    ) {
      return "—";
    }
    return formatDistance(periodDistanceM);
  });

  const activeDayCount = $derived(
    activeDays.filter((day) => day.activity_count > 0).length,
  );

  const movementSecondary = $derived.by<{ value: string; detail: string } | null>(() => {
    if (!summary || summary.totals.activity_count <= 0) return null;
    if (!/^0\b/.test(streak.trim())) {
      return { value: streak, detail: "Active streak" };
    }
    if (activeDayCount > 0) {
      return {
        value:
          activeDayCount === 1
            ? "1 day"
            : `${formatMetricValue(activeDayCount, "count")} days`,
        detail: "Active days",
      };
    }
    if (summary.totals.elevation_gain_m > 0) {
      return {
        value: formatElevation(summary.totals.elevation_gain_m),
        detail: "Total elevation gain",
      };
    }
    if (summary.by_sport.length > 0) {
      return {
        value: formatMetricValue(summary.by_sport.length, "count"),
        detail: summary.by_sport.length === 1 ? "Activity type" : "Activity types",
      };
    }
    return {
      value: formatMetricValue(summary.totals.activity_count, "count"),
      detail: "All-time records",
    };
  });

  const secondaryItems = $derived.by<FocalSecondaryItem[]>(() => {
    if (!summary) return [];
    const items: FocalSecondaryItem[] = [];

    if (movementSecondary) {
      items.push({
        label: "Movement",
        value: movementSecondary.value,
        detail: movementSecondary.detail,
        href: "/motion",
        linkLabel: "Explore movement →",
      });
    }

    if (!sleepLoading && !sleepError && sleepSummary.nightCount > 0) {
      items.push({
        label: "Main sleep",
        value: formatHumanDuration(sleepSummary.averageAsleepS),
        detail: "Average per night",
        href: "/night",
        linkLabel: "Explore sleep →",
      });
    }

    if (
      !mediaLoading &&
      !mediaError &&
      (mediaAggregates?.totals.item_count ?? 0) > 0
    ) {
      items.push({
        label: "Library",
        value: formatMetricValue(
          mediaAggregates?.totals.item_count ?? 0,
          "count",
        ),
        detail: "Total items",
        href: "/library",
        linkLabel: "Explore library →",
      });
    }

    return items;
  });

  const emptyNotes = $derived.by<string[]>(() => {
    if (!summary) return [];
    const notes: string[] = [];

    if (summary.totals.activity_count === 0) {
      notes.push("Movement has no records");
    }
    if (!sleepLoading) {
      if (sleepError) {
        notes.push("Sleep unavailable");
      } else if (sleepSummary.nightCount === 0) {
        notes.push("Sleep has no records");
      }
    }
    if (!mediaLoading) {
      if (mediaError) {
        notes.push("Library unavailable");
      } else if ((mediaAggregates?.totals.item_count ?? 0) === 0) {
        notes.push("Library has no records");
      }
    }
    if (!routes || routes.features.length === 0) {
      notes.push("No routes recorded yet");
    }

    return notes;
  });

  const sports = $derived(
    (summary?.by_sport ?? [])
      .toSorted((left, right) => right.activity_count - left.activity_count)
      .slice(0, 5),
  );
  const comparison = $derived.by(() => {
    if (monthly.length < 2) return null;
    const current = monthly[monthly.length - 1];
    const previous = monthly[monthly.length - 2];
    return { difference: current.distance_m - previous.distance_m, previous };
  });
</script>

<section class="grapher-dashboard" data-theme={theme} aria-label="Overview">
  <RouteHeader title="Overview" context="Canonical movement, sleep, and library">
    {#snippet actions()}
      <a class="archive-link" href="/motion"
        >Open movement archive <span aria-hidden="true">→</span></a
      >
    {/snippet}
  </RouteHeader>

  {#if loading && !summary}
    <p class="status" role="status">Loading your long view…</p>
  {:else if error}
    <section class="status error" role="alert">
      <p>{error}</p>
      <button type="button" onclick={onRetry}>Retry overview</button>
    </section>
  {:else if !summary}
    <p class="status">No canonical overview is available yet.</p>
  {:else}
    {#if loading}
      <p class="updating" role="status">Updating overview…</p>
    {/if}

    <FocalBand
      heading={periodLabel}
      primary={{
        label: "Period distance",
        value: primaryDistanceValue,
        detail:
          periodActivityCount > 0
            ? `${formatMetricValue(periodActivityCount, "count")} activities · ${formatHumanDuration(periodDurationS)}`
            : "No recorded movement in this window",
      }}
      secondary={secondaryItems}
      empty={emptyNotes.length ? emptyNotes.join(" · ") : undefined}
    >
      {#snippet controls()}
        <div
          class="period-control"
          role="group"
          aria-label="Distance chart range"
        >
          {#each periods as months (months)}
            <button
              type="button"
              aria-label={`${months}M: Last ${months} months`}
              aria-pressed={period === months}
              class:selected={period === months}
              onclick={() => (period = months)}>{months}M</button
            >
          {/each}
        </div>
      {/snippet}
    </FocalBand>

    <div class="heatmap-panel">
      <PanelFrame title="Activity calendar">
        {#snippet meta()}
          <span>{formatMetricValue(activeDayCount, "count")} active days</span>
        {/snippet}
        <ActivityHeatmap
          days={activeDays}
          endDay={heatmapEndDay}
          title="Activity calendar"
          embedded
        />
      </PanelFrame>
    </div>

    <div class="dashboard-grid">
      <div class="chart-panel">
        <PanelFrame title="Monthly distance">
          {#snippet meta()}
            {#if comparison && activeMonths.length >= 2}
              <span class="comparison">
                {comparison.difference > 0
                  ? "▲"
                  : comparison.difference < 0
                    ? "▼"
                    : "→"}
                {formatDistance(Math.abs(comparison.difference))}
                vs {formatMonth(comparison.previous.key)}
              </span>
            {/if}
          {/snippet}
          {#if activeMonths.length >= 2}
            <BarChart
              categories={monthly.map((bucket) => formatMonth(bucket.key))}
              primary={{
                name: "Distance",
                values: monthly.map((bucket) => bucket.distance_m / 1000),
                color: "var(--accent)",
                axis: { unit: "km" },
                formatter: (value) => formatDistance(value * 1000),
              }}
              primaryType="line"
              height={270}
            />
          {:else if activeMonths.length === 1}
            <p class="empty-note">
              Single active month: {formatMonth(activeMonths[0].key)} with {formatDistance(
                activeMonths[0].distance_m,
              )}.
            </p>
          {:else}
            <p class="empty-note">No recorded distance in this window.</p>
          {/if}
        </PanelFrame>
      </div>

      <div class="sport-panel">
        <PanelFrame title="By activity">
          {#snippet meta()}
            <a class="panel-link" href="/motion"
              >All records <span aria-hidden="true">→</span></a
            >
          {/snippet}
          {#if sports.length}
            <ul class="sport-list">
              {#each sports as sport (sport.key)}
                <li>
                  <button
                    class="sport-row"
                    type="button"
                    aria-label={`${sport.key.replaceAll("_", " ")}: ${sport.activity_count} activity records`}
                    onclick={() => onOpenSport(sport.key)}
                  >
                    <span class="sport-name"
                      >{sport.key.replaceAll("_", " ")}</span
                    >
                    <span class="sport-count">
                      {formatMetricValue(sport.activity_count, "count")}
                    </span>
                  </button>
                </li>
              {/each}
            </ul>
          {:else}
            <p class="empty-note">No activity categories to compare yet.</p>
          {/if}
        </PanelFrame>
      </div>

      <div class="table-panel">
        <PanelFrame title="Recent movement">
          {#snippet meta()}
            <a class="panel-link" href="/motion"
              >Browse archive <span aria-hidden="true">→</span></a
            >
          {/snippet}
          {#if activities.length}
            <MetricTable
              ariaLabel="Recent movement"
              caption="Recent movement records with distance and duration"
              minWidth="34rem"
              nowrap
              dense
            >
              <thead>
                <tr>
                  <th>Date</th>
                  <th>Activity</th>
                  <th>Distance</th>
                  <th>Duration</th>
                </tr>
              </thead>
              <tbody>
                {#each activities.slice(0, 8) as activity (activity.id)}
                  <tr>
                    <td
                      >{formatDate(
                        activity.started_at,
                        activity.timezone,
                      )}</td
                    >
                    <td>
                      <button
                        class="activity-link"
                        type="button"
                        onclick={() => onOpenActivity(activity.id)}
                        >{activity.title || activity.sport_type}</button
                      >
                    </td>
                    <td>{formatDistance(activity.distance_m)}</td>
                    <td>
                      {formatDuration(
                        activity.duration_s ?? activity.moving_time_s,
                      )}
                    </td>
                  </tr>
                {/each}
              </tbody>
            </MetricTable>
          {:else}
            <p class="empty-note">No movement records to show yet.</p>
          {/if}
        </PanelFrame>
      </div>

      {#if routes && routes.features.length > 0}
        <div class="route-panel">
          <PanelFrame title="Route footprint">
            {#snippet meta()}
              <span>{routes.features.length} traces</span>
            {/snippet}
            {@render children?.()}
          </PanelFrame>
        </div>
      {/if}
    </div>
  {/if}

  <footer>Source: canonical activity summary.</footer>
</section>

<style>
  .grapher-dashboard {
    display: grid;
    gap: 1.1rem;
    min-width: 0;
  }

  .grapher-dashboard > * {
    min-width: 0;
  }

  .grapher-dashboard :global(.focal-primary-value),
  .grapher-dashboard :global(.focal-secondary-item dd > span:first-child) {
    white-space: nowrap;
  }

  p {
    margin: 0;
  }

  .archive-link,
  .panel-link {
    display: inline-flex;
    align-items: center;
    gap: 0.2rem;
    width: fit-content;
    min-height: var(--control-target-min);
    padding: 0 0.15rem;
    font-size: var(--type-label);
    font-weight: 650;
  }

  .archive-link:focus-visible,
  .panel-link:focus-visible,
  .period-control button:focus-visible,
  .sport-row:focus-visible,
  .activity-link:focus-visible,
  .status button:focus-visible {
    outline: 2px solid var(--color-focus);
    outline-offset: 3px;
  }

  .dashboard-grid {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    align-items: start;
    gap: var(--gap-grid);
  }

  .dashboard-grid > * {
    min-width: 0;
  }

  @media (min-width: 1024px) {
    .dashboard-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }

    .table-panel,
    .route-panel {
      grid-column: 1 / -1;
    }
  }

  .heatmap-panel {
    min-width: 0;
  }

  .empty-note,
  footer {
    color: var(--text-muted);
    font-size: var(--type-caption);
  }

  .comparison {
    color: var(--accent);
    font-size: var(--type-label);
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }

  .period-control {
    display: inline-flex;
    gap: 0.1rem;
    padding: 0.18rem;
    border: 1px solid var(--border);
    border-radius: 999px;
    background: var(--surface-2);
  }

  .period-control button {
    min-width: 2.45rem;
    min-height: var(--control-target-min);
    padding: 0.25rem 0.45rem;
    border: 0;
    border-radius: 999px;
    background: transparent;
    color: var(--text-muted);
    font-size: var(--type-caption);
    font-weight: 700;
    cursor: pointer;
    transition:
      background-color var(--motion-micro),
      color var(--motion-micro),
      transform var(--motion-micro);
  }

  .period-control button:hover {
    background: color-mix(in srgb, var(--accent) 14%, var(--surface-2));
    color: var(--text);
  }

  .period-control button:active {
    transform: scale(0.96);
  }

  .period-control button.selected {
    background: var(--accent);
    color: var(--bg);
  }

  .sport-list {
    display: grid;
    gap: 0.2rem;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .sport-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    align-items: center;
    gap: 0.35rem 0.7rem;
    width: 100%;
    min-height: var(--control-target-min);
    padding: 0.5rem 0.2rem;
    border: 0;
    border-bottom: 1px solid var(--border);
    background: transparent;
    color: var(--text);
    text-align: left;
    cursor: pointer;
  }

  .sport-row:hover .sport-name {
    color: var(--accent);
  }

  .sport-name {
    overflow: hidden;
    font-size: var(--type-label);
    font-weight: 650;
    text-overflow: ellipsis;
    text-transform: capitalize;
    white-space: nowrap;
  }

  .sport-count {
    color: var(--text-muted);
    font-size: var(--type-caption);
    font-variant-numeric: tabular-nums;
  }

  .activity-link {
    display: block;
    max-width: 16rem;
    min-width: var(--control-target-min);
    min-height: var(--control-target-min);
    overflow: hidden;
    padding: 0;
    border: 0;
    background: transparent;
    color: var(--text);
    font: inherit;
    font-weight: 700;
    line-height: var(--control-target-min);
    text-align: left;
    text-overflow: ellipsis;
    white-space: nowrap;
    cursor: pointer;
  }

  .activity-link:hover {
    color: var(--accent);
  }

  .route-panel :global(.map) {
    height: 20rem;
  }

  .status {
    display: grid;
    gap: 0.75rem;
    padding: 1.5rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text-muted);
  }

  .status button {
    width: fit-content;
    min-height: var(--control-target-min);
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 4px);
    background: var(--surface-2);
    color: var(--text);
    font: inherit;
    font-size: var(--type-caption);
    cursor: pointer;
  }

  .status.error {
    color: var(--danger);
  }

  .updating {
    color: var(--text-muted);
    font-size: var(--type-caption);
  }

  footer {
    border-top: 1px solid var(--border);
    padding-top: 0.75rem;
  }

  @media (max-width: 640px) {
    .dashboard-grid {
      gap: 0.75rem;
    }

    .route-panel :global(.map) {
      height: 16rem;
    }
  }
</style>
