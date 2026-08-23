<script lang="ts">
  import { APP_VERSION } from "$lib/config";
  import { page } from "$app/state";
  import { Command, HeartPulse, LayoutDashboard } from "@lucide/svelte";
  import CommandPalette from "$lib/components/CommandPalette.svelte";
  import NavigationMenu from "$lib/components/NavigationMenu.svelte";
  import ThemeToggle from "$lib/components/ThemeToggle.svelte";
  import DesignLanguagePicker from "$lib/components/DesignLanguagePicker.svelte";
  import { navigationGroups } from "$lib/navigation";
  import ThemeFrame from "$lib/themes/ThemeFrame.svelte";
  import ThemeProvider from "$lib/themes/ThemeProvider.svelte";
  import "./app.css";

  let { children } = $props();

  function isActive(href: string) {
    return href === "/"
      ? page.url.pathname === "/"
      : page.url.pathname === href || page.url.pathname.startsWith(`${href}/`);
  }

  function openCommandPalette() {
    window.dispatchEvent(new CustomEvent("iroha:command-palette:toggle"));
  }
</script>

<ThemeProvider>
  <a class="skip-link" href="#main-content">Skip to main content</a>
  <div class="app">
    <header class="appbar">
      <a
        class="brand brand-observatory"
        href="/"
        aria-label="iroha — sound and flower"
      >
        <img class="brand-mark" src="/favicon.svg" alt="" aria-hidden="true" />
        <span>iroha</span>
        <small class="brand-version">v{APP_VERSION}</small>
      </a>
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
      <nav class="main-nav" aria-label="Primary navigation">
        <a
          class:active={isActive(navigationGroups[0].items[0].href)}
          href={navigationGroups[0].items[0].href}
          ><HeartPulse size={14} />{navigationGroups[0].items[0].label}</a
        >
        <a
          class:active={isActive(navigationGroups[0].items[1].href)}
          href={navigationGroups[0].items[1].href}
          ><LayoutDashboard size={14} />{navigationGroups[0].items[1].label}</a
        >
        {#each navigationGroups.slice(1) as group}
          <NavigationMenu {group} active={isActive} />
        {/each}
      </nav>
    </header>
    <CommandPalette />
    <ThemeFrame>
      {@render children()}
    </ThemeFrame>
  </div>
</ThemeProvider>

<style>
  .skip-link {
    position: fixed;
    top: 0.5rem;
    left: 0.5rem;
    z-index: 100;
    min-height: 2rem;
    padding: 0.45rem 0.7rem;
    transform: translateY(calc(-100% - 1rem));
    border: 2px solid var(--color-focus);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    font-weight: 700;
  }

  .skip-link:focus {
    transform: translateY(0);
  }
</style>
