<script lang="ts">
	import {
		getPublicSummary,
		listPublicActivities,
		type Summary,
		type SummaryBucket,
		type PublicActivity
	} from '$lib/api';
	import { formatDistance, formatDuration, formatDateOnly } from '$lib/format';

	const MONTH_LABELS = [
		'Jan',
		'Feb',
		'Mar',
		'Apr',
		'May',
		'Jun',
		'Jul',
		'Aug',
		'Sep',
		'Oct',
		'Nov',
		'Dec'
	];

	// --- Summary (totals / year tabs / month chart / sport breakdown) ---
	let summary = $state<Summary | null>(null);
	let summaryLoading = $state(true);
	let summaryError = $state<string | null>(null);
	let selectedYear = $state<string | null>(null);
	let sportFilter = $state<string | null>(null);

	async function loadSummary() {
		summaryLoading = true;
		summaryError = null;
		try {
			summary = await getPublicSummary();
			// by_year is sorted desc by the server — the first entry is the latest.
			if (summary.by_year.length > 0) selectedYear = summary.by_year[0].key;
		} catch (e) {
			summaryError = e instanceof Error ? e.message : String(e);
		} finally {
			summaryLoading = false;
		}
	}

	const years = $derived(summary?.by_year.map((y) => y.key) ?? []);

	// 12 slots (Jan..Dec) for the selected year, filled in from by_month where
	// present. Missing months render as zero-height bars.
	const monthSlots = $derived.by(() => {
		if (!summary || !selectedYear) return [];
		const byKey = new Map(summary.by_month.map((b) => [b.key, b]));
		return MONTH_LABELS.map((label, idx) => {
			const key = `${selectedYear}-${String(idx + 1).padStart(2, '0')}`;
			const bucket = byKey.get(key);
			return { label, key, bucket };
		});
	});

	// Bars are sized by distance; if a whole year has no distance data (e.g. an
	// all-strength-training year), fall back to activity count so bars aren't
	// all flat.
	const monthMetric = $derived.by(() => {
		const totalDistance = monthSlots.reduce((sum, m) => sum + (m.bucket?.distance_m ?? 0), 0);
		return totalDistance > 0 ? 'distance_m' : 'activity_count';
	});

	const monthMax = $derived(
		Math.max(1, ...monthSlots.map((m) => (m.bucket ? m.bucket[monthMetric] : 0)))
	);

	const sportMax = $derived(Math.max(1, ...(summary?.by_sport.map((s) => s.activity_count) ?? [])));

	function selectYear(year: string) {
		selectedYear = year;
	}

	function toggleSport(sport: string) {
		sportFilter = sportFilter === sport ? null : sport;
		activities = [];
		cursor = null;
		loadActivities(false);
	}

	// --- Activity table (paginated) ---
	let activities = $state<PublicActivity[]>([]);
	let activitiesLoading = $state(true);
	let activitiesLoadingMore = $state(false);
	let activitiesError = $state<string | null>(null);
	let cursor = $state<string | null>(null);
	let hasMore = $state(false);

	async function loadActivities(append: boolean) {
		if (append) activitiesLoadingMore = true;
		else activitiesLoading = true;
		activitiesError = null;
		try {
			const page = await listPublicActivities({
				limit: 20,
				sport_type: sportFilter ?? undefined,
				cursor: append && cursor ? cursor : undefined
			});
			activities = append ? [...activities, ...page.items] : page.items;
			cursor = page.next_cursor;
			hasMore = page.has_more;
		} catch (e) {
			activitiesError = e instanceof Error ? e.message : String(e);
			if (!append) activities = [];
		} finally {
			activitiesLoading = false;
			activitiesLoadingMore = false;
		}
	}

	// Initial loads (run once — no reactive dependencies read).
	$effect(() => {
		loadSummary();
		loadActivities(false);
	});
</script>

<h1>Public activity log</h1>

