<script lang="ts">
	import { listActivities, type Activity, type ListActivitiesParams } from '$lib/api';
	import { formatDistance, formatDuration, formatDate, formatSport } from '$lib/format';

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

	// Initial unfiltered load (runs once — no reactive dependencies read).
	$effect(() => {
		load(false);
	});
</script>

<h1>Activities</h1>

<form class="filters" onsubmit={apply}>
	<label>
		Sport
		<select bind:value={sportType}>
			<option value="">All</option>
			{#each sportOptions as option (option)}
				<option value={option}>{option}</option>
			{/each}
		</select>
	</label>
	<label>
		From
		<input type="date" bind:value={dateFrom} />
	</label>
	<label>
		To
		<input type="date" bind:value={dateTo} />
	</label>
	<label>
		Min km
		<input type="number" min="0" step="0.1" bind:value={minKm} />
	</label>
	<label>
		Max km
		<input type="number" min="0" step="0.1" bind:value={maxKm} />
	</label>
	<button type="submit">Apply</button>
	<button type="button" class="secondary" onclick={clear}>Clear</button>
</form>

{#if loading}
	<p class="muted">Loading activities…</p>
{:else if error}
	<p class="error">Failed to load activities: {error}</p>
{:else if activities.length === 0}
	<p class="muted">No activities found.</p>
{:else}
	<p class="muted result-count">
		{activities.length} shown{hasMore ? ' (more available)' : ''}
	</p>
	<ul class="activity-list">
		{#each activities as activity (activity.id)}
			<li>
				<a class="activity-card" href={`/activities/${activity.id}`}>
					<div class="title">{activity.title || 'Untitled activity'}</div>
					<div class="meta">
						<span class="badge">{formatSport(activity.sport_type)}</span>
						<span>{formatDate(activity.started_at, activity.timezone)}</span>
						<span>{formatDistance(activity.distance_m)}</span>
						<span>{formatDuration(activity.duration_s ?? activity.moving_time_s)}</span>
					</div>
				</a>
			</li>
		{/each}
	</ul>

	{#if hasMore}
		<button class="load-more" onclick={() => load(true)} disabled={loadingMore}>
			{loadingMore ? 'Loading…' : 'Load more'}
		</button>
	{/if}
{/if}
