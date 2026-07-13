<script lang="ts">
	// A bare change-over-time mark: no axes, no ticks, one hue. Per the dataviz
	// skill a sparkline is the minimal line form; it carries a numeric value
	// beside it (in the caller) so identity/quantity never rests on color alone.
	// Data marks use the MARK tier (in-band), never the neon tier.
	let {
		values,
		color = 'var(--mark-teal)',
		width = 104,
		height = 30,
		fill = true
	}: {
		values: number[];
		color?: string;
		width?: number;
		height?: number;
		fill?: boolean;
	} = $props();

	const pad = 2;

	const geom = $derived.by(() => {
		const clean = values.filter((v) => Number.isFinite(v));
		if (clean.length === 0) return null;
		const min = Math.min(...clean);
		const max = Math.max(...clean);
		const span = max - min || 1;
		const n = clean.length;
		const coords = clean.map((v, i) => {
			const x = n === 1 ? width / 2 : pad + (i / (n - 1)) * (width - 2 * pad);
			const y = height - pad - ((v - min) / span) * (height - 2 * pad);
			return [x, y] as const;
		});
		const line = coords.map(([x, y]) => `${x.toFixed(1)},${y.toFixed(1)}`).join(' ');
		const area = `${coords[0][0].toFixed(1)},${height} ${line} ${coords[n - 1][0].toFixed(1)},${height}`;
		const last = coords[n - 1];
		return { line, area, last };
	});
</script>

{#if geom}
	<svg
		class="spark"
		viewBox={`0 0 ${width} ${height}`}
		{width}
		{height}
		preserveAspectRatio="none"
		aria-hidden="true"
	>
		{#if fill}
			<polygon points={geom.area} fill={color} fill-opacity="0.12" />
		{/if}
		<polyline
			points={geom.line}
			fill="none"
			stroke={color}
			stroke-width="2"
			stroke-linecap="round"
			stroke-linejoin="round"
		/>
		<circle cx={geom.last[0]} cy={geom.last[1]} r="2.4" fill={color} />
	</svg>
{:else}
	<span class="spark-empty">—</span>
{/if}

<style>
	.spark {
		display: block;
		overflow: visible;
	}
	.spark-empty {
		color: var(--text-muted);
		font-size: 0.8rem;
	}
</style>
