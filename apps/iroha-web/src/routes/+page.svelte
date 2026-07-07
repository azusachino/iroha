<script lang="ts">
	import { listActivities, type Activity } from '$lib/api';
	import { formatDistance, formatDuration, formatDateOnly } from '$lib/format';

	let sportType = $state('');
	let activities = $state<Activity[]>([]);
	let sportOptions = $state<string[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	async function load() {
		loading = true;
		error = null;
		try {
			const rows = await listActivities(sportType ? { sport_type: sportType } : {});
			activities = rows;
			// Keep a stable, growing set of known sport types for the filter.
			const known = new Set(sportOptions);
			for (const a of rows) if (a.sport_type) known.add(a.sport_type);
			sportOptions = [...known].sort();
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
			activities = [];
		} finally {
			loading = false;
		}
	}

	// Re-fetch whenever the selected sport type changes.
	$effect(() => {
		sportType;
		load();
	});
</script>

<h1>Activities</h1>

<div class="filters">
	<label for="sport">Sport type</label>
	<select id="sport" bind:value={sportType}>
		<option value="">All</option>
		{#each sportOptions as option (option)}
			<option value={option}>{option}</option>
		{/each}
	</select>
</div>

{#if loading}
	<p class="muted">Loading activities…</p>
{:else if error}
	<p class="error">Failed to load activities: {error}</p>
{:else if activities.length === 0}
	<p class="muted">No activities found.</p>
{:else}
	<ul class="activity-list">
		{#each activities as activity (activity.id)}
			<li>
				<a class="activity-card" href={`/activities/${activity.id}`}>
					<div class="title">{activity.title || 'Untitled activity'}</div>
					<div class="meta">
						<span class="badge">{activity.sport_type}</span>
						<span>{formatDateOnly(activity.started_at, activity.timezone)}</span>
						<span>{formatDistance(activity.distance_m)}</span>
						<span>{formatDuration(activity.duration_s ?? activity.moving_time_s)}</span>
					</div>
				</a>
			</li>
		{/each}
	</ul>
{/if}
