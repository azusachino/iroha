<script lang="ts">
	import { onMount } from 'svelte';
	import { BarChart } from 'echarts/charts';
	import { GridComponent, TooltipComponent } from 'echarts/components';
	import { init, use } from 'echarts/core';
	import { CanvasRenderer } from 'echarts/renderers';
	import type { ECharts } from 'echarts/core';

	use([BarChart, GridComponent, TooltipComponent, CanvasRenderer]);

	type Stage = { name: string; value: number; color: string };

	let { stages }: { stages: Stage[] } = $props();
	let container: HTMLDivElement;
	let chart: ECharts | undefined;

	function formatMinutes(seconds: number): string {
		const minutes = Math.round(seconds / 60);
		return `${Math.floor(minutes / 60)}h ${minutes % 60}m`;
	}

	function render() {
		if (!chart) return;
		const styles = getComputedStyle(container);
		chart.setOption({
			animationDuration: 450,
			animationDurationUpdate: 350,
			animationEasing: 'cubicOut',
			animationEasingUpdate: 'cubicInOut',
			grid: { top: 1, right: 0, bottom: 1, left: 0 },
			xAxis: { type: 'value', max: Math.max(1, stages.reduce((sum, stage) => sum + stage.value, 0)), show: false },
			yAxis: { type: 'category', data: ['sleep'], show: false },
			tooltip: {
				trigger: 'item',
				backgroundColor: styles.getPropertyValue('--surface-2').trim(),
				borderColor: styles.getPropertyValue('--border').trim(),
				textStyle: { color: styles.getPropertyValue('--text').trim(), fontSize: 11 },
				formatter: (params: { seriesName: string; value: number }) => `${params.seriesName}<br/><strong>${formatMinutes(params.value)}</strong>`
			},
			series: stages.map((stage) => ({
				name: stage.name,
				type: 'bar',
				stack: 'sleep',
				barWidth: 18,
				data: [stage.value],
				itemStyle: { color: stage.color, borderColor: styles.getPropertyValue('--surface').trim(), borderWidth: 1 },
				emphasis: { itemStyle: { shadowBlur: 8, shadowColor: stage.color } }
			}))
		});
	}

	onMount(() => {
		chart = init(container, undefined, { renderer: 'canvas' });
		render();
		const resize = new ResizeObserver(() => chart?.resize());
		resize.observe(container);
		return () => {
			resize.disconnect();
			chart?.dispose();
		};
	});

	$effect(() => {
		stages;
		render();
	});
</script>

<div class="night-bar" bind:this={container} aria-label="Sleep stage composition bar"></div>

<style>
	.night-bar { width: 6rem; height: 1.45rem; }
	@media (max-width: 520px) { .night-bar { width: 4rem; } }
</style>
