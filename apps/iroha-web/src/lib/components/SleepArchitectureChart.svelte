<script lang="ts">
	import { onMount } from 'svelte';
	import { init, use } from 'echarts/core';
	import { PieChart } from 'echarts/charts';
	import { LegendComponent, TooltipComponent } from 'echarts/components';
	import { CanvasRenderer } from 'echarts/renderers';
	import type { ECharts } from 'echarts/core';

	use([PieChart, LegendComponent, TooltipComponent, CanvasRenderer]);

	type Stage = { name: string; value: number; color: string };

	let {
		stages,
		selectedStage,
		onStageSelect,
		onStageHover
	}: {
		stages: Stage[];
		selectedStage: string;
		onStageSelect?: (stage: string) => void;
		onStageHover?: (stage: string | null) => void;
	} = $props();

	let container: HTMLDivElement;
	let chart: ECharts | undefined;

	function render() {
		if (!chart) return;
		const styles = getComputedStyle(container);
		chart.setOption({
			tooltip: {
				trigger: 'item',
				backgroundColor: styles.getPropertyValue('--surface-2').trim(),
				borderColor: styles.getPropertyValue('--border').trim(),
				textStyle: { color: styles.getPropertyValue('--text').trim() },
				formatter: (params: { name: string; value: number; percent: number }) =>
					`${params.name}<br/><strong>${formatMinutes(params.value)}</strong> · ${params.percent}%`
			},
			series: [
				{
					type: 'pie',
					radius: ['58%', '82%'],
					center: ['50%', '50%'],
					avoidLabelOverlap: true,
					itemStyle: { borderColor: styles.getPropertyValue('--surface').trim(), borderWidth: 3 },
					label: { show: false },
					emphasis: { scale: true, scaleSize: 5, label: { show: false } },
					data: stages.map((stage) => ({ name: stage.name, value: stage.value, itemStyle: { color: stage.color } }))
				}
			]
		});
		chart.dispatchAction({ type: 'highlight', seriesIndex: 0, name: selectedStage });
	}

	function formatMinutes(seconds: number): string {
		const minutes = Math.round(seconds / 60);
		return `${Math.floor(minutes / 60)}h ${minutes % 60}m`;
	}

	onMount(() => {
		chart = init(container, undefined, { renderer: 'canvas' });
		chart.on('click', (params) => onStageSelect?.(String(params.name)));
		chart.on('mouseover', (params) => onStageHover?.(String(params.name)));
		chart.on('mouseout', () => onStageHover?.(null));
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
		selectedStage;
		render();
	});
</script>

<div class="architecture-chart" bind:this={container} aria-label="Interactive sleep stage composition"></div>

<style>
	.architecture-chart { width: 11rem; height: 11rem; flex: 0 0 11rem; }
	@media (max-width: 520px) { .architecture-chart { width: 9rem; height: 9rem; flex-basis: 9rem; } }
</style>
