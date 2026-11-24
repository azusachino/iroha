<script lang="ts">
  import { onMount } from "svelte";
  import type { NavigationGroup } from "$lib/navigation";
  import { navGroupIcons, navItemIcons } from "$lib/nav-icons";

  let {
    group,
    active,
  }: {
    group: NavigationGroup;
    active: (href: string) => boolean;
  } = $props();

  const groupActive = $derived(group.items.some((item) => active(item.href)));
  const GroupIcon = $derived(navGroupIcons[group.id]);
  const popoverId = $derived(`${group.id}-navigation-menu`);
  const anchorName = $derived(`--nav-menu-${group.id}`);

  let root = $state<HTMLDivElement>();
  let trigger = $state<HTMLButtonElement>();
  let popover = $state<HTMLDivElement>();
  let open = $state(false);
  let inlineMode = $state(false);
  let hoverPreviewed = false;
  let fallbackStyle = $state("");
  let openTimer: ReturnType<typeof setTimeout> | undefined;
  let closeTimer: ReturnType<typeof setTimeout> | undefined;

  function supportsAnchorPositioning(): boolean {
    return (
      typeof CSS !== "undefined" &&
      typeof CSS.supports === "function" &&
      CSS.supports("position-area", "block-end")
    );
  }

  // In the side rail (desktop Grapher header >= 1025px) the sub-menu is
  // expanded inline like an accordion: it opens and closes only on click and
  // stays open across navigation. In the horizontal app bar it is a native
  // popover="auto" positioned with CSS Anchor Positioning.
  function detectInlineMode(): boolean {
    if (typeof window === "undefined" || !root) return false;
    return (
      Boolean(root.closest(".grapher-header")) &&
      window.matchMedia("(min-width: 1025px)").matches
    );
  }

  function syncLayoutMode() {
    if (!popover) return;
    const nextInline = detectInlineMode();
    if (nextInline) {
      if (popover.hasAttribute("popover")) {
        popover.removeAttribute("popover");
      }
      popover.hidden = !open;
      fallbackStyle = "";
    } else {
      popover.hidden = false;
      if (!popover.hasAttribute("popover")) {
        popover.setAttribute("popover", "auto");
        open = false;
      } else {
        open = popover.matches(":popover-open");
      }
      if (open) requestAnimationFrame(updateFallbackPosition);
    }
    inlineMode = nextInline;
  }

  function updateFallbackPosition() {
    if (
      inlineMode ||
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
    const viewportPadding = 8;
    const viewportWidth = document.documentElement.clientWidth;
    const preferredWidth = Math.min(208, viewportWidth - viewportPadding * 2);
    const height = popover.scrollHeight;
    const left = Math.min(
      Math.max(viewportPadding, triggerRect.left),
      viewportWidth - preferredWidth - viewportPadding,
    );
    const top = Math.min(
      triggerRect.bottom + 6,
      Math.max(viewportPadding, window.innerHeight - height - viewportPadding),
    );
    fallbackStyle = `--navigation-popover-top:${top}px;--navigation-popover-left:${left}px;`;
  }

  function cancelOpen() {
    if (openTimer) {
      clearTimeout(openTimer);
      openTimer = undefined;
    }
  }

  function cancelClose() {
    if (closeTimer) {
      clearTimeout(closeTimer);
      closeTimer = undefined;
    }
  }

  function closeFloatingMenu() {
    cancelOpen();
    cancelClose();
    hoverPreviewed = false;
    if (!inlineMode && popover?.matches(":popover-open")) {
      popover.hidePopover();
    }
  }

  function handleTriggerClick(event: MouseEvent) {
    if (inlineMode) {
      open = !open;
      if (popover) popover.hidden = !open;
      return;
    }
    // If the popover is currently open only from hover preview, clicking the
    // trigger pins it open instead of toggling it closed.
    if (hoverPreviewed && open) {
      hoverPreviewed = false;
      event.preventDefault();
    }
  }

  function handlePopoverToggle(event: ToggleEvent) {
    cancelOpen();
    open = event.newState === "open";
    if (!open) {
      hoverPreviewed = false;
      fallbackStyle = "";
    } else {
      requestAnimationFrame(updateFallbackPosition);
    }
  }

  function isHoverPointer(event: PointerEvent): boolean {
    return (
      (event.pointerType === "mouse" || event.pointerType === "pen") &&
      window.matchMedia("(hover: hover) and (pointer: fine)").matches
    );
  }

  function openOnPointer(event: PointerEvent) {
    if (inlineMode || !isHoverPointer(event)) return;
    cancelClose();
    if (open || openTimer) return;
    openTimer = setTimeout(() => {
      openTimer = undefined;
      if (inlineMode || !popover || popover.matches(":popover-open")) return;
      hoverPreviewed = true;
      popover.showPopover();
    }, 180);
  }

  function closeOnPointer(event: PointerEvent) {
    if (inlineMode || !isHoverPointer(event)) return;
    cancelOpen();
    cancelClose();
    closeTimer = setTimeout(() => {
      closeTimer = undefined;
      // Pointer departure must not hide a link being reached by keyboard.
      if (!root?.contains(document.activeElement)) {
        closeFloatingMenu();
      }
    }, 180);
  }

  function closeAfterNavigation() {
    if (!inlineMode) closeFloatingMenu();
  }

  onMount(() => {
    syncLayoutMode();
    const mediaQuery = window.matchMedia("(min-width: 1025px)");
    const onMediaChange = () => syncLayoutMode();
    const onReposition = () => updateFallbackPosition();
    mediaQuery.addEventListener("change", onMediaChange);
    window.addEventListener("resize", onReposition);
    window.addEventListener("scroll", onReposition, true);
    return () => {
      cancelOpen();
      cancelClose();
      mediaQuery.removeEventListener("change", onMediaChange);
      window.removeEventListener("resize", onReposition);
      window.removeEventListener("scroll", onReposition, true);
    };
  });
</script>

<div
  bind:this={root}
  class="navigation-menu"
  class:active={groupActive}
  class:open
  class:inline={inlineMode}
  class:floating={!inlineMode}
  role="group"
  aria-label={group.label}
  onpointerenter={openOnPointer}
  onpointerleave={closeOnPointer}
>
  <button
    bind:this={trigger}
    type="button"
    class="navigation-trigger"
    aria-expanded={open}
    aria-controls={popoverId}
    popovertarget={inlineMode ? undefined : popoverId}
    style={`anchor-name:${anchorName};`}
    onclick={handleTriggerClick}
  >
    {#if GroupIcon}
      <GroupIcon size={14} aria-hidden="true" />
    {/if}
    <span>{group.label}</span>
    <i class="chevron" aria-hidden="true"></i>
  </button>
  <div
    bind:this={popover}
    id={popoverId}
    class="navigation-popover"
    hidden={inlineMode && !open}
    style={`position-anchor:${anchorName};${fallbackStyle}`}
    ontoggle={handlePopoverToggle}
  >
    {#each group.items as item}
      {@const ItemIcon = navItemIcons[item.id]}
      <a
        class:active={active(item.href)}
        aria-current={active(item.href) ? "page" : undefined}
        href={item.href}
        onclick={closeAfterNavigation}
      >
        <span class="nav-item-title">
          {#if ItemIcon}
            <ItemIcon size={14} aria-hidden="true" />
          {/if}
          <strong>{item.label}</strong>
        </span>
        <small>{item.hint}</small>
      </a>
    {/each}
  </div>
</div>

<style>
  .navigation-menu {
    position: relative;
  }

  .navigation-trigger {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    min-height: 2rem;
    padding: 0 0.65rem;
    border: 0;
    background: transparent;
    color: var(--text-muted);
    font-size: 0.78rem;
    font-weight: 700;
    cursor: pointer;
  }

  .navigation-trigger :global(svg) {
    flex-shrink: 0;
  }

  .chevron {
    width: 0.46rem;
    height: 0.46rem;
    margin-left: 0.15rem;
    border-right: 1.5px solid currentColor;
    border-bottom: 1.5px solid currentColor;
    transform: rotate(45deg) translateY(-1px);
    transition: transform var(--motion-micro);
  }

  .navigation-menu.open .chevron {
    transform: rotate(225deg) translate(-1px, -1px);
  }

  .navigation-menu.active .navigation-trigger {
    color: var(--color-focus);
  }

  .navigation-menu.active .navigation-trigger :global(svg) {
    color: var(--accent);
  }

  .nav-item-title {
    display: inline-flex;
    align-items: center;
    gap: 0.45rem;
  }

  .nav-item-title :global(svg) {
    flex-shrink: 0;
    color: var(--text-muted);
    transition: color var(--motion-micro);
  }

  a:hover .nav-item-title :global(svg),
  a.active .nav-item-title :global(svg) {
    color: var(--accent);
  }

  .navigation-menu.floating .navigation-popover {
    gap: 0.25rem;
    min-width: 13rem;
    max-inline-size: calc(100vw - 1rem);
    max-block-size: min(60vh, 20rem);
    overflow-y: auto;
    padding: 0.45rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--glass-surface-strong, var(--surface-2));
    backdrop-filter: blur(calc(var(--glass-blur, 0px) / 2))
      saturate(var(--glass-saturate, 100%));
    box-shadow: var(--glass-highlight, none), var(--tile-shadow);
    color: var(--text);
  }

  .navigation-menu.floating .navigation-popover:popover-open {
    display: grid;
  }

  .navigation-menu.inline .navigation-popover[hidden] {
    display: none;
  }

  @supports (position-area: block-end) {
    .navigation-menu.floating .navigation-popover {
      position-area: block-end span-inline-end;
      position-try-fallbacks:
        flip-inline,
        --nav-viewport-fit,
        flip-block,
        flip-block flip-inline;
      inset: auto;
      height: auto;
      margin: 0.35rem 0 0;
    }
  }

  @supports not (position-area: block-end) {
    .navigation-menu.floating .navigation-popover {
      position: fixed;
      inset: auto;
      margin: 0;
      top: var(--navigation-popover-top, 0);
      left: var(--navigation-popover-left, 0);
      width: min(13rem, calc(100vw - 1rem));
      min-width: 0;
    }
  }

  @media (prefers-reduced-motion: no-preference) {
    .navigation-menu.floating .navigation-popover {
      opacity: 0;
      translate: 0 -0.25rem;
      transition-property: opacity, translate, display, overlay;
      transition-duration: 140ms;
      transition-timing-function: ease-out;
      transition-behavior: allow-discrete;
    }

    .navigation-menu.floating .navigation-popover:popover-open {
      opacity: 1;
      translate: 0 0;
      transition-duration: 100ms;

      @starting-style {
        opacity: 0;
        translate: 0 -0.25rem;
      }
    }
  }

  a {
    display: grid;
    gap: 0.18rem;
    min-width: 0;
    padding: 0.55rem 0.65rem;
    border-radius: calc(var(--radius) - 4px);
    color: var(--text);
    text-decoration: none;
  }

  a:hover,
  a.active {
    background: color-mix(in srgb, var(--accent) 12%, transparent);
    color: var(--text);
  }

  small {
    min-width: 0;
    color: var(--text-muted);
    font-size: 0.68rem;
  }

  @media (max-width: 768px) {
    .navigation-menu {
      flex: 0 1 auto;
      min-width: 0;
    }

    .navigation-menu.floating .navigation-popover {
      width: min(13rem, calc(100vw - 1rem));
      min-width: 0;
    }
  }

  @media (max-width: 640px) {
    .navigation-trigger {
      width: 100%;
      justify-content: center;
      gap: 0.15rem;
      padding: 0 0.1rem;
      font-size: 0.62rem;
    }

    .chevron {
      width: 0.4rem;
      height: 0.4rem;
      margin-left: 0.2rem;
    }
  }

  @media (prefers-reduced-transparency: reduce) {
    .navigation-menu.floating .navigation-popover {
      background: var(--surface-2);
      backdrop-filter: none;
    }
  }
</style>
