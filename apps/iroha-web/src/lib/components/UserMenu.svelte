<script lang="ts">
  import { LogOut, Settings, Shield } from "@lucide/svelte";
  import { auth, logout } from "$lib/auth.svelte";
  import AccountDialog from "./AccountDialog.svelte";

  let menu: HTMLDetailsElement;
  let accountOpen = $state(false);
  const name = $derived(auth.displayName || auth.username);
  const initial = $derived(name.slice(0, 1).toUpperCase());

  function close() {
    if (menu) menu.open = false;
  }

  function openAccount() {
    close();
    accountOpen = true;
  }

  function onWindowClick(event: MouseEvent) {
    if (menu?.open && !menu.contains(event.target as Node)) close();
  }

  function onKeydown(event: KeyboardEvent) {
    if (event.key === "Escape" && menu?.open) {
      close();
      menu.querySelector("summary")?.focus();
    }
  }
</script>

<svelte:window onclick={onWindowClick} onkeydown={onKeydown} />

<details class="user-menu" bind:this={menu}>
  <summary aria-label={`Account menu for ${name}`}>
    <span class="avatar" aria-hidden="true">{initial}</span>
    <span class="user-name">{name}</span>
  </summary>
  <div class="user-popover" role="menu">
    <div class="user-identity">
      <strong>{name}</strong>
      {#if auth.displayName}<small>@{auth.username}</small>{/if}
    </div>
    <button type="button" role="menuitem" onclick={openAccount}>
      <Settings size={15} /> Account settings
    </button>
    <a href="/admin" role="menuitem" onclick={close}>
      <Shield size={15} /> Admin
    </a>
    <button
      type="button"
      role="menuitem"
      class="logout"
      onclick={() => {
        close();
        void logout();
      }}
    >
      <LogOut size={15} /> Log out
    </button>
  </div>
</details>

<AccountDialog bind:open={accountOpen} />

<style>
  .user-menu {
    position: relative;
  }

  summary {
    display: inline-flex;
    min-height: 2.4rem;
    align-items: center;
    gap: 0.45rem;
    padding: 0 0.55rem 0 0.3rem;
    border: 1px solid var(--border);
    border-radius: 999px;
    background: var(--surface);
    color: var(--text);
    cursor: pointer;
    list-style: none;
  }

  summary::-webkit-details-marker {
    display: none;
  }

  summary:focus-visible {
    outline: 2px solid var(--color-focus);
    outline-offset: 2px;
  }

  .avatar {
    display: grid;
    width: 1.8rem;
    height: 1.8rem;
    place-items: center;
    border-radius: 50%;
    background: var(--accent);
    color: var(--surface);
    font-size: 0.8rem;
    font-weight: 700;
  }

  .user-name {
    max-width: 9rem;
    overflow: hidden;
    font-size: 0.85rem;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .user-popover {
    position: absolute;
    top: calc(100% + 0.4rem);
    right: 0;
    z-index: 40;
    display: grid;
    min-width: 13rem;
    padding: 0.35rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    box-shadow: 0 12px 32px rgb(0 0 0 / 0.14);
  }

  .user-identity {
    display: grid;
    gap: 0.1rem;
    padding: 0.55rem 0.6rem 0.65rem;
    margin-bottom: 0.25rem;
    border-bottom: 1px solid var(--border);
  }

  .user-identity small {
    color: var(--text-muted);
  }

  .user-popover button,
  .user-popover a {
    display: flex;
    min-height: 2.4rem;
    align-items: center;
    gap: 0.55rem;
    padding: 0 0.6rem;
    border: 0;
    border-radius: calc(var(--radius) - 4px);
    background: none;
    color: var(--text);
    font: inherit;
    font-size: 0.88rem;
    text-align: left;
    text-decoration: none;
    cursor: pointer;
  }

  .user-popover button:hover,
  .user-popover a:hover,
  .user-popover button:focus-visible,
  .user-popover a:focus-visible {
    background: color-mix(in srgb, var(--accent) 10%, transparent);
    outline: none;
  }

  .logout {
    color: var(--danger) !important;
  }

  @media (max-width: 640px) {
    .user-name {
      display: none;
    }
  }
</style>
