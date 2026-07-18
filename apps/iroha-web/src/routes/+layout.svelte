<script lang="ts">
  import { onMount } from "svelte";
  import favicon from "$lib/assets/favicon.svg";
  import { page } from "$app/state";
  import {
    Activity,
    BookOpen,
    Command,
    HeartPulse,
    LayoutDashboard,
    Moon,
    Share2,
  } from "@lucide/svelte";
  import CommandPalette from "$lib/components/CommandPalette.svelte";
  import ThemeToggle from "$lib/components/ThemeToggle.svelte";
  import DesignLanguagePicker from "$lib/components/DesignLanguagePicker.svelte";
  import GrapherShell from "$lib/themes/grapher/Shell.svelte";
  import { getDesignLanguage, type DesignLanguage } from "$lib/design-language";
  import "./app.css";

  let { children } = $props();
  let language = $state<DesignLanguage>("field-journal");

  onMount(() => {
    language = getDesignLanguage();
    const onLanguageChange = (event: Event) => {
      language = (event as CustomEvent<DesignLanguage>).detail;
    };
    window.addEventListener("iroha:design-language-change", onLanguageChange);
    return () =>
      window.removeEventListener(
        "iroha:design-language-change",
        onLanguageChange,
      );
  });

  const shareActive = $derived(page.url.pathname.startsWith("/share"));
  const mediaActive = $derived(page.url.pathname.startsWith("/media"));
  const homeActive = $derived(page.url.pathname === "/");
  const dashboardActive = $derived(page.url.pathname.startsWith("/dashboard"));
  const dailyActive = $derived(page.url.pathname.startsWith("/daily"));
  const activitiesActive = $derived(
    page.url.pathname.startsWith("/activities"),
  );
  const sleepActive = $derived(page.url.pathname.startsWith("/sleep"));

  function openCommandPalette() {
    window.dispatchEvent(new CustomEvent("iroha:command-palette:toggle"));
  }
</script>

<svelte:head>
  <link rel="icon" href={favicon} />
</svelte:head>

<div class="app">
  <header class="appbar">
    <a
      class="brand brand-observatory"
      href="/"
      aria-label="iroha — sound and flower"
    >
      <span class="brand-mark" aria-hidden="true">✽</span>
      <span>iroha</span>
    </a>
    <nav class="main-nav" aria-label="Primary navigation">
      <a class:active={homeActive} href="/"><HeartPulse size={14} />Today</a>
      <a class:active={dashboardActive} href="/dashboard"
        ><LayoutDashboard size={14} />Overview</a
      >
      <a class:active={dailyActive} href="/daily"
        ><Activity size={14} />Patterns</a
      >
      <a class:active={activitiesActive} href="/activities">Motion</a>
      <a class:active={sleepActive} href="/sleep"><Moon size={14} />Night</a>
      <a class:active={mediaActive} href="/media"
        ><BookOpen size={14} />Library</a
      >
    </nav>
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
      <DesignLanguagePicker />
      <ThemeToggle />
    </div>
  </header>
  <CommandPalette />
  {#if language === "grapher"}
    <GrapherShell>
      {@render children()}
    </GrapherShell>
  {:else}
    <main class="content">
      {@render children()}
    </main>
    <footer class="footer">Private activity viewer</footer>
  {/if}
</div>
