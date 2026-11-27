<script lang="ts">
  import type { Snippet } from "svelte";
  import type {
    Activity,
    Lap,
    RoutePoint,
    SamplingPoint,
  } from "../../domain/activity";
  import FocalBand, {
    type FocalPrimaryMetric,
    type FocalSecondaryItem,
  } from "../../components/FocalBand.svelte";
  import MetricTable from "../../components/MetricTable.svelte";
  import PanelFrame from "../../components/PanelFrame.svelte";
  import SourceBadge from "../../components/SourceBadge.svelte";
  import LapChart from "../components/LapChart.svelte";
  import {
    formatDate,
    formatDistance,
    formatDuration,
    formatHr,
    formatPace,
    formatSwimmingPace,
  } from "../../format/format";
  import { isSwimming, sportColor, sportLabel } from "../../domain/sport";

  let {
    activity,
    derivedDistanceM,
    route,
    samplings,
    laps,
    routeReady = true,
    samplesReady = true,
    lapsDerived = false,
    children,
  }: {
    activity: Activity;
    derivedDistanceM?: number;
    route: RoutePoint[];
    samplings: SamplingPoint[];
    laps: Lap[];
    routeReady?: boolean;
    samplesReady?: boolean;
    lapsDerived?: boolean;
    children?: Snippet;
  } = $props();

  const swimming = $derived(isSwimming(activity.sport_type));
  const distance = $derived(activity.distance_m ?? derivedDistanceM);
  const paceValue = $derived(
    swimming
      ? formatSwimmingPace(distance, activity.duration_s)
      : formatPace(activity.avg_pace_s_per_km),
  );

  const primaryMetric = $derived<FocalPrimaryMetric>(
    distance != null
      ? {
          label: "Distance",
          value: formatDistance(distance),
          detail:
            activity.duration_s != null
              ? `${formatDuration(activity.duration_s)} total duration`
              : undefined,
        }
      : {
          label: "Duration",
          value: formatDuration(activity.duration_s),
        },
  );

  const secondaryItems = $derived.by<FocalSecondaryItem[]>(() => {
    const items: FocalSecondaryItem[] = [];
    if (distance != null && activity.duration_s != null) {
      items.push({
        label: "Duration",
        value: formatDuration(activity.duration_s),
      });
    }
    if (paceValue !== "—") {
      items.push({
        label: swimming ? "Pace / 100m" : "Average pace",
        value: paceValue,
      });
    }
    if (activity.avg_hr != null) {
      items.push({
        label: "Average HR",
        value: formatHr(activity.avg_hr),
      });
    }
    if (
      items.length < 3 &&
      activity.elevation_gain_m != null &&
      activity.elevation_gain_m > 0
    ) {
      items.push({
        label: "Elevation",
        value: formatDistance(activity.elevation_gain_m),
      });
    }
    return items;
  });

  const emptyNote = $derived.by(() => {
    const missing: string[] = [];
    if (activity.avg_hr == null) missing.push("Average HR unavailable");
    if (activity.elevation_gain_m == null) missing.push("Elevation unavailable");
    return missing.length ? missing.join(" · ") : undefined;
  });
</script>

<article
  class="grapher-detail"
  style={`--activity-accent: ${sportColor(activity.sport_type)}`}
  aria-labelledby="grapher-detail-title"
>
  <header class="detail-header">
    <div>
      <p class="kicker">Motion / record detail</p>
      <h1 id="grapher-detail-title">
        {activity.title || sportLabel(activity.sport_type)}
      </h1>
      <p class="detail-meta">
        {formatDate(activity.started_at, activity.timezone)} · {sportLabel(
          activity.sport_type,
        )}
      </p>
    </div>
    <SourceBadge source={activity.source_kind} />
  </header>

  <FocalBand
    heading="Session summary"
    primary={primaryMetric}
    secondary={secondaryItems}
    empty={emptyNote}
  />

  {@render children?.()}

  {#if laps.length}
    <PanelFrame title="Lap chart">
      {#snippet meta()}
        <span>{lapsDerived ? "Route-derived intervals" : "Measured intervals"}</span>
      {/snippet}
      <LapChart {laps} {swimming} />
      <MetricTable
        ariaLabel="Activity laps"
        caption="Recorded lap splits"
        minWidth="28rem"
        nowrap
        numericAlign
        dense
      >
        <thead>
          <tr>
            <th>Lap</th>
            <th>Distance</th>
            <th>Duration</th>
            <th>Pace</th>
          </tr>
        </thead>
        <tbody>
          {#each laps as lap (lap.id)}
            <tr>
              <td>{lap.lap_no}</td>
              <td>{formatDistance(lap.distance_m)}</td>
              <td>{formatDuration(lap.duration_s)}</td>
              <td>{formatPace(lap.avg_pace_s_per_km)}</td>
            </tr>
          {/each}
        </tbody>
      </MetricTable>
    </PanelFrame>
  {/if}

  <div class="evidence-grid">
    <PanelFrame title="Exact record">
      {#snippet meta()}
        <span>Source measurements</span>
      {/snippet}
      <dl>
        <div>
          <dt>Samples</dt>
          <dd>{samplesReady ? samplings.length : "—"}</dd>
        </div>
        <div>
          <dt>Moving time</dt>
          <dd>{formatDuration(activity.moving_time_s)}</dd>
        </div>
        <div>
          <dt>Maximum HR</dt>
          <dd>{formatHr(activity.max_hr)}</dd>
        </div>
        <div>
          <dt>Route points</dt>
          <dd>{routeReady ? route.length : "—"}</dd>
        </div>
      </dl>
    </PanelFrame>
  </div>
</article>

<style>
  .grapher-detail {
    --accent: var(--activity-accent);
    display: grid;
    gap: var(--gap-section);
    min-width: 0;
  }

  .grapher-detail > * {
    min-width: 0;
  }

  h1,
  p {
    margin: 0;
  }

  h1 {
    max-width: 18ch;
    font-size: var(--grapher-utility-title-size, var(--type-display));
    letter-spacing: -0.03em;
    line-height: 1.1;
  }

  .kicker {
    margin-bottom: var(--space-1);
    color: var(--accent);
    font-size: var(--type-caption);
    font-weight: 700;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }

  .detail-header {
    display: flex;
    flex-wrap: wrap;
    justify-content: space-between;
    gap: var(--space-4);
    align-items: end;
    border-bottom: 1px solid var(--border);
    padding-bottom: var(--space-5);
  }

  .detail-meta {
    margin-top: var(--space-1);
    color: var(--text-muted);
    font-size: var(--type-label);
  }

  .evidence-grid {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: var(--gap-grid);
  }

  dt {
    color: var(--text-muted);
    font-size: var(--type-caption);
    font-weight: 700;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }

  dl {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--space-3);
    margin: 0;
  }

  dd {
    margin: var(--space-1) 0 0;
    font-size: var(--type-body);
    font-variant-numeric: tabular-nums;
  }
</style>
