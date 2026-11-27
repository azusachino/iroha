<script lang="ts">
  import type { TodayThemeProps } from "../../view-contracts/today-view";
  import type { ActivityRing } from "../../view-contracts/daily-view";
  import {
    formatDistance,
    formatDateOnly,
    formatDuration,
    formatHr,
    formatMetricValue,
    formatPace,
  } from "../../format/format";
  import { sportColor, sportLabel } from "../../domain/sport";
  import { mediaEventVerb } from "../../domain/media";
  import FocalBand, {
    type FocalSecondaryItem,
  } from "../../components/FocalBand.svelte";
  import MetricTable from "../../components/MetricTable.svelte";
  import PanelFrame from "../../components/PanelFrame.svelte";
  import RouteHeader from "../../components/RouteHeader.svelte";
  import RingGauge from "../components/RingGauge.svelte";
  import MediaUpdateList from "../components/MediaUpdateList.svelte";

  let {
    unavailableSections = [],
    dRow,
    mainNight,
    acts,
    mediaEvents,
    mediaUpdates,
    theme,
    controls,
    onOpenActivity,
    onOpenMedia,
  }: TodayThemeProps = $props();

  const rings = $derived<ActivityRing[]>(
    dRow?.ring
      ? [
          {
            label: "Move",
            value: dRow.ring.move_kcal,
            goal: dRow.ring.move_goal_kcal,
            unit: "kcal",
            color: "var(--ring-move)",
          },
          {
            label: "Exercise",
            value: dRow.ring.exercise_min,
            goal: dRow.ring.exercise_goal_min,
            unit: "min",
            color: "var(--ring-exercise)",
          },
          {
            label: "Stand",
            value: dRow.ring.stand_hours,
            goal: dRow.ring.stand_goal_hours,
            unit: "h",
            color: "var(--ring-stand)",
          },
        ]
      : [],
  );

  const hasMedia = $derived(mediaEvents.length > 0 || mediaUpdates.length > 0);

  const secondaryItems = $derived.by<FocalSecondaryItem[]>(() => {
    const items: FocalSecondaryItem[] = [];

    if (mainNight) {
      items.push({
        label: "Sleep",
        value: formatDuration(mainNight.asleep_s),
        detail: `${Math.round(mainNight.efficiency * 100)}% efficiency · ${formatDuration(mainNight.time_in_bed_s)} in bed`,
      });
    }

    if (dRow?.resting_hr != null) {
      items.push({
        label: "Resting HR",
        value: formatHr(dRow.resting_hr),
      });
    }

    if (hasMedia) {
      const totalMedia = mediaEvents.length + mediaUpdates.length;
      items.push({
        label: "Media",
        value: `${totalMedia} ${totalMedia === 1 ? "record" : "records"}`,
      });
    }

    return items;
  });

  const emptyNotes = $derived.by<string[]>(() => {
    const notes: string[] = [];

    if (!dRow?.ring) {
      notes.push(
        unavailableSections.includes("daily")
          ? "Activity rings unavailable for this day."
          : "No activity rings recorded for this day.",
      );
    }

    if (dRow?.steps == null) {
      notes.push(
        unavailableSections.includes("daily")
          ? "Steps unavailable for this day."
          : "No steps recorded for this day.",
      );
    }

    if (!mainNight) {
      notes.push(
        unavailableSections.includes("sleep")
          ? "Sleep session unavailable for this day."
          : "No sleep session recorded for this day.",
      );
    }

    if (!hasMedia) {
      notes.push(
        unavailableSections.includes("media")
          ? "Media sessions unavailable for this day."
          : "No exact media sessions recorded for this day.",
      );
    }

    return notes;
  });
</script>

