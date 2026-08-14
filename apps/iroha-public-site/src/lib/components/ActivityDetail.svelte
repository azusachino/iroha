<script lang="ts">
  import {
    formatDate,
    formatDistance,
    formatDuration,
    formatElevation,
    formatHr,
    formatPace,
    formatSport,
  } from "$lib/format";
  import SportBadge from "@iroha/shared/SportBadge.svelte";
  import type { Activity } from "$lib/types";
  import type { RouteFeatureCollection } from "$lib/types";
  import RoutesMap from "$lib/components/RoutesMap.svelte";

  let {
    activity,
    routes,
    backHref,
  }: {
    activity: Activity;
    routes: RouteFeatureCollection;
    backHref: string;
  } = $props();

  const activityRoutes = $derived({
    type: "FeatureCollection" as const,
    features: routes.features.filter(
      (feature) => feature.properties.activity_id === activity.id,
    ),
  });
</script>

<section class="detail tile">
  <a class="back-link" href={backHref}>← Back to archive</a>
  <div class="detail-heading">
    <SportBadge sport={activity.sport_type} />
    <div>
      <p class="eyebrow">Public activity</p>
      <h2>{activity.title || formatSport(activity.sport_type)}</h2>
      <p class="muted">
        {formatDate(activity.started_at, activity.timezone)}
        {#if activity.timezone}
          · {activity.timezone}{/if}
      </p>
    </div>
  </div>

  <div class="detail-grid">
    <div>
      <span>Distance</span>
      <strong>{formatDistance(activity.distance_m)}</strong>
    </div>
    <div>
      <span>Duration</span>
      <strong>{formatDuration(activity.duration_s)}</strong>
    </div>
    <div>
      <span>Moving time</span>
      <strong>{formatDuration(activity.moving_time_s)}</strong>
    </div>
    <div>
      <span>Pace</span>
      <strong>{formatPace(activity.avg_pace_s_per_km)}</strong>
    </div>
    <div>
      <span>Elevation</span>
      <strong>{formatElevation(activity.elevation_gain_m)}</strong>
    </div>
    <div>
      <span>Heart rate</span>
      <strong>{formatHr(activity.avg_hr)}</strong>
    </div>
    <div>
      <span>Max heart rate</span>
      <strong>{formatHr(activity.max_hr)}</strong>
    </div>
  </div>

  {#if activityRoutes.features.length > 0}
    <section class="route-section">
      <div>
        <p class="eyebrow">Route</p>
        <h3>Activity trace</h3>
      </div>
      <div class="route-map tile">
        <RoutesMap data={activityRoutes} />
      </div>
    </section>
  {/if}

  <p class="detail-note muted">
    {#if activityRoutes.features.length > 0}
      This detail is rendered from the sanitized public snapshot. The route is
      included because the snapshot was exported with routes enabled.
    {:else}
      This public record has summary metrics only; no route trace was included
      in the snapshot.
    {/if}
  </p>
</section>

<style>
  .detail {
    padding: 1.25rem;
  }

  .back-link {
    display: inline-block;
    margin-bottom: 1.25rem;
    font-size: 0.85rem;
    font-weight: 700;
  }

  .detail-heading {
    display: flex;
    align-items: center;
    gap: 0.8rem;
    margin-bottom: 1.5rem;
  }

  .eyebrow {
    margin: 0 0 0.25rem;
    color: var(--accent);
    font-size: 0.7rem;
    font-weight: 700;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }

  h2 {
    margin: 0;
    font-size: clamp(1.6rem, 4vw, 2.4rem);
    letter-spacing: -0.03em;
  }

  .detail-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 0.75rem;
  }

  .detail-grid div {
    padding: 0.8rem;
    border: 1px solid var(--border);
    border-radius: 10px;
    background: var(--surface-2);
  }

  .detail-grid span,
  .detail-grid strong {
    display: block;
  }

  .detail-grid span {
    margin-bottom: 0.3rem;
    color: var(--text-muted);
    font-size: 0.72rem;
  }

  .detail-grid strong {
    font-size: 1rem;
  }

  .detail-note {
    margin: 1.25rem 0 0;
    font-size: 0.82rem;
  }

  .route-section {
    display: grid;
    gap: 0.75rem;
    margin-top: 1.5rem;
  }

  h3 {
    margin: 0;
    font-size: 1.2rem;
  }

  .route-map {
    height: 24rem;
    padding: 0.5rem;
  }

  @media (max-width: 768px) {
    .detail-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }
</style>
