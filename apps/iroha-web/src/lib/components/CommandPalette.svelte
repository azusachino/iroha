<script lang="ts">
  import { goto } from "$app/navigation";
  import { onMount } from "svelte";

  type Command = {
    label: string;
    href: string;
    hint: string;
  };

  const commands: Command[] = [
    { label: "Cockpit", href: "/", hint: "Any-day cross-domain view" },
    { label: "Dashboard", href: "/dashboard", hint: "Activity overview" },
    {
      label: "Activities",
      href: "/activities",
      hint: "Private activity domain",
    },
    { label: "Sleep", href: "/sleep", hint: "Recovery and sleep sessions" },
    {
      label: "Daily & Vitals",
      href: "/daily",
      hint: "Rings, movement and body vitals",
    },
    { label: "Media", href: "/media", hint: "Watch and reading history" },
    { label: "Share", href: "/share", hint: "Public activity view" },
  ];

  let open = $state(false);
  let selected = $state(0);

  onMount(() => {
    const onToggle = () => {
      open = !open;
      selected = 0;
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
        open = false;
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

  async function activate(command: Command) {
    open = false;
    await goto(command.href);
  }
</script>

{#if open}
  <div
    class="palette-backdrop"
    role="presentation"
    onclick={() => (open = false)}
  >
    <div
      class="palette tile"
      role="dialog"
      aria-modal="true"
      aria-label="Command palette"
      tabindex="-1"
      onclick={(event) => event.stopPropagation()}
      onkeydown={(event) => event.stopPropagation()}
    >
      <header>
        <span>Command</span>
        <kbd>Esc</kbd>
      </header>
      <div class="command-list" role="listbox" aria-label="Navigation commands">
        {#each commands as command, index}
          <button
            type="button"
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
    padding: 12vh 1rem 1rem;
    background: rgb(5 7 10 / 0.62);
    backdrop-filter: blur(10px);
  }

  .palette {
    width: min(34rem, 100%);
    padding: 0.65rem;
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
    display: grid;
    gap: 0.3rem;
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
</style>
