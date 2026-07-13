<script lang="ts">
	import { onMount } from 'svelte';
	import { LineChart } from 'echarts/charts';
	import { GridComponent, TooltipComponent } from 'echarts/components';
	import { init, use } from 'echarts/core';
	import { CanvasRenderer } from 'echarts/renderers';
	import type { ECharts } from 'echarts/core';

	use([LineChart, GridComponent, TooltipComponent, CanvasRenderer]);

	let {
		xValues,
		xLabel,
		pace,
		heartRate,
		elevation,
		onHover
	}: {
		xValues: number[];
		xLabel: string;
		pace: (number | null)[];
		heartRate: (number | null)[];
		elevation: (number | null)[];
		onHover?: (index: number | null) => void;
	} = $props();

	let container: HTMLDivElement;
	let chart: ECharts | undefined;

	function render() {
		if (!chart) return;
		const styles = getComputedStyle(container);
		const text = styles.getPropertyValue('--text').trim();
		const muted = styles.getPropertyValue('--text-muted').trim() || '#9aa3b2';
		const border = styles.getPropertyValue('--border').trim() || '#2a2f3a';
		const surface = styles.getPropertyValue('--surface-2').trim();
		chart.setOption({
			animationDuration: 500,
			grid: { top: 34, right: 52, bottom: 52, left: 48 },
			tooltip: {
				trigger: 'axis',
				axisPointer: { type: 'line', lineStyle: { color: muted, width: 1 } },
				backgroundColor: surface,
				borderColor: border,
				textStyle: { color: text, fontSize: 11 },
				formatter: (params: Array<{ seriesName: string; value: [number, number | null]; color: string }>) => {
					const x = params[0]?.value?.[0];
					const rows = params
						.filter((item) => item.value?.[1] != null)
						.map((item) => `<span style="color:${item.color}">●</span> ${item.seriesName}: <strong>${formatValue(item.value[1], item.seriesName)}</strong>`);
					return `${xLabel}: ${formatXAxis(x)}<br/>${rows.join('<br/>')}`;
				}
			},
			xAxis: {
				type: 'value',
				name: xLabel,
				nameLocation: 'middle',
				nameGap: 30,
				axisLabel: { color: muted, fontSize: 10, formatter: (value: number) => formatXAxis(value) },
				axisLine: { lineStyle: { color: border } },
				splitLine: { lineStyle: { color: border, opacity: 0.55 } }
			},
			yAxis: [
				{ type: 'value', name: 'Pace', inverse: true, axisLabel: { color: '#4f8cff', fontSize: 10, formatter: formatPace }, axisLine: { lineStyle: { color: '#4f8cff' } }, splitLine: { lineStyle: { color: border, opacity: 0.35 } } },
				{ type: 'value', name: 'HR', position: 'right', axisLabel: { color: '#ff6b6b', fontSize: 10 }, axisLine: { lineStyle: { color: '#ff6b6b' } }, splitLine: { show: false } },
				{ type: 'value', name: 'Elev.', position: 'right', offset: 42, axisLabel: { color: '#3ecf8e', fontSize: 10 }, axisLine: { lineStyle: { color: '#3ecf8e' } }, splitLine: { show: false } }
			],
			series: [
				makeSeries('Pace', pace, '#4f8cff', 0),
				makeSeries('Heart rate', heartRate, '#ff6b6b', 1),
				makeSeries('Elevation', elevation, '#3ecf8e', 2)
			].filter((series) => series.data.some((point) => point[1] != null))
		});
		chart.off('updateAxisPointer');
		chart.on('updateAxisPointer', (rawEvent) => {
			const event = rawEvent as { axesInfo?: Array<{ value?: number }> };
			const x = event.axesInfo?.[0]?.value;
			if (x == null || xValues.length === 0) return;
			let nearest = 0;
			for (let i = 1; i < xValues.length; i++) if (Math.abs(xValues[i] - x) < Math.abs(xValues[nearest] - x)) nearest = i;
			onHover?.(nearest);
		});
		chart.on('globalout', () => onHover?.(null));
	}

	function makeSeries(name: string, values: (number | null)[], color: string, yAxisIndex: number) {
		return { name, type: 'line', yAxisIndex, showSymbol: false, smooth: 0.12, connectNulls: false, lineStyle: { color, width: 2 }, itemStyle: { color }, data: xValues.map((x, i) => [x, values[i] ?? null]) };
	}
	function formatXAxis(value: number | undefined) { return value == null ? '' : xLabel.includes('Distance') ? `${value.toFixed(value < 10 ? 1 : 0)} km` : `${value.toFixed(value < 10 ? 1 : 0)} min`; }
	function formatPace(value: number) { if (!Number.isFinite(value)) return ''; const min = Math.floor(value / 60); return `${min}:${String(Math.round(value % 60)).padStart(2, '0')}`; }
	function formatValue(value: number | null, name: string) { if (value == null) return '—'; return name === 'Pace' ? formatPace(value) : `${value.toFixed(0)}${name === 'Heart rate' ? ' bpm' : ' m'}`; }

	onMount(() => {
		chart = init(container, undefined, { renderer: 'canvas' });
		render();
		const resize = new ResizeObserver(() => chart?.resize());
		resize.observe(container);
		return () => { resize.disconnect(); chart?.dispose(); };
	});

	$effect(() => { xValues; xLabel; pace; heartRate; elevation; render(); });
</script>

<div class="chart" bind:this={container} aria-label="Synchronized activity chart"></div>

<style>
	.chart { width: 100%; height: 330px; min-height: 250px; }
</style>
