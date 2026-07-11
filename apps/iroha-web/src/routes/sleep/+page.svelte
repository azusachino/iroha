<script lang="ts">
	import { onMount } from 'svelte';
	import { ArrowLeft } from '@lucide/svelte';
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
	let sessionsLoading = $state(true);
	let loadingMore = $state(false);
	let cursor = $state<string | null>(null);
	let hasMore = $state(false);
	let loadMoreSentinel = $state<HTMLDivElement>();
	let nightListContainer = $state<HTMLDivElement>();
	let segmentsLoading = $state(false);
	let error = $state<string | null>(null);
	let yearBuckets = $state<SleepAggregateBucket[]>([]);
	let monthBuckets = $state<SleepAggregateBucket[]>([]);
	let aggregatesLoading = $state(true);
	let aggregatesError = $state<string | null>(null);
	let selectedYear = $state('all');
	let selectedMonth = $state('all');

	const mainSleep = $derived(sessions.filter((session) => session.is_main_sleep));
	const averageAsleep = $derived(
		mainSleep.length ? mainSleep.reduce((total, session) => total + session.asleep_s, 0) / mainSleep.length : 0
	);
	const averageEfficiency = $derived(
		mainSleep.length ? mainSleep.reduce((total, session) => total + session.efficiency, 0) / mainSleep.length : 0
	);
	const monthlyBuckets = $derived(monthBuckets.slice().reverse());
	const yearlyBuckets = $derived(yearBuckets.slice().reverse());
	const availableYears = $derived(yearBuckets.map((bucket) => String(new Date(bucket.period).getUTCFullYear())).reverse());
	const availableMonths = $derived(
		monthBuckets
			.filter((bucket) => selectedYear === 'all' || bucket.period.startsWith(`${selectedYear}-`))
			.map((bucket) => bucket.period.slice(0, 7))
			.reverse()
	);
	const visibleYears = $derived(
		selectedYear === 'all' ? yearlyBuckets : yearlyBuckets.filter((bucket) => bucket.period.startsWith(`${selectedYear}-`))
	);
	const visibleMonths = $derived(
		monthlyBuckets.filter((bucket) => {
			const period = bucket.period.slice(0, 7);
			return (selectedYear === 'all' || period.startsWith(`${selectedYear}-`)) && (selectedMonth === 'all' || period === selectedMonth);
		})
	);
	const focusedBucket = $derived(
		selectedMonth !== 'all'
			? monthBuckets.find((bucket) => bucket.period.startsWith(`${selectedMonth}-01`))
			: selectedYear !== 'all'
				? yearBuckets.find((bucket) => bucket.period.startsWith(`${selectedYear}-`))
				: null
	);
	const isPeriodFiltered = $derived(selectedYear !== 'all');
	const heroEyebrow = $derived(
		isPeriodFiltered
			? `Selected night · ${selectedMonth !== 'all' ? formatPeriod(`${selectedMonth}-01T00:00:00Z`, 'month') : selectedYear}`
			: `Last night · ${selected ? formatDateOnly(selected.wake_date) : ''}`
	);
	const nightsHeading = $derived(isPeriodFiltered ? 'Nights in selected period' : 'Recent nights');
	const monthMaxAsleep = $derived(Math.max(1, ...monthBuckets.map((bucket) => bucket.average_asleep_s)));
	const yearMaxSessions = $derived(Math.max(1, ...yearBuckets.map((bucket) => bucket.session_count)));

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

	function selectedRange(): { from?: string; to?: string } {
		if (selectedMonth !== 'all') {
			const [year, month] = selectedMonth.split('-').map(Number);
			const lastDay = new Date(Date.UTC(year, month, 0)).getUTCDate();
			return { from: `${selectedMonth}-01`, to: `${selectedMonth}-${String(lastDay).padStart(2, '0')}` };
		}
		if (selectedYear !== 'all') return { from: `${selectedYear}-01-01`, to: `${selectedYear}-12-31` };
		return {};
	}

	async function loadSessions(append = false) {
		if (append && (!hasMore || !cursor || loadingMore)) return;
		if (append) loadingMore = true;
		else {
			sessionsLoading = true;
			cursor = null;
			hasMore = false;
		}
		error = null;
		try {
			const page = await listSleep({ limit: PAGE_SIZE, cursor: append ? cursor ?? undefined : undefined, ...selectedRange() });
			sessions = append ? [...sessions, ...page.items] : page.items;
			cursor = page.next_cursor;
			hasMore = page.has_more;
			if (!append) {
				if (page.items[0]) await selectSession(page.items[0]);
				else {
					selected = null;
					segments = [];
				}
			}
		} catch (value) {
			error = errorMessage(value);
		} finally {
			sessionsLoading = false;
			loadingMore = false;
		}
	}

	function changeYear(value: string) {
		selectedYear = value;
		selectedMonth = 'all';
		void loadSessions(false);
	}

	function changeMonth(value: string) {
		selectedMonth = value;
		void loadSessions(false);
	}

	$effect(() => {
		if (!loadMoreSentinel || !nightListContainer || !hasMore) return;
		const observer = new IntersectionObserver(
			(entries) => {
				if (entries.some((entry) => entry.isIntersecting)) void loadSessions(true);
			},
			{ root: nightListContainer, rootMargin: '120px' }
		);
		observer.observe(loadMoreSentinel);
		return () => observer.disconnect();
	});

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

	function compositionStyle(session: SleepSession): string {
		const total = session.core_s + session.deep_s + session.rem_s + session.awake_s + session.unspecified_s;
		if (!total) return '';
		const core = (session.core_s / total) * 100;
		const deep = core + (session.deep_s / total) * 100;
		const rem = deep + (session.rem_s / total) * 100;
		return `background: conic-gradient(#5c8dff 0 ${core}%, #8870e8 ${core}% ${deep}%, #e879b4 ${deep}% ${rem}%, #d39a4c ${rem}% 100%);`;
	}

	onMount(() => {
		void loadSessions(false);
		void loadAggregates();
	});