<section class="grapher-today" data-theme={theme} aria-label="Today">
  <RouteHeader
    title="What did this day contain?"
    context="One imported day, rendered as comparable evidence rather than a score."
  />

  <FocalBand
    heading="Daily summary"
    {controls}
    secondary={secondaryItems}
    empty={emptyNotes.length ? emptyNotes.join(" · ") : undefined}
  >
    {#snippet primary()}
      <div class="focal-rings">
        {#if rings.length > 0}
          <RingGauge {rings} />
        {/if}
        {#if dRow?.steps != null}
          <div class="focal-steps">
            <div class="focal-steps-label">Steps</div>
            <div class="focal-steps-metric">
              <span class="focal-steps-value">
                {formatMetricValue(dRow.steps, "count")}
              </span>
              <span class="focal-steps-unit">steps</span>
            </div>
            {#if dRow.distance_km != null && dRow.distance_km > 0}
              <p class="focal-steps-detail">
                {formatDistance(dRow.distance_km * 1000)}
              </p>
            {/if}
          </div>
        {/if}
      </div>
    {/snippet}
  </FocalBand>

  <PanelFrame title="Sessions">
    {#snippet meta()}
      <span>
        {unavailableSections.includes("activities")
          ? "Sessions unavailable"
          : `${acts.length} ${acts.length === 1 ? "session" : "sessions"}`}
      </span>
    {/snippet}
    {#if acts.length}
      <MetricTable
        ariaLabel="Sessions"
        caption="Activity sessions recorded for this day"
        minWidth="32rem"
        nowrap
        dense
      >
        <thead>
          <tr>
            <th>Session</th>
            <th>Sport</th>
            <th>Distance</th>
            <th>Duration</th>
            <th>Pace</th>
            <th>Avg HR</th>
          </tr>
        </thead>
        <tbody>
          {#each acts as activity (activity.id)}
            <tr>
              <td>
                <button
                  type="button"
                  class="session-link"
                  style={`--sport: ${sportColor(activity.sport_type)}`}
                  onclick={() => onOpenActivity(activity.id)}
                >
                  {activity.title || sportLabel(activity.sport_type)}
                </button>
              </td>
              <td>{sportLabel(activity.sport_type)}</td>
              <td>{formatDistance(activity.distance_m)}</td>
              <td>
                {formatDuration(activity.duration_s ?? activity.moving_time_s)}
              </td>
              <td>{formatPace(activity.avg_pace_s_per_km)}</td>
              <td>{formatHr(activity.avg_hr)}</td>
            </tr>
          {/each}
        </tbody>
      </MetricTable>
    {:else}
      <p class="muted">
        {unavailableSections.includes("activities")
          ? "Activity sessions unavailable for this day."
          : "No activity sessions recorded for this day."}
      </p>
    {/if}
  </PanelFrame>

  {#if hasMedia}
    <PanelFrame title="Media record">
      {#if mediaEvents.length}
        <MetricTable
          ariaLabel="Media record"
          caption="Library changes fixed to this day"
          dense
        >
          <thead>
            <tr>
              <th>Title</th>
              <th>Evidence</th>
              <th>At</th>
            </tr>
          </thead>
          <tbody>
            {#each mediaEvents as event (event.id)}
              <tr>
                <td>
                  <button
                    class="media-link"
                    type="button"
                    onclick={() => onOpenMedia(event.media_id)}
                  >
                    {event.native_title || event.title}
                  </button>
                </td>
                <td>{mediaEventVerb(event)}</td>
                <td>{formatDateOnly(event.occurred_at)}</td>
              </tr>
            {/each}
          </tbody>
        </MetricTable>
      {:else}
        <p class="muted">No exact media sessions recorded for this day.</p>
      {/if}
      {#if mediaUpdates.length}
        <h3 class="media-updates-heading">Dated provider updates</h3>
        <MediaUpdateList updates={mediaUpdates} {onOpenMedia} />
      {/if}
    </PanelFrame>
  {/if}

  <footer class="source-note">
    This view is a presentation of imported facts. It does not calculate a
    readiness score.
  </footer>
</section>

<style>
  .grapher-today {
    display: grid;
    gap: var(--gap-section);
    min-width: 0;
  }

  .grapher-today > * {
    min-width: 0;
  }

  .focal-rings {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-4) var(--space-6);
    min-width: 0;
  }

  .focal-steps {
    display: grid;
    gap: var(--space-1);
    min-width: 0;
  }

  .focal-steps-label {
    color: var(--text-muted);
    font-size: var(--type-label);
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .focal-steps-metric {
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    gap: var(--space-2);
  }

  .focal-steps-value {
    color: var(--text);
    font-size: var(--type-hero);
    font-weight: 800;
    font-variant-numeric: tabular-nums;
    line-height: 1.05;
    letter-spacing: -0.02em;
  }

  .focal-steps-unit {
    color: var(--text-muted);
    font-size: var(--type-title-small);
    font-weight: 600;
  }

  .focal-steps-detail {
    margin: 0;
    color: var(--text-muted);
    font-size: var(--type-label);
    line-height: 1.4;
  }

  .session-link,
  .media-link {
    display: inline-flex;
    align-items: center;
    min-height: 24px;
    padding: 0;
    border: 0;
    background: transparent;
    color: var(--accent);
    font: inherit;
    font-weight: 700;
    text-align: left;
    cursor: pointer;
    border-radius: var(--radius-sm, 4px);
    transition: color var(--motion-micro);
  }

  .session-link {
    padding-left: var(--space-2);
    border-left: 4px solid var(--sport);
  }

  .session-link:hover,
  .media-link:hover {
    text-decoration: underline;
  }

  .session-link:focus-visible,
  .media-link:focus-visible {
    outline: 2px solid var(--color-focus);
    outline-offset: 3px;
  }

  .media-updates-heading {
    margin: var(--space-4) 0 var(--space-2);
    color: var(--text-muted);
    font-size: var(--type-caption);
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .muted {
    margin: 0;
    color: var(--text-muted);
    font-size: var(--type-label);
  }

  .source-note {
    padding: var(--space-3) 0;
    border-top: 1px solid var(--border);
    color: var(--text-muted);
    font-size: var(--type-caption);
  }
</style>
