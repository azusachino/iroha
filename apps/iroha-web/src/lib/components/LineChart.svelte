<script lang="ts">
	import { onMount } from 'svelte';
	import uPlot from 'uplot';
	import 'uplot/dist/uPlot.min.css';

	export interface ChartSeries {
		label: string;
		values: (number | null)[];
		stroke: string;
	}

	let {
		title,
		xValues,
		xLabel,
		series
	}: {
		title: string;
		xValues: number[];
		xLabel: string;
		series: ChartSeries[];
	} = $props();

	let container: HTMLDivElement;

	onMount(() => {
		const data: uPlot.AlignedData = [xValues, ...series.map((s) => s.values)];

		const opts: uPlot.Options = {
			title,
			width: container.clientWidth || 600,
			height: 220,
			cursor: { drag: { x: true, y: false } },
			scales: { x: { time: false } },
			axes: [
				{ label: xLabel, stroke: '#9aa3b2', grid: { stroke: '#2a2f3a' }, ticks: { stroke: '#2a2f3a' } },
				{ stroke: '#9aa3b2', grid: { stroke: '#2a2f3a' }, ticks: { stroke: '#2a2f3a' } }
			],
			series: [
				{ label: xLabel },
				...series.map((s) => ({
					label: s.label,
					stroke: s.stroke,
					width: 1.5,
					spanGaps: false
				}))
			]
		};

		const plot = new uPlot(opts, data, container);

		const ro = new ResizeObserver(() => {
			plot.setSize({ width: container.clientWidth || 600, height: 220 });
		});
		ro.observe(container);

		return () => {
			ro.disconnect();
			plot.destroy();
		};
	});
</script>

<div class="chart">
	<div bind:this={container}></div>
</div>
