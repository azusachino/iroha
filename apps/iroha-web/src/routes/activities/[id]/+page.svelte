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
	<div class="filters">
		<span class="badge">{activity.sport_type}</span>
		<span class="muted">{formatDate(activity.started_at, activity.timezone)}</span>
	</div>

	<div class="metrics">
		<div class="metric">
			<div class="label">Distance</div>
			<div class="value">{formatDistance(activity.distance_m)}</div>
		</div>
		<div class="metric">
			<div class="label">Duration</div>
			<div class="value">{formatDuration(activity.duration_s)}</div>
		</div>
		<div class="metric">
			<div class="label">Moving time</div>
			<div class="value">{formatDuration(activity.moving_time_s)}</div>
		</div>
		<div class="metric">
			<div class="label">Elevation gain</div>
			<div class="value">{formatElevation(activity.elevation_gain_m)}</div>
		</div>
		<div class="metric">
			<div class="label">Avg pace</div>
			<div class="value">{formatPace(activity.avg_pace_s_per_km)}</div>
		</div>
		<div class="metric">
			<div class="label">Avg HR</div>
			<div class="value">{formatHr(activity.avg_hr)}</div>
		</div>
		<div class="metric">
			<div class="label">Max HR</div>
			<div class="value">{formatHr(activity.max_hr)}</div>
		</div>
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
