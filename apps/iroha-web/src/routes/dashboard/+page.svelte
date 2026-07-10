<script lang="ts">
	import { onMount } from 'svelte';
	import {
		getPublicRoutes,
		getPublicSummary,
		listActivities,
		type Activity,
		type RouteFeatureCollection,
		type Summary
	} from '$lib/api';
	import DomainTile from '$lib/components/DomainTile.svelte';
	import Heatmap from '$lib/components/Heatmap.svelte';
	import RoutesMap from '$lib/components/RoutesMap.svelte';
	import SportBadge from '$lib/components/SportBadge.svelte';
	import StatTile from '$lib/components/StatTile.svelte';
	import { formatDate, formatDistance, formatDuration } from '$lib/format';
	import { currentActivityStreak } from '$lib/streak';

	const ACTIVITY_SWEEP_LIMIT = 500;
	const RECENT_ACTIVITY_LIMIT = 5;

	let summary = $state<Summary | null>(null);
	let summaryError = $state<string | null>(null);
	let summaryLoading = $state(true);

	let activities = $state<Activity[]>([]);
	let activitiesError = $state<string | null>(null);
	let activitiesLoading = $state(true);

	let routes = $state<RouteFeatureCollection | null>(null);
	let routesError = $state<string | null>(null);
	let routesLoading = $state(true);

	const recentActivities = $derived(activities.slice(0, RECENT_ACTIVITY_LIMIT));
	const heatmapDates = $derived(activities.map((activity) => activity.started_at));
	const streak = $derived(currentActivityStreak(heatmapDates));
	const hasRoutes = $derived((routes?.features.length ?? 0) > 0);
	const activityCount = $derived(summary?.totals.activity_count ?? 0);
	const totalDistance = $derived(formatDistance(summary?.totals.distance_m));
	const totalDuration = $derived(
		formatDuration(summary?.totals.moving_time_s || summary?.totals.duration_s)
	);
	const streakValue = $derived(streak === 1 ? '1 day' : `${streak} days`);

	function errorMessage(error: unknown): string {
		return error instanceof Error ? error.message : String(error);
	}

	async function loadSummary() {
		summaryLoading = true;
		summaryError = null;
		try {
			summary = await getPublicSummary();
		} catch (error) {
			summaryError = errorMessage(error);
		} finally {
			summaryLoading = false;
		}
	}

	async function loadActivities() {
		activitiesLoading = true;
		activitiesError = null;
		try {
			activities = (await listActivities({ limit: ACTIVITY_SWEEP_LIMIT })).items;
		} catch (error) {
			activitiesError = errorMessage(error);
		} finally {
			activitiesLoading = false;
		}
	}

	async function loadRoutes() {
		routesLoading = true;
		routesError = null;
		try {
			routes = await getPublicRoutes();
		} catch (error) {
			routesError = errorMessage(error);
		} finally {
			routesLoading = false;
		}
	}

	onMount(() => {
		void loadSummary();
		void loadActivities();
		void loadRoutes();
	});
</script>

