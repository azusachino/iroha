<script lang="ts">
  import { onMount } from "svelte";
  import type { NavigationGroup } from "$lib/navigation";

  let {
    group,
    active,
  }: {
    group: NavigationGroup;
    active: (href: string) => boolean;
  } = $props();

  const groupActive = $derived(group.items.some((item) => active(item.href)));
  let menu: HTMLDetailsElement;
  let summary: HTMLElement;
  let popoverStyle = $state("");

  function updatePopoverPosition() {
    if (!menu?.open || !summary || typeof window === "undefined") return;
    const popover = menu.querySelector<HTMLElement>(".navigation-popover");
    if (!popover) return;

    const trigger = summary.getBoundingClientRect();
    const viewportPadding = 8;
    const viewportWidth = document.documentElement.clientWidth;
    const preferredWidth = Math.min(208, viewportWidth - viewportPadding * 2);
    const height = popover.scrollHeight;
    const left = Math.min(
      Math.max(viewportPadding, trigger.left),
      viewportWidth - preferredWidth - viewportPadding,
    );
    const top = Math.min(
      trigger.bottom + 6,
      Math.max(viewportPadding, window.innerHeight - height - viewportPadding),
    );
    popoverStyle = `--navigation-popover-top:${top}px;--navigation-popover-left:${left}px;`;
  }

  function closeMenu() {
    if (menu) menu.open = false;
    popoverStyle = "";
  }

  function handleToggle() {
    if (!menu?.open) {
      popoverStyle = "";
      return;
    }
    window.dispatchEvent(
      new CustomEvent<HTMLDetailsElement>("iroha:navigation-open", {
        detail: menu,
      }),
    );
    requestAnimationFrame(updatePopoverPosition);
  }

  function isHoverPointer(event: PointerEvent): boolean {
    return (
      (event.pointerType === "mouse" || event.pointerType === "pen") &&
      window.matchMedia("(hover: hover) and (pointer: fine)").matches
    );
  }

  function openOnPointer(event: PointerEvent) {
    if (isHoverPointer(event)) menu.open = true;
  }

  function closeOnPointer(event: PointerEvent) {
    if (isHoverPointer(event)) closeMenu();
  }

  function closeAfterNavigation() {
    closeMenu();
  }

  onMount(() => {
    const closeOtherMenus = (event: Event) => {
      const opened = (event as CustomEvent<HTMLDetailsElement>).detail;
      if (opened !== menu) closeMenu();
    };
    const closeOnOutsidePointer = (event: PointerEvent) => {
      if (menu?.open && !menu.contains(event.target as Node)) closeMenu();
    };
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape" && menu?.open) {
        closeMenu();
        menu.querySelector("summary")?.focus();
      }
    };
    window.addEventListener("iroha:navigation-open", closeOtherMenus);
    document.addEventListener("pointerdown", closeOnOutsidePointer);
    window.addEventListener("keydown", closeOnEscape);
    window.addEventListener("resize", updatePopoverPosition);
    window.addEventListener("scroll", updatePopoverPosition, true);
    return () => {
      window.removeEventListener("iroha:navigation-open", closeOtherMenus);
      document.removeEventListener("pointerdown", closeOnOutsidePointer);
      window.removeEventListener("keydown", closeOnEscape);
      window.removeEventListener("resize", updatePopoverPosition);
      window.removeEventListener("scroll", updatePopoverPosition, true);
    };
  });
</script>

<details
  bind:this={menu}
  ontoggle={handleToggle}
  onpointerenter={openOnPointer}
  onpointerleave={closeOnPointer}
  class:active={groupActive}
  class="navigation-menu"
>
  <summary bind:this={summary} aria-controls={`${group.id}-navigation-menu`}
    ><span>{group.label}</span><i class="chevron" aria-hidden="true"
    ></i></summary
  >
  <div
    id={`${group.id}-navigation-menu`}
    class="navigation-popover"
    style={popoverStyle}
  >
    {#each group.items as item}
      <a
        class:active={active(item.href)}
        href={item.href}
        onclick={closeAfterNavigation}
      >
        <strong>{item.label}</strong><small>{item.hint}</small>
      </a>
    {/each}
  </div>
</details>

<style>
  .navigation-menu {
    position: relative;
  }
  summary {
    display: inline-flex;
    align-items: center;
    min-height: 2rem;
    padding: 0 0.65rem;
    color: var(--text-muted);
    font-size: 0.78rem;
    font-weight: 700;
    cursor: pointer;
    list-style: none;
  }
  summary::-webkit-details-marker {
    display: none;
  }
  .chevron {
    width: 0.46rem;
    height: 0.46rem;
    margin-left: 0.35rem;
    border-right: 1.5px solid currentColor;
    border-bottom: 1.5px solid currentColor;
    transform: rotate(45deg) translateY(-1px);
    transition: transform 160ms ease;
  }
  .navigation-menu[open] .chevron {
    transform: rotate(225deg) translate(-1px, -1px);
  }
  .navigation-menu.active summary {
    color: var(--accent);
  }
  .navigation-popover {
    position: absolute;
    top: calc(100% + 0.35rem);
    left: 0;
    z-index: 20;
    display: grid;
    gap: 0.25rem;
    min-width: 13rem;
    padding: 0.45rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface-2);
    box-shadow: var(--tile-shadow);
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
    color: var(--accent);
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
    .navigation-popover {
      position: fixed;
      top: var(--navigation-popover-top, -9999px);
      left: var(--navigation-popover-left, -9999px);
      width: min(13rem, calc(100vw - 1rem));
      min-width: 0;
      max-height: min(60vh, 20rem);
      overflow-y: auto;
    }
  }
  @media (max-width: 640px) {
    summary {
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
</style>
