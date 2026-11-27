<script lang="ts">
  import {
    formatDate,
    formatDistance,
    formatDuration,
    formatHr,
    formatPace,
  } from "$lib/format";
  import {
    formatCyclingSpeed,
    formatSwimmingPace,
    hasMeaningfulActivityTitle,
    isCycling,
    isNonDistanceSport,
    isSwimming,
    type PublicSortEntry,
    type PublicSortKey,
  } from "$lib/public/showcase-helpers";
  import type { Activity } from "$lib/public/types";
  import PanelFrame from "@iroha/shared/components/PanelFrame.svelte";
  import SportBadge from "@iroha/shared/components/SportBadge.svelte";

  const columns: { id: PublicSortKey; header: string }[] = [
    { id: "started_at", header: "Date" },
    { id: "sport_type", header: "Activity" },
    { id: "distance_m", header: "Distance / Duration" },
    { id: "avg_pace_s_per_km", header: "Pace / HR" },
  ];

  let {
    filteredCount,
    totalSortedCount,
    visibleActivities,
    sorting,
    hasMore,
    activityHref,
    onToggleSort,
    onLoadMore,
  }: {
    filteredCount: number;
    totalSortedCount: number;
    visibleActivities: Activity[];
    sorting: PublicSortEntry[];
    hasMore: boolean;
    activityHref: (id: string) => string;
    onToggleSort: (id: PublicSortKey, event: MouseEvent) => void;
    onLoadMore: () => void;
  } = $props();

  let recordsScroll = $state<HTMLDivElement>();
</script>

<section class="section-heading">
  <h2>Activities</h2>
</section>

{#if filteredCount === 0}
  <p class="muted">No activities for this selection.</p>
{:else}
  <PanelFrame label="Activity records">
    <p class="muted small">
      All record columns are preserved. Swipe horizontally, or focus the table
      region and use arrow keys, to reach later columns.
    </p>
    <button
      class="scroll-hint"
      type="button"
      aria-controls="public-records-scroll"
      onclick={() => recordsScroll?.focus({ preventScroll: true })}
    >
      Focus public records to scroll all columns
    </button>
    <div
      class="table-wrap"
      id="public-records-scroll"
      bind:this={recordsScroll}
      role="region"
      aria-label="Public activity records — horizontally scrollable"
      tabindex="-1"
    >
      <table aria-label="Public activity records">
        <thead>
          <tr>
            {#each columns as column (column.id)}
              {@const activeSort = sorting.find(
                (sort) => sort.id === column.id,
              )}
              <th
                aria-sort={sorting[0]?.id === column.id
                  ? sorting[0].desc
                    ? "descending"
                    : "ascending"
                  : "none"}
              >
                <button
                  type="button"
                  class="sort-header"
                  onclick={(event) => onToggleSort(column.id, event)}
                >
                  {column.header}
                  {#if activeSort}{activeSort.desc ? "▼" : "▲"}{/if}
                </button>
              </th>
            {/each}
          </tr>
        </thead>
        <tbody>
          {#each visibleActivities as activity (activity.id)}
            <tr>
              <td class="nowrap">
                <a class="activity-link" href={activityHref(activity.id)}>
                  {formatDate(activity.started_at, activity.timezone)}
                </a>
              </td>
              <td>
                <div class="activity-cell">
                  <SportBadge sport={activity.sport_type} />
                  {#if hasMeaningfulActivityTitle(activity)}
                    <span class="activity-title">{activity.title}</span>
                  {/if}
                </div>
              </td>
              <td>
                {#if isNonDistanceSport(activity.sport_type, activity.distance_m)}
                  {formatDuration(
                    activity.duration_s ?? activity.moving_time_s,
                  )}
                {:else}
                  {formatDistance(activity.distance_m)}
                {/if}
              </td>
              <td>
                {#if isNonDistanceSport(activity.sport_type, activity.distance_m)}
                  {#if activity.avg_hr}Avg HR: {formatHr(
                      activity.avg_hr,
                    )}{:else if activity.max_hr}Max HR: {formatHr(
                      activity.max_hr,
                    )}{:else}—{/if}
                {:else if isCycling(activity.sport_type)}
                  {formatCyclingSpeed(
                    activity.distance_m,
                    activity.duration_s ?? activity.moving_time_s,
                  )}
                {:else if isSwimming(activity.sport_type)}
                  {formatSwimmingPace(
                    activity.distance_m,
                    activity.duration_s ?? activity.moving_time_s,
                  )}
                {:else}
                  {formatPace(activity.avg_pace_s_per_km)}
                  {#if activity.avg_hr}· {formatHr(activity.avg_hr)}{/if}
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
    {#if hasMore}
      <div class="load-more">
        <button type="button" onclick={onLoadMore}>Load more</button>
        <span class="muted small">
          Showing {visibleActivities.length} of {totalSortedCount}
        </span>
      </div>
    {/if}
  </PanelFrame>
{/if}

<style>
  .small {
    font-size: var(--type-label);
  }
  .section-heading {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 1rem;
    margin: 1.5rem 0 0.75rem;
  }
  .section-heading h2 {
    margin: 0;
    font-size: var(--type-title);
    letter-spacing: -0.02em;
  }
  .scroll-hint {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border: 0;
  }
  .scroll-hint:focus,
  .scroll-hint:focus-visible {
    position: static;
    width: auto;
    height: auto;
    min-height: 44px;
    padding: 0.5rem 1rem;
    margin: 0.5rem 0;
    overflow: visible;
    clip: auto;
    white-space: normal;
    background: var(--surface, #1e222b);
    color: var(--text, #fff);
    border: 1px solid var(--border, #333);
    border-radius: 6px;
    font-size: var(--type-label);
    font-weight: 500;
    cursor: pointer;
    display: inline-block;
    outline: 2px solid var(--color-focus, var(--accent));
    outline-offset: 2px;
  }
  .table-wrap {
    overflow-x: auto;
    padding: 0.25rem;
  }
  table {
    width: 100%;
    border-collapse: collapse;
    font-size: var(--type-label);
  }
  th,
  td {
    padding: 0.6rem 0.75rem;
    border-bottom: 1px solid var(--border);
    text-align: left;
  }
  tbody tr {
    transition: background 0.15s ease;
  }
  tbody tr:hover {
    background: color-mix(in srgb, var(--accent) 7%, transparent);
  }
  .sort-header {
    min-height: 44px;
    min-width: 44px;
    border: none;
    background: none;
    color: var(--text-muted);
    font: inherit;
    font-weight: 700;
    cursor: pointer;
  }
  .nowrap {
    white-space: nowrap;
  }
  .activity-cell {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
  .activity-link {
    color: var(--accent);
    font-weight: 700;
  }
  .activity-title {
    font-weight: 600;
  }
  .load-more {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 1rem;
    padding: 0.75rem;
  }
  .load-more button {
    padding: 0.35rem 0.9rem;
    border: 1px solid var(--border);
    border-radius: 999px;
    background: transparent;
    color: var(--text);
    cursor: pointer;
  }
</style>