<section class="dashboard-shell">
	<header class="dashboard-heading">
		<div>
			<p class="eyebrow">Dashboard</p>
			<h1>Your data cockpit</h1>
			<p class="muted">A living view of your activity record.</p>
		</div>
		<a class="activity-link" href="/activities">Explore activities</a>
	</header>

	<div class="stats-grid" aria-label="Activity totals">
		<StatTile
			label="Total distance"
			value={summaryLoading || summaryError ? '—' : totalDistance}
			sub={summaryLoading ? 'Loading totals…' : summaryError ? 'Summary unavailable' : undefined}
		/>
		<StatTile
			label="Activities"
			value={summaryLoading || summaryError ? '—' : activityCount.toLocaleString()}
			sub={summaryLoading ? 'Loading totals…' : summaryError ? 'Summary unavailable' : undefined}
		/>
		<StatTile
			label="Total time"
			value={summaryLoading || summaryError ? '—' : totalDuration}
			sub={summaryLoading ? 'Loading totals…' : summaryError ? 'Summary unavailable' : undefined}
		/>
		<StatTile
			label="Current streak"
			value={activitiesLoading || activitiesError ? '—' : streakValue}
			sub={activitiesLoading ? 'Loading activity days…' : activitiesError ? 'Activity history unavailable' : 'Consecutive days ending today'}
		/>
	</div>

	<div class="bento-grid">
		{#if activitiesLoading}
			<section class="status-tile tile heatmap-tile"><p>Loading activity history…</p></section>
		{:else if activitiesError}
			<section class="status-tile tile heatmap-tile"><p>Activity history could not be loaded.</p></section>
		{:else}
			<Heatmap dates={heatmapDates} title="Activity history" />
		{/if}

		<section class="recent-tile tile">
			<header class="tile-heading">
				<div>
					<h2>Recent activity</h2>
					<p>Start where you last left off.</p>
				</div>
				<a href="/activities">View all</a>
			</header>
			{#if activitiesLoading}
				<p class="muted">Loading activities…</p>
			{:else if activitiesError}
				<p class="error">Recent activity could not be loaded.</p>
			{:else if recentActivities.length === 0}
				<p class="muted">No activities imported yet.</p>
			{:else}
				<ul class="recent-list">
					{#each recentActivities as activity (activity.id)}
						<li>
							<a class="recent-row" href={`/activities/${activity.id}`}>
								<SportBadge sport={activity.sport_type} />
								<span class="recent-title">{activity.title || 'Untitled activity'}</span>
								<span class="recent-metrics">
									{formatDistance(activity.distance_m)} · {formatDuration(activity.duration_s ?? activity.moving_time_s)}
								</span>
								<span class="recent-date">{formatDate(activity.started_at, activity.timezone)}</span>
							</a>
						</li>
					{/each}
				</ul>
			{/if}
		</section>

		<section class="routes-tile tile">
			<header class="tile-heading">
				<div>
					<h2>All routes</h2>
					<p>Your privacy-trimmed route footprint.</p>
				</div>
			</header>
			{#if routesLoading}
				<p class="muted">Loading routes…</p>
			{:else if routesError}
				<p class="error">Routes could not be loaded.</p>
			{:else if hasRoutes && routes}
				<div class="dashboard-map"><RoutesMap data={routes} /></div>
			{:else}
				<p class="muted">No routes available yet.</p>
			{/if}
		</section>

		<section class="domains-tile">
			<header class="domains-heading">
				<h2>Data domains</h2>
				<p>Expand your cockpit as more of your life arrives here.</p>
			</header>
			<div class="domain-grid">
				<DomainTile
					name="Activity"
					stat={summaryLoading || summaryError ? 'Loading activity count…' : `${activityCount.toLocaleString()} activities`}
					href="/activities"
					state="active"
				/>
				<DomainTile name="Sleep" stat="Recovery and sleep sessions" state="soon" />
				<DomainTile name="Media" stat="Reading and watching history" state="soon" />
			</div>
		</section>
	</div>
</section>

<style>
	.dashboard-shell {
		display: grid;
		gap: 1.25rem;
	}

	.dashboard-heading,
	.tile-heading {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 1rem;
	}

	.dashboard-heading h1,
	.tile-heading h2,
	.domains-heading h2 {
		margin: 0;
	}

	.dashboard-heading .muted,
	.tile-heading p,
	.domains-heading p {
		margin: 0.35rem 0 0;
	}

	.eyebrow {
		margin: 0 0 0.4rem;
		color: var(--accent);
		font-size: 0.72rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}

	.activity-link {
		flex: 0 0 auto;
		padding: 0.55rem 0.75rem;
		border: 1px solid var(--border);
		border-radius: var(--radius);
		background: var(--surface);
		color: var(--text);
		font-size: 0.86rem;
		text-decoration: none;
	}

	.activity-link:hover {
		border-color: var(--accent);
		text-decoration: none;
	}

	.stats-grid {
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		gap: 0.75rem;
	}

	.bento-grid {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 1rem;
	}

	.bento-grid :global(.heatmap) {
		grid-column: 1 / -1;
	}

	.status-tile,
	.recent-tile,
	.routes-tile {
		padding: 1rem;
	}

	.status-tile {
		grid-column: 1 / -1;
		min-height: 16rem;
		display: grid;
		place-items: center;
		color: var(--text-muted);
	}

	.tile-heading h2,
	.domains-heading h2 {
		font-size: 1rem;
	}

	.tile-heading p,
	.domains-heading p {
		color: var(--text-muted);
		font-size: 0.84rem;
	}

	.tile-heading a {
		font-size: 0.84rem;
		white-space: nowrap;
	}

	.recent-list {
		list-style: none;
		margin: 1rem 0 0;
		padding: 0;
	}

	.recent-list li + li {
		border-top: 1px solid var(--border);
	}

	.recent-row {
		display: grid;
		grid-template-columns: minmax(7rem, 0.8fr) minmax(0, 1.5fr) auto;
		gap: 0.35rem 0.75rem;
		padding: 0.8rem 0;
		color: var(--text);
		text-decoration: none;
	}

	.recent-row:hover {
		color: var(--accent);
		text-decoration: none;
	}

	.recent-title {
		min-width: 0;
		overflow: hidden;
		font-weight: 650;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.recent-metrics,
	.recent-date {
		grid-column: 2;
		color: var(--text-muted);
		font-size: 0.78rem;
	}

	.recent-date {
		grid-column: 3;
		grid-row: 1 / span 2;
		align-self: center;
		text-align: right;
	}

	.routes-tile {
		min-height: 21rem;
	}

	.dashboard-map {
		margin-top: 1rem;
	}

	.dashboard-map :global(.map) {
		height: 18rem;
	}

	.domains-tile {
		grid-column: 1 / -1;
	}

	.domains-heading {
		margin-bottom: 0.75rem;
	}

	.domain-grid {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 0.75rem;
	}

	@media (max-width: 800px) {
		.stats-grid {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}

		.bento-grid {
			grid-template-columns: 1fr;
		}
	}

	@media (max-width: 560px) {
		.dashboard-heading,
		.tile-heading {
			flex-direction: column;
		}

		.activity-link {
			width: 100%;
			text-align: center;
		}

		.recent-row {
			grid-template-columns: minmax(0, 1fr) auto;
		}

		.recent-title,
		.recent-metrics {
			grid-column: 1;
		}

		.recent-date {
			grid-column: 2;
		}

		.domain-grid {
			grid-template-columns: 1fr;
		}
	}
</style>
