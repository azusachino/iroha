<script lang="ts">
  import { goto } from "$app/navigation";
  import { Search, SunMoon } from "@lucide/svelte";
  import { onMount, tick } from "svelte";
  import {
    buildPaletteCommands,
    filterPaletteCommands,
    nextSelectedIndex,
    type PaletteCommand,
  } from "$lib/command-palette";
  import { dialogLightDismiss } from "$lib/dialog";
  import { navItemIcons } from "$lib/nav-icons";
  import { cycleTheme, themeState } from "$lib/theme";

  const commands = buildPaletteCommands();

  let open = $state(false);
  let query = $state("");
  let selected = $state(0);
  let dialog = $state<HTMLDialogElement>();
  let searchInput = $state<HTMLInputElement>();
  let listbox = $state<HTMLDivElement>();

  const visibleCommands = $derived(filterPaletteCommands(commands, query));

  $effect(() => {
    if (visibleCommands.length === 0) {
      selected = -1;
    } else if (selected < 0 || selected >= visibleCommands.length) {
      selected = 0;
    }
  });

  const activeOptionId = $derived(
    selected >= 0 && selected < visibleCommands.length
      ? `command-${selected}`
      : undefined,
  );

  function openPalette() {
    if (!dialog || dialog.open) return;
    query = "";
    selected = 0;
    open = true;
    dialog.showModal();
    searchInput?.focus();
  }

  function closePalette() {
    if (dialog?.open) {
      dialog.close();
    } else {
      open = false;
    }
  }

  function handleClose() {
    open = false;
    query = "";
  }

  onMount(() => {
    const onToggle = () => {
      if (dialog?.open) closePalette();
      else openPalette();
    };
    const onKeydown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        onToggle();
      }
    };

    window.addEventListener("iroha:command-palette:toggle", onToggle);
    window.addEventListener("keydown", onKeydown);
    return () => {
      window.removeEventListener("iroha:command-palette:toggle", onToggle);
      window.removeEventListener("keydown", onKeydown);
    };
  });

  async function selectCommand(index: number) {
    selected = index;
    if (index < 0) return;
    await tick();
    listbox?.querySelector<HTMLElement>(`#command-${index}`)?.scrollIntoView({
      block: "nearest",
    });
  }

  function onDialogKeydown(event: KeyboardEvent) {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      void selectCommand(
        nextSelectedIndex(selected, 1, visibleCommands.length),
      );
      return;
    }
    if (event.key === "ArrowUp") {
      event.preventDefault();
      void selectCommand(
        nextSelectedIndex(selected, -1, visibleCommands.length),
      );
      return;
    }
    if (event.key === "Enter") {
      const target = visibleCommands[selected];
      if (!target) return;
      event.preventDefault();
      void activate(target);
    }
  }

  async function activate(command: PaletteCommand) {
    closePalette();
    if (command.action.type === "cycle-theme") {
      cycleTheme();
      return;
    }
    await goto(command.action.href);
  }

  function hintFor(command: PaletteCommand): string {
    if (command.action.type === "cycle-theme") {
      return `Current: ${themeState.preference} (${themeState.effective}) · ${command.hint}`;
    }
    return command.hint;
  }
</script>

<dialog
  bind:this={dialog}
  class="palette tile"
  aria-label="Command palette"
  closedby="any"
  use:dialogLightDismiss
  onclose={handleClose}
  onkeydown={onDialogKeydown}
