<script lang="ts">
	import { onMount } from 'svelte';
	import { getSleepSegments, listSleep, type SleepSegment, type SleepSession } from '$lib/api';
	import StatTile from '$lib/components/StatTile.svelte';
	import { formatDateOnly, formatDuration } from '$lib/format';

	const PAGE_SIZE = 24;
	let sessions = $state<SleepSession[]>([]);
	let selected = $state<SleepSession | null>(null);
	let segments = $state<SleepSegment[]>([]);
	let loading = $state(true);
	let segmentsLoading = $state(false);
	let error = $state<string | null>(null);

	const mainSleep = $derived(sessions.filter((session) => session.is_main_sleep));
	const averageAsleep = $derived(
		mainSleep.length ? mainSleep.reduce((total, session) => total + session.asleep_s, 0) / mainSleep.length : 0
	);
	const averageEfficiency = $derived(
		mainSleep.length ? mainSleep.reduce((total, session) => total + session.efficiency, 0) / mainSleep.length : 0
	);

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
	@media (max-width: 760px) { .stats-grid, .sleep-grid { grid-template-columns: 1fr 1fr; } .sleep-grid { grid-column: 1 / -1; } .session-row { grid-template-columns: 1fr auto; } .session-efficiency, .session-kind { justify-self: end; } }
	@media (max-width: 520px) { .stats-grid { grid-template-columns: 1fr 1fr; } .page-heading { flex-direction: column; } }
</style>
