<script lang="ts">
	import type { SummaryBucket } from '$lib/api';
	import { formatDistance } from '$lib/format';

	// Cumulative distance-over-the-year curve: the selected year raced against
	// the previous year. Extends the per-year lens from discrete monthly bars to
	// a running total you can compare against where you were 12 months ago.
	let {
		byMonth,
		year,
		sportName
	}: { byMonth: SummaryBucket[]; year: string; sportName?: string } = $props();

	const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

	// Geometry (viewBox units; the SVG scales to its container via width:100%).
	const W = 800;
	const H = 260;
	const PAD_L = 46;
	const PAD_R = 18;
	const PAD_T = 26;
	const PAD_B = 30;
	const PLOT_W = W - PAD_L - PAD_R;
	const PLOT_H = H - PAD_T - PAD_B;

	interface YearSeries {
		year: string;
		// Cumulative meters at the end of each month; null past the last month
		// that actually has data (so a partial current year doesn't flatline to Dec).
		cumulative: (number | null)[];
		lastIdx: number; // last month index (0-11) with data
	}

	function seriesFor(y: string): YearSeries | null {
		const monthly = new Array(12).fill(0);
		let seen = false;
		let lastIdx = -1;
		for (const b of byMonth) {
			if (!b.key.startsWith(`${y}-`)) continue;
			const idx = Number(b.key.slice(5, 7)) - 1;
			if (idx < 0 || idx > 11) continue;
			monthly[idx] = b.distance_m;
			seen = true;
			if (idx > lastIdx) lastIdx = idx;
		}
		if (!seen) return null;
		let running = 0;
		const cumulative: (number | null)[] = monthly.map((m, i) => {
			running += m;
			return i <= lastIdx ? running : null;
		});
		return { year: y, cumulative, lastIdx };
	}

	const current = $derived(seriesFor(year));
	const prior = $derived(seriesFor(String(Number(year) - 1)));

	const maxY = $derived.by(() => {
		const vals: number[] = [];
		for (const s of [current, prior]) {
			if (!s) continue;
			for (const v of s.cumulative) if (v != null) vals.push(v);
		}
		return Math.max(1, ...vals);
	});

	function x(i: number): number {
		return PAD_L + (i / 11) * PLOT_W;
	}
	function y(v: number): number {
		return PAD_T + PLOT_H - (v / maxY) * PLOT_H;
	}

	// Build an SVG polyline `points` string for a cumulative series.
	function linePoints(s: YearSeries): string {
		const pts: string[] = [];
		for (let i = 0; i < 12; i++) {
			const v = s.cumulative[i];
			if (v == null) break;
			pts.push(`${x(i).toFixed(1)},${y(v).toFixed(1)}`);
		}
		return pts.join(' ');
	}

	// Area fill under the current-year line, closed to the baseline.
	const currentArea = $derived.by(() => {
		if (!current) return '';
		const pts: string[] = [];
		let lastX = PAD_L;
		for (let i = 0; i < 12; i++) {
			const v = current.cumulative[i];
			if (v == null) break;
			lastX = x(i);
			pts.push(`${i === 0 ? 'M' : 'L'}${x(i).toFixed(1)},${y(v).toFixed(1)}`);
		}
		if (pts.length === 0) return '';
		const baseY = (PAD_T + PLOT_H).toFixed(1);
		return `${pts.join(' ')} L${lastX.toFixed(1)},${baseY} L${PAD_L.toFixed(1)},${baseY} Z`;
	});

	// Y gridlines at 0/¼/½/¾/max.
	const yTicks = $derived([0, 0.25, 0.5, 0.75, 1].map((f) => f * maxY));

	// Delta: current year's total-to-date vs the prior year at the same month.
	const delta = $derived.by(() => {
		if (!current || !prior) return null;
		const idx = current.lastIdx;
		const cur = current.cumulative[idx];
		const prev = prior.cumulative[Math.min(idx, prior.lastIdx)];
		if (cur == null || prev == null) return null;
		const diff = cur - prev;
		return { diff, ahead: diff >= 0, month: MONTHS[idx] };
	});

	const endLabel = $derived.by(() => {
		if (!current) return null;
		const v = current.cumulative[current.lastIdx];
		if (v == null) return null;
		return { x: x(current.lastIdx), y: y(v), value: v };
	});
</script>

