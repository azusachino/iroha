<script lang="ts">
  import { APP_VERSION } from "$lib/config";
  import { page } from "$app/state";
  import {
    Activity,
    BookOpen,
    FileText,
    Command,
    Footprints,
    HeartPulse,
    LayoutDashboard,
    ListTodo,
    Moon,
    WalletCards,
  } from "@lucide/svelte";
  import CommandPalette from "$lib/components/CommandPalette.svelte";
  import ThemeToggle from "$lib/components/ThemeToggle.svelte";
  import DesignLanguagePicker from "$lib/components/DesignLanguagePicker.svelte";
  import { primaryNavigation } from "$lib/navigation";
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
      <nav class="main-nav" aria-label="Primary navigation">
        <a
          class:active={isActive(primaryNavigation[0].href)}
          href={primaryNavigation[0].href}
          ><HeartPulse size={14} />{primaryNavigation[0].label}</a
        >
        <a
          class:active={isActive(primaryNavigation[1].href)}
          href={primaryNavigation[1].href}
          ><LayoutDashboard size={14} />{primaryNavigation[1].label}</a
        >
        <a
          class:active={isActive(primaryNavigation[2].href)}
          href={primaryNavigation[2].href}
          ><Activity size={14} />{primaryNavigation[2].label}</a
        >
        <a
          class:active={isActive(primaryNavigation[3].href)}
          href={primaryNavigation[3].href}
          ><Footprints size={14} />{primaryNavigation[3].label}</a
        >
        <a
          class:active={isActive(primaryNavigation[4].href)}
          href={primaryNavigation[4].href}
          ><Moon size={14} />{primaryNavigation[4].label}</a
        >
        <a
          class:active={isActive(primaryNavigation[5].href)}
          href={primaryNavigation[5].href}
          ><BookOpen size={14} />{primaryNavigation[5].label}</a
        >
        <a
          class:active={isActive(primaryNavigation[6].href)}
          href={primaryNavigation[6].href}
          ><ListTodo size={14} />{primaryNavigation[6].label}</a
        >
        <a
          class:active={isActive(primaryNavigation[7].href)}
          href={primaryNavigation[7].href}
          ><WalletCards size={14} />{primaryNavigation[7].label}</a
        >
        <a
          class:active={isActive(primaryNavigation[8].href)}
          href={primaryNavigation[8].href}
          ><FileText size={14} />{primaryNavigation[8].label}</a
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
