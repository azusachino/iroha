<script lang="ts">
	import { sportColor, sportLabel } from '$lib/sport';

	let { sport }: { sport?: string | null } = $props();

	const label = $derived(sportLabel(sport));
	const color = $derived(sportColor(sport));
</script>

<span class="sport-badge" title={label}>
	<span class="sport-dot" style:background={color}></span>
	<span>{label}</span>
</span>

<style>
	.sport-badge {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		max-width: 100%;
		color: var(--text);
		font-size: 0.8rem;
		font-weight: 650;
		line-height: 1;
		white-space: nowrap;
	}

	.sport-dot {
		width: 0.55rem;
		height: 0.55rem;
		flex: 0 0 auto;
		border-radius: 999px;
		box-shadow: 0 0 0 3px color-mix(in srgb, currentColor 10%, transparent);
	}
</style>
