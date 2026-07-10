<script lang="ts">
	import { onMount } from 'svelte';
	import { Sun, Moon } from '@lucide/svelte';
	import { getTheme, toggleTheme, type Theme } from '$lib/theme';

	let theme = $state<Theme>('dark');

	onMount(() => {
		theme = getTheme();
	});

	function onClick() {
		theme = toggleTheme();
	}
</script>

<button
	class="theme-toggle"
	type="button"
	aria-label={theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}
	title="Toggle theme"
	onclick={onClick}
>
	{#if theme === 'dark'}
		<Sun size={18} />
	{:else}
		<Moon size={18} />
	{/if}
</button>
