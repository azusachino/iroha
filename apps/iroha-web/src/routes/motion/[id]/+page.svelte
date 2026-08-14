<script lang="ts">
  import { page } from "$app/state";
  import {
    getActivity,
    getActivityRoute,
    getActivitySamplings,
    getActivityLaps,
    type Activity,
    type RoutePoint,
    type SamplingPoint,
    type Lap,
  } from "$lib/api";
  import {
    deriveRouteDistanceM,
    populateRouteDistances,
  } from "$lib/activity-metrics";
  import {
    formatDistance,
    formatDuration,
    formatPace,
    formatSwimmingPace,
    formatElevation,
    formatHr,
    formatDate,
  } from "$lib/format";
  import RouteMap from "$lib/components/RouteMap.svelte";
  import FusedActivityChart from "$lib/components/FusedActivityChart.svelte";
  import ActivityDetailChart from "$lib/components/ActivityDetailChart.svelte";
  import LapChart from "@iroha/shared/theme-ui/components/LapChart.svelte";
  import SportBadge from "$lib/components/SportBadge.svelte";
  import StatTile from "$lib/components/StatTile.svelte";
  import RouteIntro from "$lib/components/RouteIntro.svelte";
  import LoadingBoundary from "$lib/components/LoadingBoundary.svelte";
  import { isSwimming, sportLabel } from "$lib/sport";
  import { useTheme } from "$lib/themes/context.svelte";
  import ThemeRouteRenderer from "$lib/themes/ThemeRouteRenderer.svelte";
  import { hasThemeRoute } from "$lib/themes/registry";

  function displayTitle(title?: string, sport?: string): string {
    if (!title) return sportLabel(sport);
    if (sportLabel(title) === sportLabel(sport)) return sportLabel(sport);
    return title;
  }

  let activity = $state<Activity | null>(null);
  let route = $state<RoutePoint[]>([]);
  let samplings = $state<SamplingPoint[]>([]);
  let laps = $state<Lap[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);
  let selectedRouteIndex = $state<number | null>(null);
  const theme = useTheme();

  const id = $derived(page.params.id ?? "");

  $effect(() => {
    const activityId = id;
    if (!activityId) return;
    loading = true;
    error = null;
    Promise.all([
      getActivity(activityId),
      // Sub-resources are best-effort; an empty/failed one should not blank the page.
      getActivityRoute(activityId)
        .then((r) => r ?? [])
        .catch(() => [] as RoutePoint[]),
      // The charts only use heart_rate; skip the larger power/energy/speed streams.
      getActivitySamplings(activityId, ["heart_rate"])
        .then((s) => s ?? [])
        .catch(() => [] as SamplingPoint[]),
      getActivityLaps(activityId)
        .then((l) => l ?? [])
        .catch(() => [] as Lap[]),
    ])
      .then(([a, r, s, l]) => {
        activity = a;
        route = r;
        samplings = s;
        laps = l;
      })
      .catch((e) => {
        error = e instanceof Error ? e.message : String(e);
      })
      .finally(() => {
        loading = false;
      });
  });

  function populateSpeed(points: RoutePoint[]): RoutePoint[] {
    for (let i = 1; i < points.length; i++) {
      const p1 = points[i - 1];
      const p2 = points[i];
      if (
        p2.speed_mps == null &&
        p1.ts &&
        p2.ts &&
        p1.distance_m != null &&
        p2.distance_m != null
      ) {
        const timeDiff =
          (new Date(p2.ts).getTime() - new Date(p1.ts).getTime()) / 1000;
        if (timeDiff > 0) {
          const distDiff = p2.distance_m - p1.distance_m;
          p2.speed_mps = distDiff / timeDiff;
        }
      }
    }
    return points;
  }

  function associateHeartRates(
    points: RoutePoint[],
    samplings: SamplingPoint[],
  ): RoutePoint[] {
    const hrs = samplings.filter((s) =>
      /heart|(^|_)hr($|_)/i.test(s.sampling_type),
    );
    if (hrs.length === 0) return points;

    const sortedHrs = [...hrs].sort(
      (a, b) => new Date(a.ts).getTime() - new Date(b.ts).getTime(),
    );

    return points.map((p) => {
      if (p.heart_rate != null || !p.ts) return p;
      const pTime = new Date(p.ts).getTime();

      let closestHr = sortedHrs[0];
      let minDist = Math.abs(new Date(closestHr.ts).getTime() - pTime);

      for (const hr of sortedHrs) {
        const dist = Math.abs(new Date(hr.ts).getTime() - pTime);
        if (dist < minDist) {
          minDist = dist;
          closestHr = hr;
        } else if (dist > minDist) {
          break;
        }
      }

      if (minDist <= 15000) {
        return { ...p, heart_rate: closestHr.value };
      }
      return p;
    });
  }

  const processedRoute = $derived.by<RoutePoint[]>(() => {
    if (!route || route.length === 0) return [];
    let pts = populateRouteDistances(route);
    pts = populateSpeed(pts);
    pts = associateHeartRates(pts, samplings);
    return pts;
  });

  const derivedDistanceM = $derived(
    activity?.distance_m == null ? deriveRouteDistanceM(route) : undefined,
  );
  const displayDistanceM = $derived(
    activity?.distance_m ??
      (isSwimming(activity?.sport_type) ? derivedDistanceM : undefined),
  );

  // Choose an x-axis shared by all route-derived charts: distance if every
  // point has it, else elapsed time, else the raw sequence number.
  interface XAxis {
    values: number[];
    label: string;
  }
  const xAxis = $derived.by<XAxis>(() => {
    if (processedRoute.length === 0) return { values: [], label: "Point" };
    if (processedRoute.every((p) => p.distance_m != null)) {
      return {
        values: processedRoute.map((p) => (p.distance_m as number) / 1000),
        label: "Distance (km)",
      };
    }
    if (processedRoute.every((p) => p.ts != null)) {
      const t0 = new Date(processedRoute[0].ts as string).getTime();
      return {
        values: processedRoute.map(
          (p) => (new Date(p.ts as string).getTime() - t0) / 60000,
        ),
        label: "Time (min)",
      };
    }
    return { values: processedRoute.map((p) => p.seq), label: "Point" };
  });

  function column(
    get: (p: RoutePoint) => number | null | undefined,
  ): (number | null)[] {
    return processedRoute.map((p) => {
      const v = get(p);
      return v == null || !Number.isFinite(v) ? null : v;
    });
  }

  function hasData(values: (number | null)[]): boolean {
    return values.some((v) => v != null);
  }

  // Pace derived from speed (s/km = 1000 / m/s); guard against zero/idle speed.
  const paceSeries = $derived.by<(number | null)[]>(() =>
    processedRoute.map((p) => {
      if (
        p.speed_mps == null ||
        !Number.isFinite(p.speed_mps) ||
        p.speed_mps <= 0
      )
        return null;
      return (isSwimming(activity?.sport_type) ? 100 : 1000) / p.speed_mps;
    }),
  );
  const hrSeries = $derived.by(() => column((p) => p.heart_rate));
  const elevationSeries = $derived.by(() => column((p) => p.elevation_m));

  // Apple exports carry no activity-level elevation gain, but route points do
  // have elevation — estimate cumulative climb from them (3 m hysteresis to
  // damp GPS noise) when the stored value is absent.
  const elevationGainM = $derived.by<number | undefined>(() => {
    if (activity?.elevation_gain_m != null) return activity.elevation_gain_m;
    if (isSwimming(activity?.sport_type)) return undefined;
    const elevs = processedRoute
      .map((p) => p.elevation_m)
      .filter((e): e is number => e != null && Number.isFinite(e));
    if (elevs.length < 2) return undefined;
    const threshold = 3;
    let gain = 0;
    let ref = elevs[0];
    for (const e of elevs) {
      if (e - ref >= threshold) {
        gain += e - ref;
        ref = e;
      } else if (e < ref) {
        ref = e;
      }
    }
    return gain;
  });

  // Fallback heart-rate series from samplings when route points carry no HR.
  interface SamplingChart {
    x: number[];
    values: (number | null)[];
  }
  const hrSamplingChart = $derived.by<SamplingChart | null>(() => {
    if (hasData(hrSeries)) return null;
    const hr = samplings.filter((s) =>
      /heart|(^|_)hr($|_)/i.test(s.sampling_type),
    );
    if (hr.length === 0) return null;
    const t0 = new Date(hr[0].ts).getTime();
    return {
      x: hr.map((s) => (new Date(s.ts).getTime() - t0) / 60000),
      values: hr.map((s) => (Number.isFinite(s.value) ? s.value : null)),
    };
  });

  // Theme detail pages receive the same canonical series as the fallback
  // page. Prefer route-enriched measurements so pace, elevation, and heart
  // rate share one axis; fall back to the sampled heart-rate stream when a
  // source has no route geometry.
  interface DetailChart {
    xValues: number[];
    xLabel: string;
    pace: (number | null)[];
    heartRate: (number | null)[];
    elevation: (number | null)[];
  }
  const detailChart = $derived.by<DetailChart | null>(() => {
    if (hasData(paceSeries) || hasData(hrSeries) || hasData(elevationSeries)) {
      return {
        xValues: xAxis.values,
        xLabel: xAxis.label,
        pace: paceSeries,
        heartRate: hrSeries,
        elevation: elevationSeries,
      };
    }
    if (hrSamplingChart) {
      return {
        xValues: hrSamplingChart.x,
        xLabel: "Time (min)",
        pace: Array.from({ length: hrSamplingChart.values.length }, () => null),
        heartRate: hrSamplingChart.values,
        elevation: Array.from(
          { length: hrSamplingChart.values.length },
          () => null,
        ),
      };
    }
    return null;
  });

  const hasRouteLine = $derived(
    processedRoute.filter(
      (p) => Number.isFinite(p.lat) && Number.isFinite(p.lon),
    ).length >= 2,
  );
  const anyChart = $derived(!!detailChart);

  const hrZones = $derived.by(() => {
    const values = (detailChart?.heartRate ?? []).filter(
      (value): value is number => value != null && value > 0,
    );
    if (!values.length) return [];
    const zones = [
      { label: "Easy", min: 0, max: 0.7, color: "#3ecf8e" },
      { label: "Steady", min: 0.7, max: 0.8, color: "#f5c451" },
      { label: "Tempo", min: 0.8, max: 0.9, color: "#ff9f43" },
      { label: "Hard", min: 0.9, max: 2, color: "#ff6b6b" },
    ];
    const maxHr = Math.max(...values);
    return zones
      .map((zone) => ({
        ...zone,
        count: values.filter(
          (value) => value <= maxHr * zone.max && value > maxHr * zone.min,
        ).length,
      }))
      .filter((zone) => zone.count > 0);
  });

  const supportsSplits = $derived(
    /run|walk|hike|ride|cycl|swim/.test(
      activity?.sport_type?.toLowerCase() ?? "",
    ),
  );

  interface CalculatedLap {
    id: string;
    lap_no: number;
    distance_m: number;
    duration_s: number;
    avg_hr?: number;
    avg_pace_s_per_km?: number;
  }

  function calculateLapsFromRoute(
    routePoints: RoutePoint[],
    sportType: string,
  ): CalculatedLap[] {
    if (!/run|walk|hike|ride|cycl|swim/.test(sportType.toLowerCase()))
      return [];
    if (routePoints.length < 2) return [];

    // Filter and sort points that have distance and timestamp
    const points = routePoints
      .filter((p) => p.distance_m != null && p.ts != null)
      .sort((a, b) => (a.distance_m as number) - (b.distance_m as number));

    if (points.length < 2) return [];

    const calculatedLaps: CalculatedLap[] = [];
    let startIdx = 0;
    let lapNo = 1;

    for (let i = 1; i < points.length; i++) {
      const startPoint = points[startIdx];
      const currentPoint = points[i];
      const segmentDist =
        (currentPoint.distance_m as number) - (startPoint.distance_m as number);

      const splitMeters = /swim/.test(sportType.toLowerCase()) ? 100 : 1000;
      // When we cross a split boundary, or if it's the very last point
      const isLastPoint = i === points.length - 1;
      if (
        segmentDist >= splitMeters ||
        (isLastPoint && segmentDist > splitMeters * 0.1)
      ) {
        const startTs = new Date(startPoint.ts as string).getTime();
        const endTs = new Date(currentPoint.ts as string).getTime();
        const durationS = (endTs - startTs) / 1000;

        // Average HR in this segment
        const hrs = points
          .slice(startIdx, i + 1)
          .map((p) => p.heart_rate)
          .filter((hr): hr is number => hr != null && hr > 0);
        const avgHr =
          hrs.length > 0
            ? hrs.reduce((sum, h) => sum + h, 0) / hrs.length
            : undefined;

        const avgPace =
          segmentDist > 0 ? durationS / (segmentDist / splitMeters) : undefined;

        calculatedLaps.push({
          id: `derived-lap-${lapNo}`,
          lap_no: lapNo,
          distance_m: segmentDist,
          duration_s: durationS,
          avg_hr: avgHr,
          avg_pace_s_per_km: avgPace,
        });

        lapNo++;
        startIdx = i;
      }
    }

    return calculatedLaps;
  }

  const displayLaps = $derived.by<Lap[]>(() => {
    const hasMeasuredLap = laps.some(
      (lap) =>
        (lap.distance_m ?? 0) > 0 ||
        (lap.duration_s ?? 0) > 0 ||
        lap.avg_hr != null ||
        lap.avg_pace_s_per_km != null,
    );
    if (!activity) return [];
    if (processedRoute && processedRoute.length >= 2) {
      const calculated = calculateLapsFromRoute(
        processedRoute,
        activity.sport_type,
      );
      if (calculated.length > 0) return calculated;
    }
    return hasMeasuredLap ? laps : [];
  });

  function formatSplitPace(value?: number): string {
    if (value == null || !Number.isFinite(value) || value <= 0) return "—";
    const minutes = Math.floor(value / 60);
    return `${minutes}:${String(Math.round(value % 60)).padStart(2, "0")} ${isSwimming(activity?.sport_type) ? "/100m" : "/km"}`;
  }
