<script lang="ts">
	import favicon from '$lib/assets/favicon.svg';
	import { page } from '$app/state';
	import './app.css';

	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

<div class="app">
	<header class="topbar">
		<a class="brand" href="/">iroha</a>
		<nav>
			<a class="nav-link" class:active={page.url.pathname === '/'} href="/">Activities</a>
			<a class="nav-link" class:active={page.url.pathname.startsWith('/u')} href="/u">Public</a>
			<a
				class="nav-link"
				class:active={page.url.pathname.startsWith('/postgrest-spike')}
				href="/postgrest-spike">PostgREST</a
			>
		</nav>
	</header>
	<main class="content">
		{@render children()}
	</main>
	<footer class="footer">Private activity viewer</footer>
</div>
