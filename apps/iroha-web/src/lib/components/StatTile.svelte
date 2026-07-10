<script lang="ts">
	let {
		label,
		value,
		sub
	}: {
		label: string;
		value: string;
		sub?: string;
	} = $props();
</script>

<section class="stat-tile tile">
	<div class="stat-label">{label}</div>
	<div class="stat-value">{value}</div>
	{#if sub}
		<div class="stat-sub">{sub}</div>
	{/if}
</section>

<style>
	.stat-tile {
		min-height: 7rem;
		padding: 1rem;
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		gap: 0.6rem;
	}

	.stat-label {
		color: var(--text-muted);
		font-size: 0.72rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}

	.stat-value {
		color: var(--text);
		font-size: clamp(1.45rem, 4vw, 2.2rem);
		font-weight: 750;
		line-height: 1;
		letter-spacing: 0;
		white-space: nowrap;
	}

	.stat-sub {
		color: var(--text-muted);
		font-size: 0.84rem;
		line-height: 1.35;
	}
</style>
