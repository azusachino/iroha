<script lang="ts">
	import { onMount } from 'svelte';
	import { LineChart } from 'echarts/charts';
	import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components';
	import { init, use } from 'echarts/core';
	import { CanvasRenderer } from 'echarts/renderers';
	import type { ECharts } from 'echarts/core';
	import type { SummaryBucket } from '$lib/api';
	import { formatDistance } from '$lib/format';

	use([LineChart, GridComponent, LegendComponent, TooltipComponent, CanvasRenderer]);

	const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

	let {
		byMonth,
		year,
		sportName
	}: { byMonth: SummaryBucket[]; year: string; sportName?: string } = $props();

	interface YearSeries {
		year: string;
		cumulative: (number | null)[];
		lastIdx: number;
	}

	let chartContainer = $state<HTMLDivElement>();
	let chart: ECharts | undefined;

	function seriesFor(y: string): YearSeries | null {
		const monthly = new Array(12).fill(0);
		let seen = false;
		let lastIdx = -1;
		for (const bucket of byMonth) {
			if (!bucket.key.startsWith(`${y}-`)) continue;
			const idx = Number(bucket.key.slice(5, 7)) - 1;
			if (idx < 0 || idx > 11) continue;
			monthly[idx] = bucket.distance_m;
			seen = true;
			if (idx > lastIdx) lastIdx = idx;
		}
		if (!seen) return null;
		let running = 0;
		return { year: y, cumulative: monthly.map((value, index) => { running += value; return index <= lastIdx ? running : null; }), lastIdx };
	}

	const current = $derived(seriesFor(year));
	const prior = $derived(seriesFor(String(Number(year) - 1)));
	const delta = $derived.by(() => {
		if (!current || !prior) return null;
		const idx = current.lastIdx;
		const cur = current.cumulative[idx];
		const previous = prior.cumulative[Math.min(idx, prior.lastIdx)];
		if (cur == null || previous == null) return null;
		const diff = cur - previous;
		return { diff, ahead: diff >= 0, month: MONTHS[idx] };
	});

	function render() {
		if (!chart || !chartContainer) return;
		const styles = getComputedStyle(chartContainer);
		const text = styles.getPropertyValue('--text').trim();
		const muted = styles.getPropertyValue('--text-muted').trim() || '#9aa3b2';
		const border = styles.getPropertyValue('--border').trim() || '#2a2f3a';
		const accent = styles.getPropertyValue('--sport-run').trim() || '#4f8cff';
		chart.setOption({
			animationDuration: 800,
			animationDurationUpdate: 550,
			animationEasing: 'cubicOut',
			animationEasingUpdate: 'cubicInOut',
			legend: {
				show: Boolean(prior),
				top: 0,
				right: 0,
				itemWidth: 12,
				itemHeight: 3,
				textStyle: { color: muted, fontSize: 11 },
				data: [year, ...(prior ? [String(Number(year) - 1)] : [])]
			},
			grid: { top: prior ? 34 : 12, right: 16, bottom: 30, left: 44 },
			tooltip: {
				trigger: 'axis',
				axisPointer: { type: 'line', lineStyle: { color: muted, opacity: 0.7 } },
				backgroundColor: styles.getPropertyValue('--surface-2').trim(),
				borderColor: border,
				textStyle: { color: text, fontSize: 11 },
				formatter: (params: Array<{ axisValue?: string; seriesName: string; value: number | null; color: string }>) => {
					const month = params[0]?.axisValue ?? '';
					const rows = params.map((item) => item.value == null ? '' : `<span style="color:${item.color}">●</span> ${item.seriesName}: <strong>${formatDistance(item.value)}</strong>`).filter(Boolean);
					return `${month}<br/>${rows.join('<br/>')}`;
				}
			},
			xAxis: { type: 'category', data: MONTHS, boundaryGap: false, axisLabel: { color: muted, fontSize: 10 }, axisLine: { lineStyle: { color: border } }, axisTick: { lineStyle: { color: border } } },
			yAxis: { type: 'value', min: 0, axisLabel: { color: muted, fontSize: 10, formatter: (value: number) => `${Math.round(value / 1000)} km` }, axisLine: { lineStyle: { color: border } }, splitLine: { lineStyle: { color: border, opacity: 0.6 } } },
			series: [
				...(prior ? [{ name: String(Number(year) - 1), type: 'line', data: prior.cumulative, showSymbol: false, connectNulls: false, smooth: 0.18, lineStyle: { color: muted, width: 1.5, type: 'dashed' }, itemStyle: { color: muted }, emphasis: { focus: 'series', lineStyle: { width: 2 } } }] : []),
				...(current ? [{ name: year, type: 'line', data: current.cumulative, showSymbol: false, connectNulls: false, smooth: 0.18, lineStyle: { color: accent, width: 2.5 }, itemStyle: { color: accent }, areaStyle: { color: accent, opacity: 0.15 }, emphasis: { focus: 'series', lineStyle: { width: 3 } } }] : [])
			]
		});
	}

	onMount(() => {
		if (!chartContainer) return;
		chart = init(chartContainer, undefined, { renderer: 'canvas' });
		render();
		const resize = new ResizeObserver(() => chart?.resize());
		resize.observe(chartContainer);
		return () => {
			resize.disconnect();
			chart?.dispose();
		};
	});

	$effect(() => {
		byMonth;
		year;
		current;
		prior;
		render();
	});
</script>

<div class="year-progress tile">
	<div class="header">
		<div class="title">Cumulative {sportName ? `${sportName.toLowerCase()} ` : ''}distance — {year}</div>
		{#if delta}
			<div class="delta" class:ahead={delta.ahead} class:behind={!delta.ahead}><span class="arrow">{delta.ahead ? '▲' : '▼'}</span>{formatDistance(Math.abs(delta.diff))}<span class="delta-sub">{delta.ahead ? 'ahead of' : 'behind'} {Number(year) - 1} at {delta.month}</span></div>
		{/if}
	</div>
	{#if !current}
		<p class="muted">No distance recorded for {year}.</p>
	{:else}
		<div class="chart" bind:this={chartContainer} role="img" aria-label={`Cumulative distance for ${year}`}></div>
	{/if}
</div>

<style>
	.year-progress { padding: 1rem; }
	.header { display: flex; align-items: baseline; justify-content: space-between; gap: 1rem; margin-bottom: 0.5rem; }
	.title { font-size: 0.8rem; color: var(--text-muted); }
	.delta { display: inline-flex; align-items: baseline; gap: 0.35rem; font-size: 0.9rem; font-weight: 700; }
	.delta.ahead { color: var(--sport-walk); }
	.delta.behind { color: var(--sport-other); }
	.delta .arrow { font-size: 0.7rem; }
	.delta-sub { font-size: 0.72rem; font-weight: 500; color: var(--text-muted); }
	.chart { width: 100%; height: 260px; }
	@media (max-width: 560px) { .header { align-items: flex-start; flex-direction: column; gap: 0.35rem; } .chart { height: 230px; } }
</style>
