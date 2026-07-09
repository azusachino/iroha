<script lang="ts">
	import favicon from '$lib/assets/favicon.svg';
	import { page } from '$app/state';
	import CommandPalette from '$lib/components/CommandPalette.svelte';
	import './app.css';

	let { children } = $props();

	const shareActive = $derived(page.url.pathname.startsWith('/share'));

	function openCommandPalette() {
		window.dispatchEvent(new CustomEvent('iroha:command-palette:toggle'));
	}
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

<div class="app">
	<header class="appbar">
		<a class="brand" href="/dashboard">iroha</a>
		<div class="appbar-actions">
			<button class="command-trigger" type="button" aria-label="Open command palette" onclick={openCommandPalette}>
				<span>Command</span>
				<kbd>⌘K</kbd>
			</button>
			<a class="share-link" class:active={shareActive} href="/share">Share</a>
		</div>
	</header>
	<CommandPalette />
	<main class="content">
		{@render children()}
	</main>
	<footer class="footer">Private activity viewer</footer>
</div>
