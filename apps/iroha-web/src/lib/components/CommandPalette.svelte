<script module lang="ts">
  import { getMetricCatalog, type MetricCatalogResponse } from "$lib/api";

  let metricCatalogPromise: Promise<MetricCatalogResponse> | null = null;

  function loadMetricCatalog(): Promise<MetricCatalogResponse> {
    if (!metricCatalogPromise) {
      metricCatalogPromise = getMetricCatalog().catch((cause) => {
        metricCatalogPromise = null;
        throw cause;
      });
    }
    return metricCatalogPromise;
  }
</script>

<script lang="ts">
  import { goto } from "$app/navigation";
  import { onMount } from "svelte";
  import { tick } from "svelte";
  import { allNavigationItems } from "$lib/navigation";

  type Command = {
    label: string;
    href: string;
    hint: string;
  };

  let commands = $state<Command[]>(allNavigationItems());

  let open = $state(false);
  let selected = $state(0);
  let dialog = $state<HTMLDivElement>();
  let listbox = $state<HTMLDivElement>();
  let previouslyFocused: HTMLElement | null = null;

  async function openPalette() {
    previouslyFocused =
      document.activeElement instanceof HTMLElement
        ? document.activeElement
        : null;
    open = true;
    selected = 0;
    void loadMetrics();
    await tick();
    listbox?.focus();
  }

  function closePalette() {
    open = false;
    const target = previouslyFocused;
    previouslyFocused = null;
    target?.focus();
  }

  onMount(() => {
    const onToggle = () => {
      if (open) closePalette();
      else void openPalette();
    };
    const onKeydown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        onToggle();
        return;
      }
      if (!open) return;
      if (event.key === "Escape") {
        event.preventDefault();
        closePalette();
        return;
      }
      if (event.key === "ArrowDown") {
        event.preventDefault();
        selected = (selected + 1) % commands.length;
        return;
      }
      if (event.key === "ArrowUp") {
        event.preventDefault();
        selected = (selected - 1 + commands.length) % commands.length;
        return;
      }
      if (event.key === "Enter") {
        event.preventDefault();
        void activate(commands[selected]);
      }
    };

    window.addEventListener("iroha:command-palette:toggle", onToggle);
    window.addEventListener("keydown", onKeydown);
    return () => {
      window.removeEventListener("iroha:command-palette:toggle", onToggle);
      window.removeEventListener("keydown", onKeydown);
    };
  });

  async function loadMetrics() {
    try {
      const catalog = await loadMetricCatalog();
      commands = [
        ...allNavigationItems(),
        ...catalog.metrics.map((metric) => ({
          label: metric.label,
          href: `/metrics?metric=${encodeURIComponent(metric.id)}`,
          hint: `${metric.domain} · ${metric.unit}`,
        })),
      ];
    } catch {
      // Navigation remains usable when metric discovery is unavailable.
    }
  }

  async function activate(command: Command) {
    closePalette();
    await goto(command.href);
  }

  function trapDialogFocus(event: KeyboardEvent) {
    if (event.key !== "Tab" || !dialog) return;
    const focusable = Array.from(
      dialog.querySelectorAll<HTMLElement>(
        "button, [tabindex]:not([tabindex='-1'])",
      ),
    );
    if (focusable.length === 0) return;
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  }
</script>

{#if open}
  <div class="palette-backdrop" role="presentation" onclick={closePalette}>
    <div
      class="palette tile"
      bind:this={dialog}
      role="dialog"
      aria-modal="true"
      aria-label="Command palette"
      tabindex="-1"
      onclick={(event) => event.stopPropagation()}
      onkeydown={trapDialogFocus}
    >
      <header>
        <span>Command</span>
        <kbd>Esc</kbd>
      </header>
      <div
        class="command-list"
        bind:this={listbox}
        role="listbox"
        tabindex="0"
        aria-label="Navigation commands"
        aria-activedescendant={`command-${selected}`}
      >
        {#each commands as command, index}
          <button
            type="button"
            id={`command-${index}`}
            class:selected={index === selected}
            role="option"
            aria-selected={index === selected}
            onmouseenter={() => (selected = index)}
            onclick={() => activate(command)}
          >
            <span class="command-label">{command.label}</span>
            <span class="command-hint">{command.hint}</span>
          </button>
        {/each}
      </div>
    </div>
  </div>
{/if}

<style>
  .palette-backdrop {
    position: fixed;
    inset: 0;
    z-index: 40;
    display: grid;
    place-items: start center;
    padding: max(12vh, 4rem) 1rem 1rem;
    background: rgb(5 7 10 / 0.62);
    backdrop-filter: blur(10px);
  }

  .palette {
    display: flex;
    flex-direction: column;
    width: min(34rem, 100%);
    max-height: min(42rem, 80svh);
    padding: 0.65rem;
    overflow: hidden;
  }

  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    padding: 0.45rem 0.55rem 0.65rem;
    color: var(--text-muted);
    font-size: 0.74rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  kbd {
    padding: 0.16rem 0.36rem;
    border: 1px solid var(--border);
    border-radius: 4px;
    background: var(--surface-2);
    color: var(--text-muted);
    font: inherit;
    letter-spacing: 0;
    text-transform: none;
  }

  .command-list {
    min-height: 0;
    overflow-y: auto;
    display: grid;
    gap: 0.3rem;
    overscroll-behavior: contain;
  }

  button {
    width: 100%;
    min-height: 3.4rem;
    padding: 0.7rem 0.8rem;
    display: grid;
    gap: 0.24rem;
    border: 1px solid transparent;
    border-radius: var(--radius);
    background: transparent;
    color: var(--text);
    font: inherit;
    text-align: left;
    cursor: pointer;
  }

  button.selected,
  button:hover {
    border-color: color-mix(in srgb, var(--accent), var(--border) 35%);
    background: color-mix(in srgb, var(--accent) 12%, transparent);
  }

  .command-label {
    font-weight: 720;
  }

  .command-hint {
    color: var(--text-muted);
    font-size: 0.82rem;
  }

  @media (max-width: 640px) {
    .palette-backdrop {
      place-items: end center;
      padding: 0;
      backdrop-filter: none;
    }

    .palette {
      width: 100%;
      max-height: min(42rem, 88svh);
      padding: 0.75rem 0.75rem calc(0.75rem + env(safe-area-inset-bottom));
      border-radius: var(--radius) var(--radius) 0 0;
    }

    header {
      padding: 0.25rem 0.35rem 0.7rem;
    }

    button {
      min-height: 3.7rem;
      padding-inline: 0.65rem;
    }
  }
</style>
