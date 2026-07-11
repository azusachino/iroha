<script lang="ts">
	import { onMount } from 'svelte';
	import {
		getSleepSegments,
		listSleep,
		listSleepAggregates,
		type SleepAggregateBucket,
		type SleepSegment,
		type SleepSession
	} from '$lib/api';
	import StatTile from '$lib/components/StatTile.svelte';
	import { formatDateOnly, formatDuration } from '$lib/format';

	const PAGE_SIZE = 24;
	let sessions = $state<SleepSession[]>([]);
	let selected = $state<SleepSession | null>(null);
	let segments = $state<SleepSegment[]>([]);
	let loading = $state(true);
	let segmentsLoading = $state(false);
	let error = $state<string | null>(null);
	let yearBuckets = $state<SleepAggregateBucket[]>([]);
	let monthBuckets = $state<SleepAggregateBucket[]>([]);
	let aggregatesLoading = $state(true);
	let aggregatesError = $state<string | null>(null);

	const mainSleep = $derived(sessions.filter((session) => session.is_main_sleep));
	const averageAsleep = $derived(
		mainSleep.length ? mainSleep.reduce((total, session) => total + session.asleep_s, 0) / mainSleep.length : 0
	);
	const averageEfficiency = $derived(
		mainSleep.length ? mainSleep.reduce((total, session) => total + session.efficiency, 0) / mainSleep.length : 0
	);
	const yearMaxSessions = $derived(Math.max(1, ...yearBuckets.map((bucket) => bucket.session_count)));
	const monthTotal = $derived(monthBuckets.reduce((total, bucket) => total + bucket.session_count, 0));

	function errorMessage(value: unknown): string {
		return value instanceof Error ? value.message : String(value);
	}

	async function selectSession(session: SleepSession) {
		selected = session;
		segmentsLoading = true;
		try {
			segments = await getSleepSegments(session.id);
		} catch (value) {
			error = errorMessage(value);
			segments = [];
		} finally {
			segmentsLoading = false;
		}
	}

	async function load() {
		loading = true;
		error = null;
		try {
			const page = await listSleep({ limit: PAGE_SIZE });
			sessions = page.items;
			if (page.items[0]) await selectSession(page.items[0]);
		} catch (value) {
			error = errorMessage(value);
		} finally {
			loading = false;
		}
	}

	async function loadAggregates() {
		aggregatesLoading = true;
		aggregatesError = null;
		try {
			const [years, months] = await Promise.all([
				listSleepAggregates('year'),
				listSleepAggregates('month')
			]);
			yearBuckets = years.buckets;
			monthBuckets = months.buckets;
		} catch (value) {
			aggregatesError = errorMessage(value);
		} finally {
			aggregatesLoading = false;
		}
	}

	function formatPeriod(period: string, granularity: 'month' | 'year'): string {
		const date = new Date(period);
		if (granularity === 'year') return String(date.getUTCFullYear());
		return new Intl.DateTimeFormat('en', { month: 'short', year: 'numeric', timeZone: 'UTC' }).format(date);
	}

	function stageClass(stage: string): string {
		return `stage-${stage}`;
	}

	function segmentWidth(segment: SleepSegment): number {
		if (!selected) return 0;
		const total = new Date(selected.ended_at).getTime() - new Date(selected.started_at).getTime();
		const duration = new Date(segment.ended_at).getTime() - new Date(segment.started_at).getTime();
		return total > 0 ? Math.max(0.4, (duration / total) * 100) : 0;
	}

	onMount(() => {
		void load();
		void loadAggregates();
	});
</script>

<svelte:head>
	<title>Sleep · iroha</title>
</svelte:head>

