<script lang="ts">
	import { onMount } from 'svelte';
	import { BarChart } from 'echarts/charts';
	import { GridComponent, TooltipComponent } from 'echarts/components';
	import { init, use } from 'echarts/core';
	import { CanvasRenderer } from 'echarts/renderers';
	import type { ECharts } from 'echarts/core';

	use([BarChart, GridComponent, TooltipComponent, CanvasRenderer]);

	type Segment = { stage: string; started_at: string; ended_at: string };

	let { segments }: { segments: Segment[] } = $props();
	let container: HTMLDivElement;
	let chart: ECharts | undefined;

	const colors: Record<string, string> = {
		core: '#5c8dff',
		deep: '#8870e8',
		rem: '#e879b4',
		awake: '#d39a4c',
		in_bed: '#788397',
		asleep_unspecified: '#788397'
	};

	function duration(segment: Segment): number {
		return Math.max(0, (new Date(segment.ended_at).getTime() - new Date(segment.started_at).getTime()) / 1000);
	}

	function formatDuration(seconds: number): string {
		const minutes = Math.round(seconds / 60);
		return `${Math.floor(minutes / 60)}h ${minutes % 60}m`;
	}

	function render() {
		if (!chart) return;
		const styles = getComputedStyle(container);
		const total = segments.reduce((sum, segment) => sum + duration(segment), 0);
		chart.setOption({
			animationDuration: 550,
			animationDurationUpdate: 450,
			animationEasing: 'cubicOut',
			animationEasingUpdate: 'cubicInOut',
			grid: { top: 12, right: 4, bottom: 12, left: 4 },
			xAxis: { type: 'value', max: Math.max(1, total), show: false },
			yAxis: { type: 'category', data: ['Sleep stages'], show: false },
			tooltip: {
				trigger: 'item',
				backgroundColor: styles.getPropertyValue('--surface-2').trim(),
				borderColor: styles.getPropertyValue('--border').trim(),
				textStyle: { color: styles.getPropertyValue('--text').trim(), fontSize: 12 },
				formatter: (params: { seriesName: string; value: number }) => `${params.seriesName}<br/><strong>${formatDuration(params.value)}</strong>`
			},
			series: segments.map((segment) => ({
				name: segment.stage,
				type: 'bar',
				stack: 'sleep',
				barWidth: 42,
				data: [duration(segment)],
				itemStyle: {
					color: colors[segment.stage] ?? colors.asleep_unspecified,
					borderColor: styles.getPropertyValue('--surface').trim(),
					borderWidth: 1
				},
				emphasis: { itemStyle: { shadowBlur: 14, shadowColor: colors[segment.stage] ?? colors.asleep_unspecified } }
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
		segments;
		render();
	});
</script>

<div class="timeline-chart" bind:this={container} aria-label="Interactive sleep stage timeline"></div>

<style>
	.timeline-chart { width: 100%; height: 5.5rem; }
</style>
