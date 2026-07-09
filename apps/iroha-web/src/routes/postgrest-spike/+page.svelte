<script lang="ts">
	import {
		getPostgrestPublicSummary,
		listPostgrestPublicActivities,
		type PostgrestPublicActivity,
		type PostgrestSummary
	} from '$lib/postgrest/client';
	import { formatDate, formatDistance, formatDuration, formatSport } from '$lib/format';

	const PAGE_SIZE = 20;

	let summary = $state<PostgrestSummary | null>(null);
	let activities = $state<PostgrestPublicActivity[]>([]);
	let sportOptions = $state<string[]>([]);
	let sportFilter = $state('');
	let loading = $state(true);
	let loadingMore = $state(false);
	let error = $state<string | null>(null);
	let offset = $state(0);
	let hasMore = $state(false);

	async function load(append: boolean) {
		if (append) loadingMore = true;
		else loading = true;
		error = null;
		try {
			if (!append) {
				summary = await getPostgrestPublicSummary();
				sportOptions = summary.by_sport.map((bucket) => bucket.key).sort();
			}

			const rows = await listPostgrestPublicActivities({
				limit: PAGE_SIZE,
				offset: append ? offset : 0,
				sport_type: sportFilter || undefined
			});
			activities = append ? [...activities, ...rows] : rows;
			offset = append ? offset + rows.length : rows.length;
			hasMore = rows.length === PAGE_SIZE;
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
			if (!append) activities = [];
		} finally {
			loading = false;
			loadingMore = false;
		}
	}

	function applySport() {
		offset = 0;
		load(false);
	}

	$effect(() => {
		load(false);
	});
</script>

<h1>PostgREST Spike</h1>

{#if loading}
	<p class="muted">Loading PostgREST data...</p>
{:else if error}
	<p class="error">Failed to load PostgREST data: {error}</p>
{:else}
	{#if summary}
		<div class="mb-6 grid grid-cols-1 gap-3 sm:grid-cols-3">
			<div class="rounded-lg border border-border bg-surface p-4 text-center">
				<div class="text-xs text-text-muted uppercase">Distance</div>
				<div class="mt-1 text-2xl font-bold">{formatDistance(summary.totals.distance_m)}</div>
			</div>
			<div class="rounded-lg border border-border bg-surface p-4 text-center">
				<div class="text-xs text-text-muted uppercase">Activities</div>
				<div class="mt-1 text-2xl font-bold">{summary.totals.activity_count}</div>
			</div>
			<div class="rounded-lg border border-border bg-surface p-4 text-center">
				<div class="text-xs text-text-muted uppercase">Time</div>
				<div class="mt-1 text-2xl font-bold">{formatDuration(summary.totals.duration_s)}</div>
			</div>
		</div>
	{/if}

	<form class="filters" onsubmit={(event) => event.preventDefault()}>
		<label>
			Sport
			<select bind:value={sportFilter} onchange={applySport}>
				<option value="">All</option>
				{#each sportOptions as sport (sport)}
					<option value={sport}>{formatSport(sport)}</option>
				{/each}
			</select>
		</label>
	</form>

	<p class="muted result-count">
		{activities.length} shown{hasMore ? ' (more available)' : ''}
	</p>

	<ul class="activity-list">
		{#each activities as activity (activity.id)}
			<li>
				<div class="activity-card">
					<div class="title">{activity.title || 'Untitled activity'}</div>
					<div class="meta">
						<span class="badge">{formatSport(activity.sport_type)}</span>
						<span>{formatDate(activity.started_at, activity.timezone)}</span>
						<span>{formatDistance(activity.distance_m)}</span>
						<span>{formatDuration(activity.duration_s ?? activity.moving_time_s)}</span>
					</div>
				</div>
			</li>
		{/each}
	</ul>

	{#if hasMore}
		<button class="load-more" onclick={() => load(true)} disabled={loadingMore}>
			{loadingMore ? 'Loading...' : 'Load more'}
		</button>
	{/if}
{/if}