<section class="sleep-shell">
	<header class="page-heading">
		<div>
			<p class="eyebrow">Sleep domain</p>
			<h1>Recovery and sleep</h1>
			<p class="muted">Nightly sessions reconstructed from Apple Health stage records.</p>
		</div>
		<a class="back-link" href="/dashboard">Back to dashboard</a>
	</header>

	<div class="stats-grid" aria-label="Sleep totals">
		<StatTile label="Sessions" value={loading ? '—' : sessions.length.toLocaleString()} sub="Most recent page" />
		<StatTile label="Main sleep" value={loading ? '—' : mainSleep.length.toLocaleString()} sub="At least 3 hours asleep" />
		<StatTile label="Avg asleep" value={loading ? '—' : formatDuration(averageAsleep)} sub="Main sleep sessions" />
		<StatTile label="Efficiency" value={loading ? '—' : `${Math.round(averageEfficiency * 100)}%`} sub="Asleep / time in bed" />
	</div>

	<section class="aggregate-panel tile" aria-label="Sleep history aggregates">
		<header class="tile-heading">
			<div>
				<h2>History at a glance</h2>
				<p>Full-history yearly and monthly sleep trends.</p>
			</div>
			<span class="source">{monthTotal.toLocaleString()} monthly sessions</span>
		</header>
		{#if aggregatesLoading}
			<p class="muted aggregate-status">Loading history aggregates…</p>
		{:else if aggregatesError}
			<p class="error aggregate-status">Aggregates could not be loaded: {aggregatesError}</p>
		{:else}
			<div class="aggregate-grid">
				<div class="yearly-panel">
					<h3>By year</h3>
					{#each yearBuckets as bucket (bucket.period)}
						<div class="year-row">
							<div class="aggregate-label"><strong>{formatPeriod(bucket.period, 'year')}</strong><span>{bucket.session_count} nights · {bucket.main_sleep_count} main</span></div>
							<div class="bar-track"><span class="bar-fill" style={`width: ${(bucket.session_count / yearMaxSessions) * 100}%`}></span></div>
							<div class="aggregate-metric">{formatDuration(bucket.average_asleep_s)} avg asleep · {Math.round(bucket.average_efficiency * 100)}%</div>
						</div>
					{/each}
				</div>
				<div class="monthly-panel">
					<h3>By month</h3>
					<div class="month-table-wrap">
						<table>
							<thead><tr><th>Month</th><th>Nights</th><th>Asleep</th><th>Efficiency</th><th>Stages</th></tr></thead>
							<tbody>
								{#each monthBuckets as bucket (bucket.period)}
									<tr>
										<td>{formatPeriod(bucket.period, 'month')}</td>
										<td>{bucket.session_count} <span class="muted">({bucket.main_sleep_count})</span></td>
										<td>{formatDuration(bucket.average_asleep_s)}</td>
										<td>{Math.round(bucket.average_efficiency * 100)}%</td>
										<td class="stage-total">{formatDuration(bucket.core_s + bucket.deep_s + bucket.rem_s)} asleep stages</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				</div>
			</div>
		{/if}
	</section>

	{#if loading}
		<section class="status tile"><p>Loading sleep sessions…</p></section>
	{:else if error && sessions.length === 0}
		<section class="status tile"><p class="error">Sleep could not be loaded: {error}</p></section>
	{:else if sessions.length === 0}
		<section class="status tile"><p class="muted">No sleep sessions imported yet.</p></section>
	{:else}
		<div class="sleep-grid">
			<section class="session-list tile" aria-label="Sleep sessions">
				<header class="tile-heading">
					<div>
						<h2>Recent nights</h2>
						<p>Choose a night to inspect its stage timeline.</p>
					</div>
				</header>
				<div class="sessions">
					{#each sessions as session (session.id)}
						<button class:selected={selected?.id === session.id} class="session-row" type="button" onclick={() => selectSession(session)}>
							<span class="session-date">{formatDateOnly(session.wake_date)}</span>
							<span class="session-duration">{formatDuration(session.asleep_s)}</span>
							<span class="session-efficiency">{Math.round(session.efficiency * 100)}%</span>
							<span class:main={session.is_main_sleep} class="session-kind">{session.is_main_sleep ? 'Main sleep' : 'Short'}</span>
						</button>
					{/each}
				</div>
			</section>

			<section class="detail tile">
				{#if selected}
					<header class="tile-heading">
						<div>
							<h2>{formatDateOnly(selected.wake_date)}</h2>
							<p>{formatDuration(selected.time_in_bed_s)} in bed · {Math.round(selected.efficiency * 100)}% efficiency</p>
						</div>
						<span class="source">{selected.source || 'Apple Health'}</span>
					</header>
					<div class="stage-summary">
						<span><i class="swatch core"></i>Core {formatDuration(selected.core_s)}</span>
						<span><i class="swatch deep"></i>Deep {formatDuration(selected.deep_s)}</span>
						<span><i class="swatch rem"></i>REM {formatDuration(selected.rem_s)}</span>
						<span><i class="swatch awake"></i>Awake {formatDuration(selected.awake_s)}</span>
					</div>
					{#if segmentsLoading}
						<p class="muted timeline-status">Loading stage timeline…</p>
					{:else}
						<div class="timeline" aria-label="Sleep stage timeline">
							{#each segments as segment (segment.id)}
								<span class={`stage ${stageClass(segment.stage)}`} style={`width: ${segmentWidth(segment)}%`} title={`${segment.stage} ${formatDuration((new Date(segment.ended_at).getTime() - new Date(segment.started_at).getTime()) / 1000)}`}></span>
							{/each}
						</div>
					{/if}
				{:else}
					<p class="muted">Select a sleep session.</p>
				{/if}
			</section>
		</div>
	{/if}
</section>

<style>
	.sleep-shell { display: grid; gap: 1.25rem; }
	.page-heading, .tile-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 1rem; }
	.page-heading h1, .tile-heading h2 { margin: 0; }
	.page-heading .muted, .tile-heading p { margin: 0.35rem 0 0; }
	.eyebrow { margin: 0 0 0.4rem; color: var(--accent); font-size: 0.72rem; font-weight: 700; letter-spacing: 0.08em; text-transform: uppercase; }
	.back-link { color: var(--text-muted); font-size: 0.86rem; }
	.stats-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 0.75rem; }
	.sleep-grid { display: grid; grid-template-columns: minmax(16rem, 0.8fr) minmax(0, 1.4fr); gap: 1rem; }
	.session-list, .detail { padding: 1rem; }
	.status { min-height: 14rem; display: grid; place-items: center; }
	.aggregate-panel { padding: 1rem; }
	.aggregate-status { min-height: 8rem; display: grid; place-items: center; }
	.aggregate-grid { display: grid; grid-template-columns: minmax(0, 0.8fr) minmax(0, 1.2fr); gap: 2rem; margin-top: 1.5rem; }
	.aggregate-grid h3 { margin: 0 0 0.8rem; font-size: 0.9rem; }
	.year-row + .year-row { margin-top: 1rem; }
	.aggregate-label { display: flex; justify-content: space-between; gap: 1rem; font-size: 0.84rem; }
	.aggregate-label span, .aggregate-metric { color: var(--text-muted); font-size: 0.76rem; }
	.bar-track { height: 0.55rem; margin: 0.35rem 0; overflow: hidden; border-radius: 99px; background: var(--surface-muted); }
	.bar-fill { display: block; height: 100%; border-radius: inherit; background: var(--accent); }
	.month-table-wrap { max-height: 19rem; overflow: auto; }
	table { width: 100%; border-collapse: collapse; font-size: 0.78rem; }
	th, td { padding: 0.5rem 0.35rem; border-bottom: 1px solid var(--border); text-align: left; white-space: nowrap; }
	th { position: sticky; top: 0; background: var(--surface); color: var(--text-muted); font-size: 0.7rem; font-weight: 700; text-transform: uppercase; }
	.stage-total { color: var(--text-muted); }
	.sessions { margin-top: 1rem; }
	.session-row { width: 100%; display: grid; grid-template-columns: 1fr auto auto auto; gap: 0.7rem; align-items: center; padding: 0.75rem 0; border: 0; border-top: 1px solid var(--border); background: transparent; color: var(--text); text-align: left; cursor: pointer; }
	.session-row:hover, .session-row.selected { color: var(--accent); }
	.session-date { font-weight: 650; }
	.session-duration, .session-efficiency, .session-kind { color: var(--text-muted); font-size: 0.82rem; }
	.session-kind.main { color: var(--accent); }
	.source { color: var(--text-muted); font-size: 0.78rem; }
	.stage-summary { display: flex; flex-wrap: wrap; gap: 0.6rem 1rem; margin: 2rem 0 1rem; color: var(--text-muted); font-size: 0.82rem; }
	.swatch { display: inline-block; width: 0.55rem; height: 0.55rem; margin-right: 0.3rem; border-radius: 50%; }
	.core, .stage-core { background: #3987e5; }
	.deep, .stage-deep { background: #6551b8; }
	.rem, .stage-rem { background: #d95926; }
	.awake, .stage-awake { background: #c98500; }
	.timeline { display: flex; min-height: 3rem; overflow: hidden; border-radius: var(--radius); background: var(--surface-muted); }
	.stage { min-width: 0.15rem; }
	.stage-in_bed, .stage-asleep_unspecified { background: #7b8794; }
	.timeline-status { min-height: 3rem; display: grid; place-items: center; }
	@media (max-width: 760px) { .stats-grid, .sleep-grid, .aggregate-grid { grid-template-columns: 1fr 1fr; } .sleep-grid, .aggregate-grid { grid-column: 1 / -1; } .session-row { grid-template-columns: 1fr auto; } .session-efficiency, .session-kind { justify-self: end; } }
	@media (max-width: 520px) { .stats-grid { grid-template-columns: 1fr 1fr; } .page-heading { flex-direction: column; } }
</style>
