<script lang="ts">
  import { APP_VERSION } from "$lib/config";
  import { page } from "$app/state";
  import {
    Activity,
    BookOpen,
    Command,
    Footprints,
    HeartPulse,
    LayoutDashboard,
    ListTodo,
    Moon,
  } from "@lucide/svelte";
  import CommandPalette from "$lib/components/CommandPalette.svelte";
  import ThemeToggle from "$lib/components/ThemeToggle.svelte";
  import DesignLanguagePicker from "$lib/components/DesignLanguagePicker.svelte";
  import ThemeFrame from "$lib/themes/ThemeFrame.svelte";
  import ThemeProvider from "$lib/themes/ThemeProvider.svelte";
  import "./app.css";

  let { children } = $props();
  const mediaActive = $derived(page.url.pathname.startsWith("/media"));
  const homeActive = $derived(page.url.pathname === "/");
  const dashboardActive = $derived(page.url.pathname.startsWith("/dashboard"));
  const dailyActive = $derived(page.url.pathname.startsWith("/daily"));
  const activitiesActive = $derived(
    page.url.pathname.startsWith("/activities"),
  );
  const sleepActive = $derived(page.url.pathname.startsWith("/sleep"));
  const adminActive = $derived(page.url.pathname.startsWith("/admin"));

  function openCommandPalette() {
    window.dispatchEvent(new CustomEvent("iroha:command-palette:toggle"));
  }
</script>

<ThemeProvider>
  <div class="app">
    <header class="appbar">
      <a
        class="brand brand-observatory"
        href="/"
        aria-label="iroha — sound and flower"
      >
        <span class="brand-mark" aria-hidden="true">✽</span>
        <span>iroha</span>
        <small class="brand-version">v{APP_VERSION}</small>
      </a>
      <nav class="main-nav" aria-label="Primary navigation">
        <a class:active={homeActive} href="/"><HeartPulse size={14} />Today</a>
        <a class:active={dashboardActive} href="/dashboard"
          ><LayoutDashboard size={14} />Overview</a
        >
        <a class:active={dailyActive} href="/daily"
          ><Activity size={14} />Patterns</a
        >
        <a class:active={activitiesActive} href="/activities"
          ><Footprints size={14} />Motion</a
        >
        <a class:active={sleepActive} href="/sleep"><Moon size={14} />Night</a>
        <a class:active={mediaActive} href="/media"
          ><BookOpen size={14} />Library</a
        >
        <a class:active={adminActive} href="/admin"
          ><ListTodo size={14} />To-go</a
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
        <DesignLanguagePicker />
        <ThemeToggle />
      </div>
    </header>
    <CommandPalette />
    <ThemeFrame>
      {@render children()}
    </ThemeFrame>
  </div>
</ThemeProvider>
