<script lang="ts">
	import { onMount } from 'svelte';
	import { LineChart } from 'echarts/charts';
	import { AxisPointerComponent, GridComponent, TooltipComponent } from 'echarts/components';
	import { init, use } from 'echarts/core';
	import { CanvasRenderer } from 'echarts/renderers';
	import type { ECharts } from 'echarts/core';

	use([LineChart, AxisPointerComponent, GridComponent, TooltipComponent, CanvasRenderer]);

	export interface SmallMultiple {
		label: string;
		values: number[];
		color: string;
		unit?: string;
	}

	let { labels, charts }: { labels: string[]; charts: SmallMultiple[] } = $props();
	let container: HTMLDivElement;
	let chart: ECharts | undefined;

	function render() {
		if (!chart) return;
		const styles = getComputedStyle(container);
		const text = styles.getPropertyValue('--text').trim();
		const muted = styles.getPropertyValue('--text-muted').trim() || '#9aa3b2';
		const border = styles.getPropertyValue('--border').trim() || '#2a2f3a';
		const count = Math.max(charts.length, 1);
		const columns = count > 2 ? 2 : count;
		const rows = Math.ceil(count / columns);
		const grid = charts.map((_, index) => ({ left: `${(index % columns) * (100 / columns) + 5}%`, top: `${Math.floor(index / columns) * (100 / rows) + 8}%`, width: `${100 / columns - 10}%`, height: `${100 / rows - 18}%` }));
		chart.setOption({
			animationDuration: 450,
			grid,
			axisPointer: { link: [{ xAxisIndex: 'all' }], snap: true },
			tooltip: { trigger: 'axis', axisPointer: { type: 'cross' }, backgroundColor: styles.getPropertyValue('--surface-2').trim(), borderColor: border, textStyle: { color: text, fontSize: 11 } },
			xAxis: charts.map((_, index) => ({ type: 'category', gridIndex: index, data: labels, boundaryGap: false, axisLabel: { show: index >= charts.length - columns, color: muted, fontSize: 9, interval: Math.max(0, Math.floor(labels.length / 4) - 1) }, axisLine: { lineStyle: { color: border } }, axisTick: { show: false }, splitLine: { show: false } })),
			yAxis: charts.map((item, index) => ({ type: 'value', gridIndex: index, name: item.unit ? `${item.label} (${item.unit})` : item.label, nameTextStyle: { color: item.color, fontSize: 10, fontWeight: 650 }, axisLabel: { color: muted, fontSize: 9 }, axisLine: { show: true, lineStyle: { color: item.color } }, splitLine: { lineStyle: { color: border, opacity: 0.35 } } })),
			series: charts.map((item, index) => ({ name: item.label, type: 'line', xAxisIndex: index, yAxisIndex: index, showSymbol: false, smooth: 0.18, lineStyle: { color: item.color, width: 2 }, itemStyle: { color: item.color }, areaStyle: { color: item.color, opacity: 0.08 }, data: item.values }))
		});
	}

	onMount(() => {
		chart = init(container, undefined, { renderer: 'canvas' });
		render();
		const resize = new ResizeObserver(() => chart?.resize());
		resize.observe(container);
		return () => { resize.disconnect(); chart?.dispose(); };
	});

	$effect(() => { labels; charts; render(); });
</script>

<div class="small-multiples" bind:this={container} aria-label="Daily trends with synchronized crosshair"></div>

<style>
	.small-multiples { width: 100%; height: 300px; min-height: 250px; }
</style>
