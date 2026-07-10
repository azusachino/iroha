<script lang="ts">
	import { onMount } from 'svelte';
	import { getPublicSummary, listActivities, type Activity, type ListActivitiesParams, type Summary } from '$lib/api';
	import SportBadge from '$lib/components/SportBadge.svelte';
	import StatTile from '$lib/components/StatTile.svelte';
	import { formatDate, formatDistance, formatDuration, formatPace } from '$lib/format';
	import { sportColor } from '$lib/sport';

	// Draft filter inputs (bound to the form); committed to `applied` on submit
	// so "Load more" keeps paging the same query the user actually ran.
	let sportType = $state('');
	let dateFrom = $state('');
	let dateTo = $state('');
	let minKm = $state('');
	let maxKm = $state('');
	let applied = $state<ListActivitiesParams>({});

	let activities = $state<Activity[]>([]);
	let sportOptions = $state<string[]>([]);
	let loading = $state(true);
	let loadingMore = $state(false);
	let cursor = $state<string | null>(null);
	let hasMore = $state(false);
	let error = $state<string | null>(null);
	let summary = $state<Summary | null>(null);
	let summaryLoading = $state(true);

	function buildParams(): ListActivitiesParams {
		const params: ListActivitiesParams = {};
		if (sportType) params.sport_type = sportType;
		// Widen date-only inputs to full-day UTC bounds (approximate; good enough
		// for exploring — a real UI would honor the activity's own timezone).
		if (dateFrom) params.started_from = `${dateFrom}T00:00:00Z`;
		if (dateTo) params.started_to = `${dateTo}T23:59:59Z`;
		if (minKm) params.min_distance_m = Number(minKm) * 1000;
		if (maxKm) params.max_distance_m = Number(maxKm) * 1000;
		return params;
	}

	// Fetch one page. `append` distinguishes a fresh query (replace) from
	// "Load more" (accumulate). Cursor + has_more drive the keyset walk.
	async function load(append: boolean) {
		if (append) loadingMore = true;
		else loading = true;
		error = null;
		try {
			const params: ListActivitiesParams = { ...applied };
			if (append && cursor) params.cursor = cursor;
			const page = await listActivities(params);
			activities = append ? [...activities, ...page.items] : page.items;
			cursor = page.next_cursor;
			hasMore = page.has_more;
			// Keep a stable, growing set of known sport types for the filter.
			const known = new Set(sportOptions);
			for (const a of activities) if (a.sport_type) known.add(a.sport_type);
			sportOptions = [...known].sort();
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
			if (!append) activities = [];
		} finally {
			loading = false;
			loadingMore = false;
		}
	}

	function apply(event: SubmitEvent) {
		event.preventDefault();
		applied = buildParams();
		cursor = null;
		load(false);
	}

	function clear() {
		sportType = '';
		dateFrom = '';
		dateTo = '';
		minKm = '';
		maxKm = '';
		applied = {};
		cursor = null;
		load(false);
	}

	async function loadSummary() {
		try {
			summary = await getPublicSummary();
		} finally {
			summaryLoading = false;
		}
	}

	const totalDuration = $derived(formatDuration(summary?.totals.moving_time_s || summary?.totals.duration_s));
	const trackedSports = $derived(summary?.by_sport.length ?? 0);

	// Initial unfiltered load (runs once — no reactive dependencies read).
	$effect(() => {
		load(false);
	});

	onMount(() => {
		void loadSummary();
	});
</script>

