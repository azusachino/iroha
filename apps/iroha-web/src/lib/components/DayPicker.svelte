<script lang="ts">
	// A compact month-grid day picker. Days that actually have data are dotted,
	// so scrubbing lands on real days. Emits the chosen 'YYYY-MM-DD'.
	let {
		value,
		days,
		max,
		onselect
	}: {
		value: string;
		days: Set<string>;
		max?: string;
		onselect: (day: string) => void;
	} = $props();

	// Shown month follows the selected value, unless the user browsed months.
	let monthOverride = $state<string | null>(null);
	$effect(() => {
		void value;
		monthOverride = null; // a new selection re-centers on its month
	});
	const view = $derived(monthOverride ?? value.slice(0, 7)); // 'YYYY-MM'

	const WEEKDAYS = ['S', 'M', 'T', 'W', 'T', 'F', 'S'];

	const cells = $derived.by<(string | null)[]>(() => {
		const [y, m] = view.split('-').map(Number);
		const startDow = new Date(Date.UTC(y, m - 1, 1)).getUTCDay();
		const daysInMonth = new Date(Date.UTC(y, m, 0)).getUTCDate();
		const out: (string | null)[] = [];
		for (let i = 0; i < startDow; i++) out.push(null);
		for (let d = 1; d <= daysInMonth; d++) out.push(`${view}-${String(d).padStart(2, '0')}`);
		return out;
	});
	const monthLabel = $derived(
		new Date(view + '-01T00:00:00Z').toLocaleDateString(undefined, {
			month: 'long',
			year: 'numeric',
			timeZone: 'UTC'
		})
	);
	function shiftMonth(delta: number) {
		const [y, m] = view.split('-').map(Number);
		monthOverride = new Date(Date.UTC(y, m - 1 + delta, 1)).toISOString().slice(0, 7);
	}
</script>

<div class="picker">
	<div class="pk-head">
		<button type="button" aria-label="Previous month" onclick={() => shiftMonth(-1)}>‹</button>
		<span>{monthLabel}</span>
		<button type="button" aria-label="Next month" onclick={() => shiftMonth(1)}>›</button>
	</div>
	<div class="pk-grid">
		{#each WEEKDAYS as w}
			<span class="dow">{w}</span>
		{/each}
		{#each cells as d}
			{#if d == null}
				<span></span>
			{:else}
				<button
					type="button"
					class="day"
					class:selected={d === value}
					class:has={days.has(d)}
					disabled={max ? d > max : false}
					onclick={() => onselect(d)}
				>
					{Number(d.slice(8))}
				</button>
			{/if}
		{/each}
	</div>
</div>

<style>
	.picker {
		width: 15rem;
		padding: 0.75rem;
	}
	.pk-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 0.5rem;
		font-weight: 650;
		color: var(--text);
	}
	.pk-head button {
		appearance: none;
		border: 0;
		background: transparent;
		color: var(--text-muted);
		font-size: 1.1rem;
		width: 1.8rem;
		height: 1.8rem;
		border-radius: var(--radius);
		cursor: pointer;
	}
	.pk-head button:hover {
		color: var(--accent);
		background: color-mix(in srgb, var(--accent) 12%, transparent);
	}
	.pk-grid {
		display: grid;
		grid-template-columns: repeat(7, 1fr);
		gap: 2px;
	}
	.dow {
		text-align: center;
		font-size: 0.68rem;
		color: var(--text-muted);
		padding-bottom: 0.25rem;
	}
	.day {
		appearance: none;
		border: 0;
		background: transparent;
		color: var(--text);
		aspect-ratio: 1;
		border-radius: var(--radius);
		font-size: 0.8rem;
		cursor: pointer;
		position: relative;
	}
	.day:hover:not(:disabled) {
		background: color-mix(in srgb, var(--accent) 14%, transparent);
	}
	.day:disabled {
		color: color-mix(in srgb, var(--text-muted) 45%, transparent);
		cursor: default;
	}
	/* dot marks days that actually have data */
	.day.has::after {
		content: '';
		position: absolute;
		bottom: 3px;
		left: 50%;
		transform: translateX(-50%);
		width: 4px;
		height: 4px;
		border-radius: 50%;
		background: var(--accent);
	}
	.day.selected {
		background: var(--accent);
		color: #08131a;
		font-weight: 700;
		box-shadow: var(--accent-glow);
	}
	.day.selected.has::after {
		background: #08131a;
	}
</style>