<div class="year-progress tile">
	<div class="header">
		<div class="title">Cumulative {sportName ? `${sportName.toLowerCase()} ` : ''}distance — {year}</div>
		{#if delta}
			<div class="delta" class:ahead={delta.ahead} class:behind={!delta.ahead}>
				<span class="arrow">{delta.ahead ? '▲' : '▼'}</span>
				{formatDistance(Math.abs(delta.diff))}
				<span class="delta-sub">{delta.ahead ? 'ahead of' : 'behind'} {Number(year) - 1} at {delta.month}</span>
			</div>
		{/if}
	</div>

	{#if !current}
		<p class="muted">No distance recorded for {year}.</p>
	{:else}
		<svg viewBox={`0 0 ${W} ${H}`} class="chart" role="img" aria-label={`Cumulative distance for ${year}`}>
			<defs>
				<linearGradient id="yp-fill" x1="0" y1="0" x2="0" y2="1">
					<stop offset="0%" stop-color="var(--sport-run)" stop-opacity="0.28" />
					<stop offset="100%" stop-color="var(--sport-run)" stop-opacity="0" />
				</linearGradient>
			</defs>

			<!-- Y grid + labels -->
			{#each yTicks as t}
				<line x1={PAD_L} x2={W - PAD_R} y1={y(t)} y2={y(t)} class="grid" />
				<text x={PAD_L - 8} y={y(t) + 3} class="axis-label" text-anchor="end">{Math.round(t / 1000)}</text>
			{/each}
			<text x={PAD_L - 8} y={PAD_T - 10} class="axis-unit" text-anchor="end">km</text>

			<!-- X labels -->
			{#each MONTHS as m, i}
				<text x={x(i)} y={H - 10} class="axis-label" text-anchor="middle">{m}</text>
			{/each}

			<!-- Prior year: muted dashed reference -->
			{#if prior}
				<polyline points={linePoints(prior)} class="prior-line" fill="none" />
			{/if}

			<!-- Current year: filled area + line -->
			<path d={currentArea} fill="url(#yp-fill)" stroke="none" />
			<polyline points={linePoints(current)} class="current-line" fill="none" />

			{#if endLabel}
				<circle cx={endLabel.x} cy={endLabel.y} r="4" class="end-dot" />
			{/if}
		</svg>

		<div class="legend">
			<span class="key"><span class="swatch current"></span>{year}</span>
			{#if prior}
				<span class="key"><span class="swatch prior"></span>{Number(year) - 1}</span>
			{/if}
		</div>
	{/if}
</div>

<style>
	.year-progress {
		padding: 1rem;
	}
	.header {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		gap: 1rem;
		margin-bottom: 0.5rem;
	}
	.title {
		font-size: 0.8rem;
		color: var(--text-muted);
	}
	.delta {
		display: inline-flex;
		align-items: baseline;
		gap: 0.35rem;
		font-size: 0.9rem;
		font-weight: 700;
	}
	.delta.ahead {
		color: var(--sport-walk);
	}
	.delta.behind {
		color: var(--sport-other);
	}
	.delta .arrow {
		font-size: 0.7rem;
	}
	.delta-sub {
		font-size: 0.72rem;
		font-weight: 500;
		color: var(--text-muted);
	}
	.chart {
		display: block;
		width: 100%;
		height: auto;
	}
	.grid {
		stroke: var(--border);
		stroke-width: 1;
	}
	.axis-label {
		fill: var(--text-muted);
		font-size: 11px;
	}
	.axis-unit {
		fill: var(--text-muted);
		font-size: 10px;
	}
	.prior-line {
		stroke: var(--text-muted);
		stroke-width: 1.5;
		stroke-dasharray: 4 4;
		opacity: 0.6;
	}
	.current-line {
		stroke: var(--sport-run);
		stroke-width: 2.5;
		stroke-linejoin: round;
		stroke-linecap: round;
	}
	.end-dot {
		fill: var(--sport-run);
		stroke: var(--surface);
		stroke-width: 2;
	}
	.legend {
		display: flex;
		gap: 1rem;
		margin-top: 0.5rem;
		font-size: 0.76rem;
		color: var(--text-muted);
	}
	.key {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
	}
	.swatch {
		display: inline-block;
		width: 0.85rem;
		height: 0.2rem;
		border-radius: 1px;
	}
	.swatch.current {
		background: var(--sport-run);
	}
	.swatch.prior {
		background: var(--text-muted);
		opacity: 0.6;
	}
</style>
