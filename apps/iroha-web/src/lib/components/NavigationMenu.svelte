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

  function closeMenu() {
    if (menu) menu.open = false;
  }

  function handleToggle() {
    if (!menu?.open) return;
    window.dispatchEvent(
      new CustomEvent<HTMLDetailsElement>("iroha:navigation-open", {
        detail: menu,
      }),
    );
  }

  function closeAfterNavigation(event: MouseEvent) {
    (event.currentTarget as HTMLAnchorElement)
      .closest("details")
      ?.removeAttribute("open");
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
    return () => {
      window.removeEventListener("iroha:navigation-open", closeOtherMenus);
      document.removeEventListener("pointerdown", closeOnOutsidePointer);
      window.removeEventListener("keydown", closeOnEscape);
    };
  });
</script>

<details
  bind:this={menu}
  ontoggle={handleToggle}
  class:active={groupActive}
  class="navigation-menu"
>
  <summary
    ><span>{group.label}</span><i class="chevron" aria-hidden="true"
    ></i></summary
  >
  <div class="navigation-popover">
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
    color: var(--text-muted);
    font-size: 0.68rem;
  }
  @media (max-width: 760px) {
    .navigation-menu {
      flex: 1 1 auto;
    }
    .navigation-popover {
      position: static;
      min-width: 0;
    }
  }
</style>