{#if summaryLoading}
	<p class="muted">Loading summary…</p>
{:else if summaryError}
	<p class="error">Failed to load summary: {summaryError}</p>
{:else if summary}
	<!-- Totals hero -->
	<div class="mb-8 grid grid-cols-1 gap-3 sm:grid-cols-3">
		<div class="rounded-lg border border-border bg-surface p-5 text-center">
			<div class="text-xs tracking-wide text-text-muted uppercase">Total distance</div>
			<div class="mt-1 text-3xl font-bold text-text">
				{formatDistance(summary.totals.distance_m)}
			</div>
		</div>
		<div class="rounded-lg border border-border bg-surface p-5 text-center">
			<div class="text-xs tracking-wide text-text-muted uppercase">Activities</div>
			<div class="mt-1 text-3xl font-bold text-text">{summary.totals.activity_count}</div>
		</div>
		<div class="rounded-lg border border-border bg-surface p-5 text-center">
			<div class="text-xs tracking-wide text-text-muted uppercase">Total time</div>
			<div class="mt-1 text-3xl font-bold text-text">
				{formatDuration(summary.totals.duration_s)}
			</div>
		</div>
	</div>

	<!-- Year tabs -->
	{#if years.length > 0}
		<div class="mb-4 flex flex-wrap gap-2">
			{#each years as year (year)}
				<button
					class="rounded-full border px-4 py-1 text-sm font-medium transition-colors"
					class:border-accent={selectedYear === year}
					class:text-text={selectedYear === year}
					class:bg-surface-2={selectedYear === year}
					class:border-border={selectedYear !== year}
					class:text-text-muted={selectedYear !== year}
					onclick={() => selectYear(year)}
				>
					{year}
				</button>
			{/each}
		</div>

		<!-- Per-month bar chart -->
		<div class="mb-8 rounded-lg border border-border bg-surface p-4">
			<div class="mb-3 text-sm text-text-muted">Monthly {monthMetric === 'distance_m' ? 'distance' : 'activities'} — {selectedYear}</div>
			<div class="flex h-40 items-end gap-2">
				{#each monthSlots as slot (slot.key)}
					{@const value = slot.bucket ? slot.bucket[monthMetric] : 0}
					{@const heightPct = Math.max(2, (value / monthMax) * 100)}
					<div class="group relative flex flex-1 flex-col items-center justify-end gap-1">
						<div
							class="pointer-events-none absolute -top-6 rounded border border-border bg-surface-2 px-1.5 py-0.5 text-[10px] whitespace-nowrap text-text opacity-0 transition-opacity group-hover:opacity-100"
						>
							{monthMetric === 'distance_m'
								? formatDistance(slot.bucket?.distance_m)
								: `${slot.bucket?.activity_count ?? 0} activities`}
						</div>
						<div
							class="w-full rounded-t bg-accent transition-all"
							style={`height: ${heightPct}%`}
						></div>
						<div class="text-[10px] text-text-muted">{slot.label}</div>
					</div>
				{/each}
			</div>
		</div>
	{/if}

	<!-- By-sport breakdown -->
	{#if summary.by_sport.length > 0}
		<div class="mb-8 rounded-lg border border-border bg-surface p-4">
			<div class="mb-3 text-sm text-text-muted">By sport</div>
			<div class="flex flex-col gap-2">
				{#each summary.by_sport as sport (sport.key)}
					<button
						class="flex items-center gap-3 rounded p-1 text-left transition-colors hover:bg-surface-2"
						onclick={() => toggleSport(sport.key)}
					>
						<span
							class="w-24 shrink-0 truncate text-sm capitalize"
							class:text-text={sportFilter === null || sportFilter === sport.key}
							class:text-text-muted={sportFilter !== null && sportFilter !== sport.key}
						>
							{sport.key}
						</span>
						<div class="h-2 flex-1 overflow-hidden rounded-full bg-surface-2">
							<div
								class="h-full rounded-full bg-accent"
								style={`width: ${Math.max(2, (sport.activity_count / sportMax) * 100)}%`}
							></div>
						</div>
						<span class="w-14 shrink-0 text-right text-xs text-text-muted"
							>{sport.activity_count}×</span
						>
						<span class="w-20 shrink-0 text-right text-xs text-text-muted"
							>{formatDistance(sport.distance_m)}</span
						>
					</button>
				{/each}
			</div>
			{#if sportFilter}
				<button class="mt-2 text-xs text-accent underline" onclick={() => toggleSport(sportFilter!)}>
					Clear sport filter ({sportFilter})
				</button>
			{/if}
		</div>
	{/if}
{/if}

<!-- Activity table -->
<h2>Activities</h2>
{#if activitiesLoading}
	<p class="muted">Loading activities…</p>
{:else if activitiesError}
	<p class="error">Failed to load activities: {activitiesError}</p>
{:else if activities.length === 0}
	<p class="muted">No activities found.</p>
{:else}
	<div class="overflow-x-auto rounded-lg border border-border">
		<table class="w-full border-collapse text-sm">
			<thead>
				<tr class="border-b border-border bg-surface-2 text-left text-text-muted">
					<th class="px-3 py-2 font-medium">Date</th>
					<th class="px-3 py-2 font-medium">Sport</th>
					<th class="px-3 py-2 font-medium">Distance</th>
					<th class="px-3 py-2 font-medium">Duration</th>
				</tr>
			</thead>
			<tbody>
				{#each activities as activity (activity.id)}
					<tr class="border-b border-border last:border-0 odd:bg-surface even:bg-surface/60">
						<td class="px-3 py-2 text-text"
							>{formatDateOnly(activity.started_at, activity.timezone)}</td
						>
						<td class="px-3 py-2">
							<span class="badge">{activity.sport_type}</span>
						</td>
						<td class="px-3 py-2 text-text">{formatDistance(activity.distance_m)}</td>
						<td class="px-3 py-2 text-text"
							>{formatDuration(activity.duration_s ?? activity.moving_time_s)}</td
						>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>

	{#if hasMore}
		<button class="load-more" onclick={() => loadActivities(true)} disabled={activitiesLoadingMore}>
			{activitiesLoadingMore ? 'Loading…' : 'Load more'}
		</button>
	{/if}
{/if}