</script>

<svelte:head>
	<title>Sleep · iroha</title>
</svelte:head>

<section class="sleep-shell">
	<header class="page-heading">
		<div>
			<p class="eyebrow">Sleep / personal history</p>
			<h1>How are you recovering?</h1>
			<p class="muted">A long-view of your nights, with the last one ready to inspect.</p>
		</div>
		<a class="back-link" href="/dashboard"><ArrowLeft size={15} /> <span>Dashboard</span></a>
	</header>

	{#if sessionsLoading && sessions.length === 0}
		<section class="status tile"><p>Loading your sleep history…</p></section>
	{:else if error && !selected}
		<section class="status tile"><p class="error">Sleep could not be loaded: {error}</p></section>
	{:else if !selected}
		<section class="status tile"><p class="muted">No sleep sessions imported yet.</p></section>
	{:else}
		<section class="hero tile">
			<div class="hero-orb"></div>
			<div class="hero-topline">
				<div>
				<p class="eyebrow">{heroEyebrow}</p>
					<h2>{formatDuration(selected.asleep_s)} <span>asleep</span></h2>
				</div>
				<div class="hero-status">
					<span class="status-pill">{selected.is_main_sleep ? 'Primary overnight sleep' : 'Short session'}</span>
					<button class="info-button" type="button" title="Primary overnight sleep means at least 3 hours asleep. Shorter sessions are treated as naps or fragments.">i</button>
				</div>
			</div>
			<p class="hero-copy">
				{#if selected.is_main_sleep}
					Your main overnight window was {formatDuration(selected.time_in_bed_s)} in bed, with {Math.round(selected.efficiency * 100)}% efficiency.
				{:else}
					This session is under three hours asleep, so iroha treats it as a nap or short fragment.
				{/if}
			</p>
			<div class="hero-metrics">
				<div><span>In bed</span><strong>{formatDuration(selected.time_in_bed_s)}</strong></div>
				<div><span>Efficiency</span><strong>{Math.round(selected.efficiency * 100)}%</strong></div>
				<div><span>Deep + REM</span><strong>{formatDuration(selected.deep_s + selected.rem_s)}</strong></div>
				<div><span>Source</span><strong>{selected.source || 'Apple Health'}</strong></div>
			</div>
		</section>

		<section class="insight-strip" aria-label="Recent sleep context">
			<StatTile label="Recent main sleep" value={mainSleep.length.toLocaleString()} sub="In the loaded recent window" />
			<StatTile label="Average asleep" value={formatDuration(averageAsleep)} sub="Recent main-sleep sessions" />
			<StatTile label="Average efficiency" value={`${Math.round(averageEfficiency * 100)}%`} sub="Asleep / time in bed" />
		</section>

		<section class="trend-panel tile">
			<header class="section-heading">
				<div><p class="eyebrow">The long view</p><h2>Sleep over time</h2></div>
				<div class="period-controls" aria-label="Sleep period filters">
					<select aria-label="Filter by year" value={selectedYear} onchange={(event) => changeYear(event.currentTarget.value)}>
						<option value="all">All years</option>
						{#each availableYears as year (year)}<option value={year}>{year}</option>{/each}
					</select>
					<select aria-label="Filter by month" value={selectedMonth} onchange={(event) => changeMonth(event.currentTarget.value)} disabled={selectedYear === 'all'}>
						<option value="all">All months</option>
						{#each availableMonths as month (month)}<option value={month}>{formatPeriod(`${month}-01T00:00:00Z`, 'month')}</option>{/each}
					</select>
				</div>
			</header>
			{#if aggregatesLoading}
				<p class="muted panel-loading">Building your history…</p>
			{:else if aggregatesError}
				<p class="error panel-loading">History could not be loaded: {aggregatesError}</p>
			{:else}
				<div class="year-trend">
					{#each visibleYears as bucket (bucket.period)}
						<div class="year-card">
							<div class="year-heading"><strong>{formatPeriod(bucket.period, 'year')}</strong><span>{bucket.session_count} nights</span></div>
							<div class="year-bar"><span style={`width: ${(bucket.session_count / yearMaxSessions) * 100}%`}></span></div>
							<div class="year-detail"><span>{formatDuration(bucket.average_asleep_s)} avg asleep</span><b>{Math.round(bucket.average_efficiency * 100)}%</b></div>
						</div>
					{/each}
				</div>
				{#if focusedBucket}
					<div class="focus-callout"><span>{selectedMonth !== 'all' ? formatPeriod(focusedBucket.period, 'month') : selectedYear}</span><strong>{focusedBucket.session_count} nights · {formatDuration(focusedBucket.average_asleep_s)} average asleep</strong><em>{focusedBucket.main_sleep_count} primary overnight sessions</em></div>
				{/if}
				<div class="trend-legend"><span><i class="dot dot-session"></i>Recorded nights</span><span>Average efficiency shown per year</span></div>
			{/if}
		</section>

		<div class="analysis-grid">
			<section class="stage-panel tile">
				<header class="section-heading"><div><p class="eyebrow">Last night</p><h2>Sleep architecture</h2></div></header>
				<div class="architecture">
					<div class="donut" style={compositionStyle(selected)}><div><strong>{formatDuration(selected.asleep_s)}</strong><span>asleep</span></div></div>
					<div class="stage-list">
						<div><i class="dot dot-core"></i><span>Core</span><b>{formatDuration(selected.core_s)}</b></div>
						<div><i class="dot dot-deep"></i><span>Deep</span><b>{formatDuration(selected.deep_s)}</b></div>
						<div><i class="dot dot-rem"></i><span>REM</span><b>{formatDuration(selected.rem_s)}</b></div>
						<div><i class="dot dot-awake"></i><span>Awake</span><b>{formatDuration(selected.awake_s)}</b></div>
					</div>
				</div>
				<p class="panel-footnote">Stage estimates are reconstructed from Apple Health records and are best read as patterns, not clinical measurements.</p>
			</section>

			<section class="month-panel tile">
				<header class="section-heading"><div><p class="eyebrow">The rhythm</p><h2>Monthly pattern</h2></div><span class="section-note">Average asleep</span></header>
				<div class="month-list">
					{#each visibleMonths.slice(0, 12) as bucket (bucket.period)}
						<div class="month-row"><span>{formatPeriod(bucket.period, 'month')}</span><div class="month-bar"><i style={`width: ${(bucket.average_asleep_s / monthMaxAsleep) * 100}%`}></i></div><b>{formatDuration(bucket.average_asleep_s)}</b></div>
					{/each}
				</div>
			</section>
		</div>

		<section class="night-panel tile">
			<header class="section-heading"><div><p class="eyebrow">Drill down</p><h2>{nightsHeading}</h2><p class="muted">Select a night to see the stage timeline.</p></div><span class="section-note">{sessions.length}{hasMore ? '+' : ''} nights in period</span></header>
			<div class="night-layout">
				<div bind:this={nightListContainer} class="night-list">
					{#each sessions as session (session.id)}
						<button class:selected={selected.id === session.id} class="night-row" type="button" onclick={() => selectSession(session)}>
							<span class="night-date">{formatDateOnly(session.wake_date)}</span><span>{formatDuration(session.asleep_s)}</span><b>{Math.round(session.efficiency * 100)}%</b>
						</button>
					{/each}
					<div bind:this={loadMoreSentinel} class="load-more-sentinel" aria-live="polite">
						{#if loadingMore}<span>Loading more nights…</span>{:else if hasMore}<span>Scroll for more nights</span>{/if}
					</div>
				</div>
				<div class="timeline-card">
					<div class="timeline-meta"><span>{formatDateOnly(selected.wake_date)}</span><span>{selected.is_main_sleep ? 'Primary overnight sleep' : 'Short session'}</span></div>
					{#if segmentsLoading}<p class="muted panel-loading">Loading stages…</p>{:else}<div class="timeline" aria-label="Sleep stage timeline">{#each segments as segment (segment.id)}<span class={`stage ${stageClass(segment.stage)}`} style={`width: ${segmentWidth(segment)}%`} title={`${segment.stage} ${formatDuration((new Date(segment.ended_at).getTime() - new Date(segment.started_at).getTime()) / 1000)}`}></span>{/each}</div>{/if}
					<div class="timeline-axis"><span>{new Date(selected.started_at).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' })}</span><span>{new Date(selected.ended_at).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' })}</span></div>
				</div>
			</div>
			{#if hasMore}
				<button class="load-more-button" type="button" disabled={loadingMore} onclick={() => loadSessions(true)}>
					{loadingMore ? 'Loading…' : 'Load more nights'}
				</button>
			{/if}
		</section>
	{/if}
</section>

<style>
	.sleep-shell { display: grid; gap: 1rem; }
	.page-heading, .section-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 1rem; }
	.page-heading h1, .section-heading h2 { margin: 0; }
	.page-heading .muted, .section-heading .muted { margin: 0.35rem 0 0; }
	.eyebrow { margin: 0 0 0.35rem; color: var(--accent); font-size: 0.68rem; font-weight: 750; letter-spacing: 0.11em; text-transform: uppercase; }
	.back-link, .section-note { color: var(--text-muted); font-size: 0.78rem; }
	.back-link { display: inline-flex; align-items: center; gap: 0.35rem; padding: 0.5rem 0.65rem; border: 1px solid var(--border); border-radius: 8px; background: var(--surface); text-decoration: none; }
	.back-link:hover { border-color: var(--accent); color: var(--text); text-decoration: none; }
	.period-controls { display: flex; gap: 0.45rem; }
	.period-controls select { min-height: 2rem; padding: 0.35rem 0.55rem; border: 1px solid var(--border); border-radius: 7px; background: var(--surface-2); color: var(--text); font: inherit; font-size: 0.76rem; }
	.period-controls select:disabled { opacity: 0.45; cursor: not-allowed; }
	.hero { position: relative; overflow: hidden; min-height: 16rem; padding: 1.5rem; background: radial-gradient(circle at 92% 8%, rgb(99 123 255 / 0.28), transparent 34%), radial-gradient(circle at 70% 100%, rgb(125 74 193 / 0.18), transparent 42%), var(--surface); }
	.hero-orb { position: absolute; width: 15rem; height: 15rem; right: -5rem; top: -7rem; border: 1px solid rgb(174 190 255 / 0.25); border-radius: 50%; box-shadow: 0 0 0 1.5rem rgb(120 134 255 / 0.04), 0 0 0 3rem rgb(120 134 255 / 0.025); }
	.hero-topline, .hero-metrics { position: relative; z-index: 1; }
	.hero-topline { display: flex; justify-content: space-between; gap: 1rem; }
	.hero h2 { margin: 0; font-size: clamp(2.8rem, 8vw, 5.4rem); line-height: 0.95; letter-spacing: -0.07em; }
	.hero h2 span { margin-left: 0.35rem; color: var(--text-muted); font-size: 0.28em; font-weight: 600; letter-spacing: -0.01em; }
	.hero-status { display: flex; align-items: center; gap: 0.4rem; padding-top: 0.25rem; }
	.status-pill { padding: 0.45rem 0.65rem; border: 1px solid rgb(128 163 255 / 0.38); border-radius: 99px; background: rgb(92 141 255 / 0.12); color: #b8caff; font-size: 0.75rem; font-weight: 700; }
	.info-button { width: 1.4rem; height: 1.4rem; border: 1px solid var(--border); border-radius: 50%; background: transparent; color: var(--text-muted); cursor: help; }
	.hero-copy { position: relative; max-width: 35rem; margin: 1.25rem 0 1.5rem; color: #c4ccdb; font-size: 0.92rem; line-height: 1.55; }
	.hero-metrics { display: grid; grid-template-columns: repeat(4, 1fr); gap: 1rem; padding-top: 1rem; border-top: 1px solid rgb(255 255 255 / 0.1); }
	.hero-metrics div { display: grid; gap: 0.25rem; }
	.hero-metrics span { color: var(--text-muted); font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.06em; }
	.hero-metrics strong { overflow: hidden; color: var(--text); font-size: 0.88rem; text-overflow: ellipsis; white-space: nowrap; }
	.insight-strip { display: grid; grid-template-columns: repeat(3, 1fr); gap: 0.75rem; }
	.trend-panel, .stage-panel, .month-panel, .night-panel { padding: 1.25rem; }
	.trend-panel { background: linear-gradient(135deg, rgb(91 128 255 / 0.07), transparent 42%), var(--tile-surface); }
	.panel-loading { min-height: 7rem; display: grid; place-items: center; }
	.year-trend { display: grid; grid-template-columns: repeat(auto-fit, minmax(8rem, 1fr)); gap: 0.75rem; margin-top: 1.5rem; }
	.year-card { padding: 0.8rem; border: 1px solid var(--border); border-radius: 10px; background: rgb(255 255 255 / 0.025); }
	.year-heading, .year-detail { display: flex; justify-content: space-between; gap: 0.5rem; font-size: 0.78rem; }
	.year-heading span, .year-detail span { color: var(--text-muted); }
	.year-bar { height: 0.48rem; margin: 1rem 0 0.6rem; overflow: hidden; border-radius: 99px; background: var(--surface-2); }
	.year-bar span { display: block; height: 100%; border-radius: inherit; background: linear-gradient(90deg, #5c8dff, #9c72ef); }
	.year-detail { font-size: 0.7rem; }
	.year-detail b { color: #b8caff; }
	.trend-legend { display: flex; gap: 1rem; margin-top: 1rem; color: var(--text-muted); font-size: 0.7rem; }
	.focus-callout { display: flex; align-items: baseline; gap: 0.75rem; margin-top: 1rem; padding: 0.75rem 0.9rem; border-left: 2px solid var(--accent); background: rgb(92 141 255 / 0.08); font-size: 0.78rem; }
	.focus-callout span { color: var(--accent); font-weight: 750; }
	.focus-callout strong { color: var(--text); }
	.focus-callout em { color: var(--text-muted); font-style: normal; }
	.analysis-grid { display: grid; grid-template-columns: minmax(0, 0.9fr) minmax(0, 1.1fr); gap: 1rem; }
	.architecture { display: flex; align-items: center; gap: 1.5rem; margin: 1.5rem 0 1rem; }
	.donut { display: grid; flex: 0 0 9rem; width: 9rem; height: 9rem; place-items: center; border-radius: 50%; }
	.donut > div { display: grid; width: 6.4rem; height: 6.4rem; place-content: center; border-radius: 50%; background: var(--surface); text-align: center; }
	.donut strong { font-size: 1.15rem; letter-spacing: -0.04em; }
	.donut span { color: var(--text-muted); font-size: 0.7rem; }
	.stage-list { display: grid; flex: 1; gap: 0.55rem; }
	.stage-list div { display: grid; grid-template-columns: 0.65rem 1fr auto; gap: 0.45rem; align-items: center; color: var(--text-muted); font-size: 0.78rem; }
	.stage-list b { color: var(--text); font-weight: 650; }
	.dot { display: inline-block; width: 0.55rem; height: 0.55rem; border-radius: 50%; }
	.dot-session, .dot-core, .stage-core { background: #5c8dff; }
	.dot-deep, .stage-deep { background: #8870e8; }
	.dot-rem, .stage-rem { background: #e879b4; }
	.dot-awake, .stage-awake { background: #d39a4c; }
	.panel-footnote { margin: 0; color: var(--text-muted); font-size: 0.72rem; line-height: 1.45; }
	.month-list { display: grid; gap: 0.75rem; margin-top: 1.5rem; }
	.month-row { display: grid; grid-template-columns: 5.5rem 1fr auto; gap: 0.65rem; align-items: center; color: var(--text-muted); font-size: 0.75rem; }
	.month-row b { color: var(--text); font-weight: 650; }
	.month-bar { height: 0.42rem; overflow: hidden; border-radius: 99px; background: var(--surface-2); }
	.month-bar i { display: block; height: 100%; border-radius: inherit; background: linear-gradient(90deg, #4d7fff, #c27de4); }
	.night-layout { display: grid; grid-template-columns: minmax(15rem, 0.75fr) minmax(0, 1.25fr); gap: 1.5rem; margin-top: 1.25rem; }
	.night-list { max-height: 19rem; overflow: auto; }
	.night-row { display: grid; grid-template-columns: 1fr auto auto; width: 100%; gap: 0.75rem; padding: 0.7rem 0; border: 0; border-top: 1px solid var(--border); background: transparent; color: var(--text-muted); text-align: left; cursor: pointer; font-size: 0.78rem; }
	.night-row:hover, .night-row.selected { color: var(--text); }
	.night-row.selected .night-date { color: var(--accent); }
	.night-row b { color: var(--text); font-weight: 650; }
	.timeline-card { align-self: center; padding: 1.25rem; border: 1px solid var(--border); border-radius: 10px; background: rgb(255 255 255 / 0.025); }
	.timeline-meta, .timeline-axis { display: flex; justify-content: space-between; gap: 1rem; color: var(--text-muted); font-size: 0.72rem; }
	.load-more-sentinel { min-height: 2rem; display: grid; place-items: center; color: var(--text-muted); font-size: 0.72rem; }
	.load-more-button { display: block; margin: 0.5rem auto 0; padding: 0.55rem 0.8rem; border: 1px solid var(--border); border-radius: 8px; background: var(--surface-2); color: var(--text); font: inherit; font-size: 0.78rem; cursor: pointer; }
	.load-more-button:hover { border-color: var(--accent); }
	.load-more-button:disabled { cursor: wait; opacity: 0.6; }
	.timeline { display: flex; min-height: 3.4rem; margin: 1rem 0 0.45rem; overflow: hidden; border-radius: 8px; background: var(--surface-2); }
	.stage { min-width: 0.15rem; }
	.stage-in_bed, .stage-asleep_unspecified { background: #788397; }
	@media (max-width: 760px) { .hero-topline, .page-heading, .section-heading { flex-direction: column; } .hero-status { padding-top: 0; } .hero-metrics, .insight-strip, .analysis-grid, .night-layout { grid-template-columns: 1fr 1fr; } .analysis-grid, .night-layout { grid-column: 1 / -1; } .hero-metrics { gap: 0.7rem; } .period-controls { width: 100%; } .period-controls select { flex: 1; } .focus-callout { align-items: flex-start; flex-direction: column; gap: 0.25rem; } }
	@media (max-width: 520px) { .hero-metrics, .insight-strip, .analysis-grid, .night-layout { grid-template-columns: 1fr; } .architecture { align-items: flex-start; } .donut { flex-basis: 7.5rem; width: 7.5rem; height: 7.5rem; } .donut > div { width: 5.4rem; height: 5.4rem; } }
</style>