>
  <header class="palette-header">
    <span class="search-field">
      <Search size={15} aria-hidden="true" />
      <input
        bind:this={searchInput}
        type="text"
        class="palette-search"
        role="combobox"
        autocomplete="off"
        spellcheck="false"
        aria-label="Filter commands"
        aria-controls="command-palette-list"
        aria-expanded={open}
        aria-autocomplete="list"
        aria-activedescendant={activeOptionId}
        placeholder="Type a route or action…"
        bind:value={query}
      />
    </span>
    <kbd>Esc</kbd>
  </header>

  <div
    id="command-palette-list"
    class="command-list"
    bind:this={listbox}
    role="listbox"
    aria-label="Navigation commands"
  >
    {#if visibleCommands.length === 0}
      <p class="palette-empty" role="status">No matching commands.</p>
    {:else}
      {#each visibleCommands as command, index (command.id)}
        {@const CommandIcon =
          command.action.type === "cycle-theme"
            ? SunMoon
            : navItemIcons[command.id]}
        <button
          type="button"
          id={`command-${index}`}
          class:selected={index === selected}
          role="option"
          aria-selected={index === selected}
          tabindex="-1"
          onmouseenter={() => (selected = index)}
          onclick={() => void activate(command)}
        >
          <span class="command-row">
            <span class="command-label">
              {#if CommandIcon}
                <CommandIcon size={14} aria-hidden="true" />
              {/if}
              {command.label}
            </span>
            {#if command.group === "Action"}
              <span class="command-badge">Action</span>
            {/if}
          </span>
          <span class="command-hint">{hintFor(command)}</span>
        </button>
      {/each}
    {/if}
  </div>
</dialog>

<style>
  .palette {
    margin: max(12vh, 4rem) auto auto;
    width: min(34rem, calc(100% - 2rem));
    max-height: min(42rem, 80svh);
    padding: 0.65rem;
    overflow: hidden;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    box-shadow: var(--tile-shadow);
  }

  .palette[open] {
    display: flex;
    flex-direction: column;
    gap: 0.45rem;
  }

  .palette-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    padding: 0.25rem 0.35rem 0.55rem;
    border-bottom: 1px solid var(--border);
  }

  .search-field {
    display: flex;
    flex: 1 1 auto;
    align-items: center;
    gap: 0.5rem;
    min-width: 0;
    color: var(--text-muted);
  }

  .search-field :global(svg) {
    flex-shrink: 0;
  }

  .palette-search {
    width: 100%;
    min-width: 0;
    min-height: 2.25rem;
    padding: 0.35rem 0.5rem;
    border: 1px solid transparent;
    border-radius: calc(var(--radius) - 4px);
    background: var(--surface-2);
    color: var(--text);
    font: inherit;
    font-size: 0.92rem;
  }

  .palette-search::placeholder {
    color: var(--text-muted);
  }

  kbd {
    flex-shrink: 0;
    padding: 0.16rem 0.36rem;
    border: 1px solid var(--border);
    border-radius: 4px;
    background: var(--surface-2);
    color: var(--text-muted);
    font-size: 0.74rem;
    font-weight: 700;
  }

  .command-list {
    min-height: 0;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    overscroll-behavior: contain;
  }

  .palette-empty {
    margin: 0;
    padding: 1.5rem 0.8rem;
    color: var(--text-muted);
    font-size: 0.88rem;
    text-align: center;
  }

  button {
    /* Rows keep their content height; a grid list squeezed them to min-height
       and left the label/hint spilling out of the selection box. */
    flex: 0 0 auto;
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

  .command-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
  }

  .command-label {
    display: inline-flex;
    align-items: center;
    gap: 0.45rem;
    font-weight: 720;
  }

  .command-label :global(svg) {
    flex-shrink: 0;
    color: var(--text-muted);
    transition: color var(--motion-micro);
  }

  button.selected .command-label :global(svg),
  button:hover .command-label :global(svg) {
    color: var(--accent);
  }

  .command-badge {
    padding: 0.1rem 0.45rem;
    border: 1px solid var(--border);
    border-radius: 999px;
    background: var(--surface-2);
    color: var(--text-muted);
    font-size: 0.68rem;
    font-weight: 650;
    letter-spacing: 0.04em;
    text-transform: uppercase;
  }

  .command-hint {
    color: var(--text-muted);
    font-size: 0.82rem;
  }

  @media (max-width: 640px) {
    .palette {
      margin: auto 0 0;
      width: 100%;
      max-width: 100%;
      max-height: min(42rem, 88svh);
      padding: 0.75rem 0.75rem calc(0.75rem + env(safe-area-inset-bottom));
      border-radius: var(--radius) var(--radius) 0 0;
    }

    .palette-header {
      padding: 0.25rem 0.25rem 0.6rem;
    }

    button {
      min-height: 3.7rem;
      padding-inline: 0.65rem;
    }
  }
</style>