<section class="activities-shell">
	<header class="domain-header">
		<div>
			<p class="eyebrow">Activity domain</p>
			<h1>Every session, in one place.</h1>
			<p class="muted">Explore the record, then open an activity for its route and measurements.</p>
		</div>
	</header>

	<div class="stat-strip" aria-label="Activity summary">
		<StatTile label="Activities" value={summaryLoading ? '—' : (summary?.totals.activity_count ?? 0).toLocaleString()} sub="Imported sessions" />
		<StatTile label="Distance" value={summaryLoading ? '—' : formatDistance(summary?.totals.distance_m)} sub="Across all activities" />
		<StatTile label="Total time" value={summaryLoading ? '—' : totalDuration} sub="Recorded duration" />
		<StatTile label="Sports" value={summaryLoading ? '—' : trackedSports.toLocaleString()} sub="Activity types tracked" />
	</div>

	<form class="activity-toolbar tile" onsubmit={apply}>
		<div class="filter-fields">
			<label>Sport <select bind:value={sportType}><option value="">All sports</option>{#each sportOptions as option (option)}<option value={option}>{option}</option>{/each}</select></label>
			<label>From <input type="date" bind:value={dateFrom} /></label>
			<label>To <input type="date" bind:value={dateTo} /></label>
			<label>Min km <input type="number" min="0" step="0.1" bind:value={minKm} /></label>
			<label>Max km <input type="number" min="0" step="0.1" bind:value={maxKm} /></label>
		</div>
		<div class="toolbar-actions"><button type="submit">Apply filters</button><button type="button" class="secondary" onclick={clear}>Clear</button></div>
	</form>

	{#if loading}
		<p class="muted">Loading activities…</p>
	{:else if error}
		<p class="error">Failed to load activities: {error}</p>
	{:else if activities.length === 0}
		<p class="muted">No activities found.</p>
	{:else}
		<p class="muted result-count">{activities.length} shown{hasMore ? ' (more available)' : ''}</p>
		<ul class="activity-grid">
			{#each activities as activity (activity.id)}
				<li>
					<a class="activity-card tile tile-interactive" href={`/activities/${activity.id}`} style={`--sport-color: ${sportColor(activity.sport_type)}`}>
						<span class="accent" aria-hidden="true"></span>
						<div class="card-top"><SportBadge sport={activity.sport_type} /><span class="activity-date">{formatDate(activity.started_at, activity.timezone)}</span></div>
						<h2>{activity.title || 'Untitled activity'}</h2>
						<div class="primary-metric">{formatDistance(activity.distance_m)}</div>
						<div class="card-metrics"><span>{formatPace(activity.avg_pace_s_per_km)}</span><span>{formatDuration(activity.duration_s ?? activity.moving_time_s)}</span></div>
					</a>
				</li>
			{/each}
		</ul>
		{#if hasMore}<button class="load-more" onclick={() => load(true)} disabled={loadingMore}>{loadingMore ? 'Loading…' : 'Load more activities'}</button>{/if}
	{/if}
</section>

<style>
	.activities-shell { display: grid; gap: 1.25rem; }
	.domain-header h1 { margin: 0; }
	.domain-header .muted { margin: 0.4rem 0 0; }
	.eyebrow { margin: 0 0 0.4rem; color: var(--accent); font-size: 0.72rem; font-weight: 700; letter-spacing: 0.08em; text-transform: uppercase; }
	.stat-strip { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 0.75rem; }
	.activity-toolbar { display: flex; align-items: end; justify-content: space-between; gap: 1rem; padding: 1rem; }
	.filter-fields { display: flex; flex: 1; flex-wrap: wrap; gap: 0.75rem; }
	.filter-fields label { display: flex; flex-direction: column; gap: 0.3rem; color: var(--text-muted); font-size: 0.76rem; font-weight: 650; }
	.toolbar-actions { display: flex; gap: 0.5rem; }
	.toolbar-actions button { border: 1px solid var(--accent); border-radius: var(--radius); background: var(--accent); color: var(--bg); padding: 0.5rem 0.75rem; font: inherit; font-size: 0.84rem; cursor: pointer; }
	.toolbar-actions .secondary { border-color: var(--border); background: var(--surface-2); color: var(--text); }
	.activity-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 0.75rem; list-style: none; margin: 0; padding: 0; }
	.activity-card { position: relative; display: grid; gap: 0.75rem; min-height: 13rem; padding: 1rem; overflow: hidden; color: var(--text); text-decoration: none; }
	.activity-card:hover { text-decoration: none; }
	.accent { position: absolute; inset: 0 auto 0 0; width: 0.25rem; background: var(--sport-color); }
	.card-top { display: flex; justify-content: space-between; gap: 0.5rem; }
	.activity-date { color: var(--text-muted); font-size: 0.72rem; text-align: right; }
	.activity-card h2 { margin: 0; font-size: 1rem; line-height: 1.25; }
	.primary-metric { align-self: end; color: var(--text); font-size: clamp(1.45rem, 3vw, 2rem); font-weight: 750; line-height: 1; }
	.card-metrics { display: flex; justify-content: space-between; gap: 0.5rem; color: var(--text-muted); font-size: 0.8rem; }
	@media (max-width: 800px) { .stat-strip { grid-template-columns: repeat(2, minmax(0, 1fr)); } .activity-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } .activity-toolbar { align-items: stretch; flex-direction: column; } }
	@media (max-width: 560px) { .activity-grid { grid-template-columns: 1fr; } .toolbar-actions { width: 100%; } .toolbar-actions button { flex: 1; } .activity-date { max-width: 9rem; } }
</style>
