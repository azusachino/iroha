<script lang="ts">
  import favicon from "$lib/assets/favicon.svg";
  import { page } from "$app/state";
  import { BookOpen, Command, Share2 } from "@lucide/svelte";
  import CommandPalette from "$lib/components/CommandPalette.svelte";
  import ThemeToggle from "$lib/components/ThemeToggle.svelte";
  import "./app.css";

  let { children } = $props();

  const shareActive = $derived(page.url.pathname.startsWith("/share"));
  const mediaActive = $derived(page.url.pathname.startsWith("/media"));

  function openCommandPalette() {
    window.dispatchEvent(new CustomEvent("iroha:command-palette:toggle"));
  }
</script>

<svelte:head>
  <link rel="icon" href={favicon} />
</svelte:head>

<div class="app">
  <header class="appbar">
    <a class="brand" href="/">iroha</a>
    <div class="appbar-actions">
      <button
        class="command-trigger"
        type="button"
        aria-label="Open command palette"
        onclick={openCommandPalette}
      >
        <Command size={15} />
        <span>Command</span>
        <kbd>⌘K</kbd>
      </button>
      <a class="share-link" class:active={shareActive} href="/share">
        <Share2 size={15} />
        <span>Share</span>
      </a>
      <a class="share-link" class:active={mediaActive} href="/media">
        <BookOpen size={15} />
        <span>Media</span>
      </a>
      <ThemeToggle />
    </div>
  </header>
  <CommandPalette />
  <main class="content">
    {@render children()}
  </main>
  <footer class="footer">Private activity viewer</footer>
</div>
