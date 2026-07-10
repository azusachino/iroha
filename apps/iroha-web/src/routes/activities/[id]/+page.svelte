<script lang="ts">
	import { page } from '$app/state';
	import {
		getActivity,
		getActivityRoute,
		getActivitySamplings,
		getActivityLaps,
		type Activity,
		type RoutePoint,
		type SamplingPoint,
		type Lap
	} from '$lib/api';
	import {
		formatDistance,
		formatDuration,
		formatPace,
		formatElevation,
		formatHr,
		formatDate
	} from '$lib/format';
	import RouteMap from '$lib/components/RouteMap.svelte';
	import LineChart, { type ChartSeries } from '$lib/components/LineChart.svelte';
	import SportBadge from '$lib/components/SportBadge.svelte';
	import StatTile from '$lib/components/StatTile.svelte';

	let activity = $state<Activity | null>(null);
	let route = $state<RoutePoint[]>([]);
	let samplings = $state<SamplingPoint[]>([]);
	let laps = $state<Lap[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	const id = $derived(page.params.id ?? '');

	$effect(() => {
		const activityId = id;
		if (!activityId) return;
		loading = true;
		error = null;
		Promise.all([
			getActivity(activityId),
			// Sub-resources are best-effort; an empty/failed one should not blank the page.
			getActivityRoute(activityId).catch(() => [] as RoutePoint[]),
			getActivitySamplings(activityId).catch(() => [] as SamplingPoint[]),
			getActivityLaps(activityId).catch(() => [] as Lap[])
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

	// Choose an x-axis shared by all route-derived charts: distance if every
	// point has it, else elapsed time, else the raw sequence number.
	interface XAxis {
		values: number[];
		label: string;
	}
	const xAxis = $derived.by<XAxis>(() => {
		if (route.length === 0) return { values: [], label: 'Point' };
		if (route.every((p) => p.distance_m != null)) {
			return { values: route.map((p) => (p.distance_m as number) / 1000), label: 'Distance (km)' };
		}
		if (route.every((p) => p.ts != null)) {
			const t0 = new Date(route[0].ts as string).getTime();
			return {
				values: route.map((p) => (new Date(p.ts as string).getTime() - t0) / 60000),
				label: 'Time (min)'
			};
		}
		return { values: route.map((p) => p.seq), label: 'Point' };
	});

	function column<T>(get: (p: RoutePoint) => T | null | undefined): (number | null)[] {
		return route.map((p) => {
			const v = get(p);
			return v == null || !Number.isFinite(v as number) ? null : (v as number);
		});
	}

	function hasData(values: (number | null)[]): boolean {
		return values.some((v) => v != null);
	}

	// Pace derived from speed (s/km = 1000 / m/s); guard against zero/idle speed.
	const paceSeries = $derived.by<(number | null)[]>(() =>
		route.map((p) => {
			if (p.speed_mps == null || !Number.isFinite(p.speed_mps) || p.speed_mps <= 0) return null;
			return 1000 / p.speed_mps;
		})
	);
	const hrSeries = $derived.by(() => column((p) => p.heart_rate));
	const elevationSeries = $derived.by(() => column((p) => p.elevation_m));

	// Apple exports carry no activity-level elevation gain, but route points do
	// have elevation — estimate cumulative climb from them (3 m hysteresis to
	// damp GPS noise) when the stored value is absent.
	const elevationGainM = $derived.by<number | undefined>(() => {
		if (activity?.elevation_gain_m != null) return activity.elevation_gain_m;
		const elevs = route
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
		const hr = samplings.filter((s) => /heart|(^|_)hr($|_)/i.test(s.sampling_type));
		if (hr.length === 0) return null;
		const t0 = new Date(hr[0].ts).getTime();
		return {
			x: hr.map((s) => (new Date(s.ts).getTime() - t0) / 60000),
			values: hr.map((s) => (Number.isFinite(s.value) ? s.value : null))
		};
	});

	const paceChart = $derived.by<ChartSeries[] | null>(() =>
		hasData(paceSeries) ? [{ label: 'Pace (s/km)', values: paceSeries, stroke: '#4f8cff' }] : null
	);
	const hrChart = $derived.by<ChartSeries[] | null>(() =>
		hasData(hrSeries) ? [{ label: 'Heart rate (bpm)', values: hrSeries, stroke: '#ff6b6b' }] : null
	);
	const elevationChart = $derived.by<ChartSeries[] | null>(() =>
		hasData(elevationSeries)
			? [{ label: 'Elevation (m)', values: elevationSeries, stroke: '#3ecf8e' }]
			: null
	);

	const hasRouteLine = $derived(
		route.filter((p) => Number.isFinite(p.lat) && Number.isFinite(p.lon)).length >= 2
	);
	const anyChart = $derived(
		!!paceChart || !!hrChart || !!elevationChart || !!hrSamplingChart
	);

	const isRun = $derived(activity?.sport_type?.toLowerCase() === 'run');

	interface CalculatedLap {
		lap_no: number;
		distance_m: number;
		duration_s: number;
		avg_hr?: number;
		avg_pace_s_per_km?: number;
	}

	function calculateLapsFromRoute(routePoints: RoutePoint[], sportType: string): CalculatedLap[] {
		if (sportType.toLowerCase() !== 'run') return [];
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
			const segmentDist = (currentPoint.distance_m as number) - (startPoint.distance_m as number);

			// When we cross a 1000m boundary, or if it's the very last point
			const isLastPoint = i === points.length - 1;
			if (segmentDist >= 1000 || (isLastPoint && segmentDist > 10)) {
				const startTs = new Date(startPoint.ts as string).getTime();
				const endTs = new Date(currentPoint.ts as string).getTime();
				const durationS = (endTs - startTs) / 1000;

				// Average HR in this segment
				const hrs = points
					.slice(startIdx, i + 1)
					.map((p) => p.heart_rate)
					.filter((hr): hr is number => hr != null && hr > 0);
				const avgHr = hrs.length > 0 ? hrs.reduce((sum, h) => sum + h, 0) / hrs.length : undefined;

				const avgPace = segmentDist > 0 ? durationS / (segmentDist / 1000) : undefined;

				calculatedLaps.push({
					lap_no: lapNo,
					distance_m: segmentDist,
					duration_s: durationS,
					avg_hr: avgHr,
					avg_pace_s_per_km: avgPace
				});

				lapNo++;
				startIdx = i;
			}
		}

		return calculatedLaps;
	}

	const displayLaps = $derived.by<CalculatedLap[]>(() => {
		if (!activity || activity.sport_type.toLowerCase() !== 'run') return [];
		if (route && route.length >= 2) {
			const calculated = calculateLapsFromRoute(route, activity.sport_type);
			if (calculated.length > 0) return calculated;
		}
		// If we don't have route points but we have laps from the database, map them.
		if (laps && laps.length > 0) {
			return laps.map((l) => ({
				lap_no: l.lap_no,
				distance_m: l.distance_m ?? 1000,
				duration_s: l.duration_s ?? 0,
				avg_hr: l.avg_hr,
				avg_pace_s_per_km: l.avg_pace_s_per_km
			}));
		}
		return [];
	});
</script>

<p><a href="/">← Back to activities</a></p>

{#if loading}
	<p class="muted">Loading activity…</p>
{:else if error}
	<p class="error">Failed to load activity: {error}</p>
{:else if activity}
	<h1>{activity.title || 'Untitled activity'}</h1>
	<div class="activity-meta">
		<SportBadge sport={activity.sport_type} />
		<span class="muted">{formatDate(activity.started_at, activity.timezone)}</span>
	</div>

	<div class="activity-stats">
		<StatTile label="Distance" value={formatDistance(activity.distance_m)} />
		<StatTile label="Duration" value={formatDuration(activity.duration_s)} />
		{#if activity.moving_time_s != null}
			<StatTile label="Moving time" value={formatDuration(activity.moving_time_s)} />
		{/if}
		<StatTile label="Elevation gain" value={formatElevation(elevationGainM)} />
		<StatTile label="Avg pace" value={formatPace(activity.avg_pace_s_per_km)} />
		<StatTile label="Avg HR" value={formatHr(activity.avg_hr)} />
		<StatTile label="Max HR" value={formatHr(activity.max_hr)} />
	</div>

	{#if hasRouteLine}
		<h2>Route</h2>
		<RouteMap points={route} />
	{/if}

	{#if anyChart}
		<h2>Charts</h2>
		{#if paceChart}
			<LineChart title="Pace" xValues={xAxis.values} xLabel={xAxis.label} series={paceChart} />
		{/if}
		{#if hrChart}
			<LineChart title="Heart rate" xValues={xAxis.values} xLabel={xAxis.label} series={hrChart} />
		{:else if hrSamplingChart}
			<LineChart
				title="Heart rate"
				xValues={hrSamplingChart.x}
				xLabel="Time (min)"
				series={[{ label: 'Heart rate (bpm)', values: hrSamplingChart.values, stroke: '#ff6b6b' }]}
			/>
		{/if}
		{#if elevationChart}
			<LineChart
				title="Elevation"
				xValues={xAxis.values}
				xLabel={xAxis.label}
				series={elevationChart}
			/>
		{/if}
	{/if}

	{#if isRun && displayLaps.length > 0}
		<h2 class="section-title">Laps (1 km splits)</h2>
		<div class="laps-container tile">
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
							<td>{formatPace(lap.avg_pace_s_per_km)}</td>
							<td>{formatHr(lap.avg_hr)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
{/if}

<style>
	.activity-meta { display: flex; gap: 0.75rem; align-items: center; margin-bottom: 1rem; }
	.activity-stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(9rem, 1fr)); gap: 0.75rem; margin-bottom: 1.5rem; }
	.section-title { margin-top: 2rem; margin-bottom: 0.75rem; font-size: 1.25rem; font-weight: 700; }
	.laps-container { overflow-x: auto; padding: 0.5rem; margin-bottom: 2rem; }
	.laps-table { width: 100%; border-collapse: collapse; text-align: left; font-size: 0.9rem; }
	.laps-table th { padding: 0.75rem 1rem; color: var(--text-muted); font-size: 0.76rem; font-weight: 700; letter-spacing: 0.08em; text-transform: uppercase; border-bottom: 1px solid var(--border); }
	.laps-table td { padding: 0.75rem 1rem; color: var(--text); border-bottom: 1px solid var(--border); }
	.laps-table tbody tr:last-child td { border-bottom: none; }
	.laps-table tbody tr:hover td { background: var(--surface-2); }
</style>
