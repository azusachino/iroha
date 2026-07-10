<script lang="ts">
	type DomainState = 'active' | 'soon';

	let {
		name,
		stat,
		href,
		state = 'soon'
	}: {
		name: string;
		stat?: string;
		href?: string;
		state?: DomainState;
	} = $props();

	const isActive = $derived(state === 'active' && !!href);
</script>

{#if isActive}
	<a class="domain-tile tile tile-interactive active" href={href} aria-label={`Open ${name}`}>
		<span class="domain-kicker">Domain</span>
		<span class="domain-name">{name}</span>
		{#if stat}
			<span class="domain-stat">{stat}</span>
		{/if}
	</a>
{:else}
	<div class="domain-tile tile soon" aria-disabled="true">
		<span class="domain-kicker">Coming soon</span>
		<span class="domain-name">{name}</span>
		{#if stat}
			<span class="domain-stat">{stat}</span>
		{/if}
	</div>
{/if}

<style>
	.domain-tile {
		min-height: 8.5rem;
		padding: 1rem;
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		gap: 0.75rem;
		color: var(--text);
		text-decoration: none;
	}

	.domain-tile:hover {
		text-decoration: none;
	}

	.domain-kicker {
		color: var(--text-muted);
		font-size: 0.72rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}

	.domain-name {
		font-size: 1.25rem;
		font-weight: 750;
		line-height: 1.05;
	}

	.domain-stat {
		color: var(--text-muted);
		font-size: 0.86rem;
		line-height: 1.3;
	}

	.soon {
		opacity: 0.58;
	}
</style>
