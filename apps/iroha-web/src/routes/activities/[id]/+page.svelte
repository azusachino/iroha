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

	{#if laps.length > 0}
		<h2>Laps</h2>
		<ul class="activity-list">
			{#each laps as lap (lap.id)}
				<li class="activity-card">
					<div class="meta">
						<span class="badge">Lap {lap.lap_no}</span>
						<span>{formatDistance(lap.distance_m)}</span>
						<span>{formatDuration(lap.duration_s)}</span>
						<span>{formatPace(lap.avg_pace_s_per_km)}</span>
						<span>{formatHr(lap.avg_hr)}</span>
					</div>
				</li>
			{/each}
		</ul>
	{/if}
{/if}

<style>
	.activity-meta { display: flex; gap: 0.75rem; align-items: center; margin-bottom: 1rem; }
	.activity-stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(9rem, 1fr)); gap: 0.75rem; margin-bottom: 1.5rem; }
</style>