</script>

<svelte:head>
  <title
    >{activity
      ? `${displayTitle(activity.title, activity.sport_type)} · Motion · iroha`
      : "Motion detail · iroha"}</title
  >
</svelte:head>

{#if hasThemeRoute(theme.definition(), "activity-detail")}
  {#if activity || loading}
    <LoadingBoundary
      {loading}
      ready={activity != null}
      label="Loading activity…"
    >
      {#snippet children()}
        {#if activity}
          <ThemeRouteRenderer
            route="activity-detail"
            props={{
              activity,
              derivedDistanceM,
              route,
              samplings,
              laps: displayLaps,
              selectedRouteIndex,
              onSelectRoute: (index: number | null) =>
                (selectedRouteIndex = index),
            }}
          >
            {#snippet children()}
              {#if hasRouteLine}
                <section
                  class="canonical-route-panel"
                  aria-labelledby="route-map-title"
                >
                  <header class="canonical-route-heading">
                    <div>
                      <p class="muted">Canonical geography</p>
                      <h2 id="route-map-title">Recorded route</h2>
                    </div>
                    <span class="muted">{processedRoute.length} GPS fixes</span>
                  </header>
                  <RouteMap
                    points={processedRoute}
                    selectedIndex={selectedRouteIndex}
                  />
                  <p class="muted swim-note">
                    {isSwimming(activity?.sport_type)
                      ? "Open-water GPS route; no pool intervals are inferred from this record."
                      : "The basemap and line use the canonical latitude/longitude record."}
                  </p>
                </section>
              {/if}
              {#if detailChart}
                <ActivityDetailChart
                  {...detailChart}
                  paceLabel={isSwimming(activity?.sport_type)
                    ? "Pace / 100m"
                    : "Pace / km"}
                  onHover={(index) => (selectedRouteIndex = index)}
                />
              {/if}
            {/snippet}
          </ThemeRouteRenderer>
        {/if}
      {/snippet}
    </LoadingBoundary>
  {:else if error}
    <p class="error">Failed to load activity: {error}</p>
  {/if}
{:else}
  <p class="detail-back"><a href="/motion">← Back to Motion</a></p>

  {#if loading}
    <p class="muted">Loading activity…</p>
  {:else if error}
    <p class="error">Failed to load activity: {error}</p>
  {:else if activity}
    <RouteIntro
      eyebrow="Motion / performance report"
      title={displayTitle(activity.title, activity.sport_type)}
      description="A measured record of this session, from the route and effort to the details worth revisiting."
    />
    <div class="activity-meta">
      <SportBadge sport={activity.sport_type} />
      <span class="muted"
        >{formatDate(activity.started_at, activity.timezone)}</span
      >
    </div>

    <div class="activity-stats">
      {#if displayDistanceM != null && displayDistanceM > 0}
        <StatTile
          label={isSwimming(activity.sport_type) ? "GPS distance" : "Distance"}
          value={formatDistance(displayDistanceM)}
        />
      {/if}
      <StatTile label="Duration" value={formatDuration(activity.duration_s)} />
      {#if isSwimming(activity.sport_type)}
        <StatTile
          label="Pace / 100m"
          value={formatSwimmingPace(displayDistanceM, activity.duration_s)}
        />
      {/if}
      {#if activity.moving_time_s != null}
        <StatTile
          label="Moving time"
          value={formatDuration(activity.moving_time_s)}
        />
      {/if}
      {#if elevationGainM != null && elevationGainM > 0}
        <StatTile
          label="Elevation gain"
          value={formatElevation(elevationGainM)}
        />
      {/if}
      {#if activity.avg_pace_s_per_km != null && activity.avg_pace_s_per_km > 0}
        <StatTile
          label="Avg pace"
          value={formatPace(activity.avg_pace_s_per_km)}
        />
      {/if}
      {#if activity.avg_hr != null && activity.avg_hr > 0}
        <StatTile label="Avg HR" value={formatHr(activity.avg_hr)} />
      {/if}
      {#if activity.max_hr != null && activity.max_hr > 0}
        <StatTile label="Max HR" value={formatHr(activity.max_hr)} />
      {/if}
    </div>

    {#if hasRouteLine}
      <h2>Route</h2>
      <RouteMap points={processedRoute} selectedIndex={selectedRouteIndex} />
      {#if isSwimming(activity.sport_type)}
        <p class="muted swim-note">
          Open-water GPS route; no pool intervals are inferred from this record.
        </p>
      {/if}
    {/if}

    {#if anyChart}
      <h2>Charts</h2>
      {#if detailChart}
        <FusedActivityChart
          xValues={detailChart.xValues}
          xLabel={detailChart.xLabel}
          pace={detailChart.pace}
          heartRate={detailChart.heartRate}
          elevation={detailChart.elevation}
          paceLabel={isSwimming(activity.sport_type)
            ? "Pace /100m"
            : "Pace /km"}
          onHover={(index) => (selectedRouteIndex = index)}
        />
      {/if}
      {#if hrZones.length > 0}
        <div class="zone-card tile">
          <div class="card-heading">
            <strong>Heart-rate zones</strong><span class="muted"
              >time distribution</span
            >
          </div>
          <div class="zone-bar">
            {#each hrZones as zone}<span
                style={`flex: ${zone.count}; background: ${zone.color}`}
                title={`${zone.label}: ${Math.round((zone.count / (detailChart?.heartRate.filter((v) => v != null).length || 1)) * 100)}%`}
              ></span>{/each}
          </div>
          <div class="zone-legend">
            {#each hrZones as zone}<span
                ><i style={`background: ${zone.color}`}></i>{zone.label}</span
              >{/each}
          </div>
        </div>
      {/if}
    {/if}

    {#if supportsSplits && displayLaps.length > 0}
      <h2 class="section-title">
        Laps ({isSwimming(activity.sport_type) ? "100 m" : "1 km"} splits)
      </h2>
      <div class="laps-container tile">
        <LapChart
          laps={displayLaps}
          swimming={isSwimming(activity.sport_type)}
        />
        <div class="split-bars" aria-label="Split pace bars">
          {#each displayLaps as lap (lap.lap_no)}
            <div class="split-row">
              <span class="split-label">{lap.lap_no}</span><span
                class="split-track"
                ><span
                  class="split-fill"
                  style={`width: ${Math.min(100, ((lap.avg_pace_s_per_km ?? 0) / Math.max(...displayLaps.map((l) => l.avg_pace_s_per_km ?? 0), 1)) * 100)}%`}
                ></span></span
              ><strong>{formatSplitPace(lap.avg_pace_s_per_km)}</strong>
            </div>
          {/each}
        </div>
        <table class="laps-table">
          <thead>
            <tr>
              <th>Lap</th>
              <th>Distance</th>
              <th>Duration</th>
              <th>Pace</th>
              <th>Avg HR</th>
            </tr>
          </thead>
          <tbody>
            {#each displayLaps as lap (lap.lap_no)}
              <tr>
                <td><strong>{lap.lap_no}</strong></td>
                <td>{formatDistance(lap.distance_m)}</td>
                <td>{formatDuration(lap.duration_s)}</td>
                <td>{formatSplitPace(lap.avg_pace_s_per_km)}</td>
                <td>{formatHr(lap.avg_hr)}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  {/if}
{/if}

<style>
  .activity-meta {
    display: flex;
    gap: 0.75rem;
    align-items: center;
    margin-bottom: 1rem;
  }
  .activity-stats {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(9rem, 1fr));
    gap: 0.75rem;
    margin-bottom: 1.5rem;
  }
  .canonical-route-panel {
    display: grid;
    gap: 0.75rem;
    margin-bottom: 1.5rem;
    padding: 1rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--surface) 92%, transparent);
  }
  .canonical-route-heading {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    align-items: end;
  }
  .canonical-route-heading h2 {
    margin: 0.2rem 0 0;
    font-size: 1.2rem;
  }
  .canonical-route-panel :global(.map) {
    border-color: color-mix(in srgb, var(--accent) 32%, var(--border));
  }
  .canonical-route-panel .swim-note {
    margin: 0;
    font-size: 0.76rem;
  }
  .section-title {
    margin-top: 2rem;
    margin-bottom: 0.75rem;
    font-size: 1.25rem;
    font-weight: 700;
  }
  .laps-container {
    overflow-x: auto;
    padding: 0.5rem;
    margin-bottom: 2rem;
  }
  .zone-card {
    padding: 1rem;
    margin-bottom: 1.5rem;
  }
  .card-heading,
  .zone-legend {
    display: flex;
    justify-content: space-between;
    gap: 0.75rem;
    align-items: center;
  }
  .zone-bar {
    display: flex;
    height: 0.65rem;
    overflow: hidden;
    border-radius: 999px;
    margin: 1rem 0 0.75rem;
    background: var(--surface-3);
  }
  .zone-bar span {
    min-width: 0.35rem;
  }
  .zone-legend {
    justify-content: flex-start;
    flex-wrap: wrap;
    color: var(--text-muted);
    font-size: 0.76rem;
  }
  .zone-legend span {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
  }
  .zone-legend i {
    width: 0.45rem;
    height: 0.45rem;
    border-radius: 50%;
  }
  .split-bars {
    display: grid;
    gap: 0.45rem;
    padding: 0.75rem 1rem 1rem;
    border-bottom: 1px solid var(--border);
  }
  .split-row {
    display: grid;
    grid-template-columns: 1.5rem 1fr auto;
    align-items: center;
    gap: 0.6rem;
    font-size: 0.8rem;
  }
  .split-label {
    color: var(--text-muted);
  }
  .split-track {
    height: 0.45rem;
    overflow: hidden;
    border-radius: 999px;
    background: var(--surface-3);
  }
  .split-fill {
    display: block;
    height: 100%;
    border-radius: inherit;
    background: linear-gradient(90deg, var(--sport-run), var(--sport-cycling));
  }
  .laps-table {
    width: 100%;
    border-collapse: collapse;
    text-align: left;
    font-size: 0.9rem;
  }
  .laps-table th {
    padding: 0.75rem 1rem;
    color: var(--text-muted);
    font-size: 0.76rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    border-bottom: 1px solid var(--border);
  }
  .laps-table td {
    padding: 0.75rem 1rem;
    color: var(--text);
    border-bottom: 1px solid var(--border);
  }
  .laps-table tbody tr:last-child td {
    border-bottom: none;
  }
  .laps-table tbody tr:hover td {
    background: var(--surface-2);
  }
</style>
