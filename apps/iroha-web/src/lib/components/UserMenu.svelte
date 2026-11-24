<script lang="ts">
  import { LogOut, Settings, Shield } from "@lucide/svelte";
  import { onMount } from "svelte";
  import { auth, logout } from "$lib/auth.svelte";
  import AccountDialog from "./AccountDialog.svelte";

  const POPOVER_ID = "user-account-menu";

  let trigger = $state<HTMLButtonElement>();
  let popover = $state<HTMLDivElement>();
  let open = $state(false);
  let accountOpen = $state(false);
  let fallbackStyle = $state("");

  const name = $derived(auth.displayName || auth.username);
  const initial = $derived(name.slice(0, 1).toUpperCase());

  function supportsAnchorPositioning(): boolean {
    return (
      typeof CSS !== "undefined" &&
      typeof CSS.supports === "function" &&
      CSS.supports("position-area", "block-end")
    );
  }

  function updateFallbackPosition() {
    if (
      !open ||
      !trigger ||
      !popover ||
      typeof window === "undefined" ||
      supportsAnchorPositioning()
    ) {
      fallbackStyle = "";
      return;
    }
    const triggerRect = trigger.getBoundingClientRect();
    const menuRect = popover.getBoundingClientRect();
    const pad = 8;
    const width = menuRect.width || 208;
    const height = menuRect.height || 180;
    const inSideRail =
      Boolean(trigger.closest(".grapher-header")) &&
      window.matchMedia("(min-width: 1025px)").matches;

    let top = inSideRail
      ? triggerRect.top - height - 6
      : triggerRect.bottom + 6;
    if (top + height > window.innerHeight - pad) {
      top = Math.max(pad, triggerRect.top - height - 6);
    }
    if (top < pad) {
      top = Math.min(window.innerHeight - height - pad, triggerRect.bottom + 6);
    }

    const preferredLeft = inSideRail
      ? triggerRect.left
      : triggerRect.right - width;
    const left = Math.min(
      Math.max(pad, preferredLeft),
      Math.max(pad, window.innerWidth - width - pad),
    );
    fallbackStyle = `--user-popover-top:${top}px;--user-popover-left:${left}px;`;
  }

  function close() {
    if (popover?.matches(":popover-open")) {
      popover.hidePopover();
    }
  }

  function openAccount() {
    close();
    accountOpen = true;
  }

  function handleToggle(event: ToggleEvent) {
    open = event.newState === "open";
    if (open) {
      requestAnimationFrame(updateFallbackPosition);
    } else {
      fallbackStyle = "";
    }
  }

  function onMenuKeydown(event: KeyboardEvent) {
    if (
      !popover ||
      !["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)
    ) {
      return;
    }
    const items = Array.from(
      popover.querySelectorAll<HTMLElement>('[role="menuitem"]'),
    );
    if (items.length === 0) return;
    const current = items.indexOf(document.activeElement as HTMLElement);
    const next =
      event.key === "Home"
        ? 0
        : event.key === "End"
          ? items.length - 1
          : current < 0
            ? event.key === "ArrowUp"
              ? items.length - 1
              : 0
            : (current + (event.key === "ArrowUp" ? -1 : 1) + items.length) %
              items.length;
    event.preventDefault();
    items[next].focus();
  }

  function onTriggerKeydown(event: KeyboardEvent) {
    if (event.key !== "ArrowDown") return;
    event.preventDefault();
    if (!open) popover?.showPopover();
    requestAnimationFrame(() => {
      popover?.querySelector<HTMLElement>('[role="menuitem"]')?.focus();
    });
  }

  onMount(() => {
    const onReposition = () => updateFallbackPosition();
    window.addEventListener("resize", onReposition);
    window.addEventListener("scroll", onReposition, true);
    return () => {
      window.removeEventListener("resize", onReposition);
      window.removeEventListener("scroll", onReposition, true);
    };
  });
</script>

<div class="user-menu">
  <button
    bind:this={trigger}
    type="button"
    class="user-menu-trigger"
    aria-label={`Account menu for ${name}`}
    aria-haspopup="menu"
    aria-expanded={open}
    aria-controls={POPOVER_ID}
    popovertarget={POPOVER_ID}
    onkeydown={onTriggerKeydown}
  >
    <span class="avatar" aria-hidden="true">{initial}</span>
    <span class="user-name">{name}</span>
  </button>
  <div
    bind:this={popover}
    id={POPOVER_ID}
    class="user-popover"
    popover="auto"
    role="menu"
    tabindex="-1"
    aria-label={`Account menu for ${name}`}
    style={fallbackStyle}
    ontoggle={handleToggle}
    onkeydown={onMenuKeydown}
  >
    <div class="user-identity">
      <strong>{name}</strong>
      {#if auth.displayName}<small>@{auth.username}</small>{/if}
    </div>
    <button type="button" role="menuitem" onclick={openAccount}>
      <Settings size={15} aria-hidden="true" /> Account settings
    </button>
    <a href="/admin" role="menuitem" onclick={close}>
      <Shield size={15} aria-hidden="true" /> Admin
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
      <LogOut size={15} aria-hidden="true" /> Log out
    </button>
  </div>
</div>

<AccountDialog bind:open={accountOpen} />

<style>
  .user-menu {
    position: relative;
  }

  .user-menu-trigger {
    anchor-name: --user-menu-trigger;
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
  }

  .user-menu-trigger:focus-visible {
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
    min-width: 13rem;
    padding: 0.35rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    box-shadow: 0 12px 32px rgb(0 0 0 / 0.18);
  }

  .user-popover:popover-open {
    display: grid;
  }

  @supports (position-area: block-end) {
    .user-popover {
      position-anchor: --user-menu-trigger;
      position-area: block-end span-inline-start;
      position-try-fallbacks:
        flip-block,
        flip-inline,
        flip-block flip-inline;
      inset: auto;
      margin: 0.4rem 0 0;
    }
  }

  @supports not (position-area: block-end) {
    .user-popover {
      position: fixed;
      inset: auto;
      margin: 0;
      top: var(--user-popover-top, 0);
      left: var(--user-popover-left, 0);
    }
  }

  @media (prefers-reduced-motion: no-preference) {
    .user-popover {
      opacity: 0;
      translate: 0 -0.25rem;
      transition-property: opacity, translate, display, overlay;
      transition-duration: 140ms;
      transition-timing-function: ease-out;
      transition-behavior: allow-discrete;
    }

    .user-popover:popover-open {
      opacity: 1;
      translate: 0 0;
      transition-duration: 100ms;

      @starting-style {
        opacity: 0;
        translate: 0 -0.25rem;
      }
    }
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
