<script lang="ts">
  import { APP_VERSION } from "$lib/config";
  import { page } from "$app/state";
  import {
    Command,
    HeartPulse,
    KeyRound,
    LayoutDashboard,
  } from "@lucide/svelte";
  import CommandPalette from "$lib/components/CommandPalette.svelte";
  import NavigationMenu from "$lib/components/NavigationMenu.svelte";
  import ThemeToggle from "$lib/components/ThemeToggle.svelte";
  import UserMenu from "$lib/components/UserMenu.svelte";
  import { navigationGroups } from "$lib/navigation";
  import ThemeFrame from "$lib/themes/ThemeFrame.svelte";
  import ThemeProvider from "$lib/themes/ThemeProvider.svelte";
  import AuthDialog from "$lib/components/AuthDialog.svelte";
  import { UNAUTHENTICATED_EVENT } from "$lib/api";
  import { auth, loadSession, sessionExpired } from "$lib/auth.svelte";
  import { onMount } from "svelte";
  import "@iroha/shared/theme/chart-presentation";
  import "./app.css";

  let { children } = $props();
  let authDialogOpen = $state(false);

  onMount(() => {
    void loadSession();
    window.addEventListener(UNAUTHENTICATED_EVENT, sessionExpired);
    return () =>
      window.removeEventListener(UNAUTHENTICATED_EVENT, sessionExpired);
  });

  function isActive(href: string) {
    return href === "/"
      ? page.url.pathname === "/"
      : page.url.pathname === href || page.url.pathname.startsWith(`${href}/`);
  }

  function openCommandPalette() {
    window.dispatchEvent(new CustomEvent("iroha:command-palette:toggle"));
  }
</script>

{#snippet brand()}
  <a
    class="brand brand-observatory"
    href="/"
    aria-label="iroha — sound and flower"
  >
    <img class="brand-mark" src="/favicon.svg" alt="" aria-hidden="true" />
    <span>iroha</span>
    <small class="brand-version">v{APP_VERSION}</small>
  </a>
{/snippet}

{#snippet nav()}
  {#if auth.status === "ready"}
    <nav class="main-nav" aria-label="Primary navigation">
      <a
        class:active={isActive(navigationGroups[0].items[0].href)}
        aria-current={isActive(navigationGroups[0].items[0].href)
          ? "page"
          : undefined}
        href={navigationGroups[0].items[0].href}
        ><HeartPulse size={14} />{navigationGroups[0].items[0].label}</a
      >
      <a
        class:active={isActive(navigationGroups[0].items[1].href)}
        aria-current={isActive(navigationGroups[0].items[1].href)
          ? "page"
          : undefined}
        href={navigationGroups[0].items[1].href}
        ><LayoutDashboard size={14} />{navigationGroups[0].items[1].label}</a
      >
      {#each navigationGroups.slice(1) as group}
        <NavigationMenu {group} active={isActive} />
      {/each}
    </nav>
  {/if}
{/snippet}

{#snippet actions()}
  {#if auth.status === "ready"}
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
    <ThemeToggle />
    <UserMenu />
  {:else}
    <ThemeToggle />
    <button
      class="signin-btn"
      type="button"
      aria-label="Sign in as owner"
      onclick={() => (authDialogOpen = true)}
    >
      <KeyRound size={15} />
      <span>Sign in</span>
    </button>
  {/if}
{/snippet}

<ThemeProvider>
  <a class="skip-link" href="#main-content">Skip to main content</a>
  <div class="app">
    {#if auth.status === "ready"}
      <CommandPalette />
    {/if}
    <ThemeFrame {brand} {nav} {actions}>
      {#if auth.status === "ready" || page.url.pathname === "/"}
        {@render children()}
      {:else}
        <section class="auth-gate tile">
          <div class="auth-gate-inner">
            <div class="gate-icon">
              <KeyRound size={28} />
            </div>
            <h1>Owner login required</h1>
            <p class="muted">
              This section of the cockpit contains private records. Sign in as
              the owner to view it.
            </p>
            <div class="gate-actions">
              <button
                type="button"
                class="primary-gate-btn"
                onclick={() => (authDialogOpen = true)}
              >
                Sign in
              </button>
              <a href="/" class="secondary-gate-link"
                >← Return to public showcase</a
              >
            </div>
          </div>
        </section>
      {/if}
    </ThemeFrame>
  </div>
  <AuthDialog bind:open={authDialogOpen} />
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

  .signin-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.45rem;
    min-height: 2.25rem;
    padding: 0.4rem 0.85rem;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--accent) 15%, transparent);
    color: var(--text);
    font: inherit;
    font-size: 0.82rem;
    font-weight: 650;
    cursor: pointer;
    transition:
      background var(--motion-micro),
      border-color var(--motion-micro);
  }

  .signin-btn:hover {
    background: color-mix(in srgb, var(--accent) 25%, transparent);
    border-color: var(--accent);
  }

  .auth-gate {
    display: grid;
    place-items: center;
    min-height: 50vh;
    padding: 3rem 1.5rem;
    text-align: center;
  }

  .auth-gate-inner {
    display: grid;
    gap: 0.85rem;
    max-width: 24rem;
    place-items: center;
  }

  .gate-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 3.5rem;
    height: 3.5rem;
    border-radius: 50%;
    background: color-mix(in srgb, var(--accent) 15%, var(--surface-2));
    color: var(--accent);
  }

  .primary-gate-btn {
    min-height: 2.4rem;
    padding: 0 1.25rem;
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    background: var(--accent);
    color: var(--color-canvas, #090d16);
    font: inherit;
    font-weight: 700;
    font-size: 0.9rem;
    cursor: pointer;
  }

  .secondary-gate-link {
    display: inline-block;
    margin-top: 0.5rem;
    color: var(--text-muted);
    font-size: 0.85rem;
  }

  .secondary-gate-link:hover {
    color: var(--text);
  }
</style>
