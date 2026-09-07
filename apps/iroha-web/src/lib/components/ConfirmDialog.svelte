<script lang="ts">
  import { tick } from "svelte";

  let {
    open = $bindable(false),
    message,
    confirmLabel = "Delete",
    cancelLabel = "Cancel",
    onConfirm,
  }: {
    open?: boolean;
    message: string;
    confirmLabel?: string;
    cancelLabel?: string;
    onConfirm: () => void;
  } = $props();

  let dialog = $state<HTMLDivElement>();
  let cancelButton = $state<HTMLButtonElement>();
  let previouslyFocused: HTMLElement | null = null;

  $effect(() => {
    if (!open) return;
    previouslyFocused =
      document.activeElement instanceof HTMLElement
        ? document.activeElement
        : null;
    void tick().then(() => cancelButton?.focus());
  });

  function close() {
    open = false;
    const target = previouslyFocused;
    previouslyFocused = null;
    target?.focus();
  }

  function confirm() {
    close();
    onConfirm();
  }

  function onKeydown(event: KeyboardEvent) {
    if (event.key === "Escape") {
      event.preventDefault();
      close();
      return;
    }
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
  <div class="confirm-backdrop" role="presentation" onclick={close}>
    <div
      class="confirm-dialog tile"
      bind:this={dialog}
      role="alertdialog"
      aria-modal="true"
      aria-label={message}
      tabindex="-1"
      onclick={(event) => event.stopPropagation()}
      onkeydown={onKeydown}
    >
      <p>{message}</p>
      <div class="confirm-actions">
        <button type="button" bind:this={cancelButton} onclick={close}
          >{cancelLabel}</button
        >
        <button type="button" class="danger" onclick={confirm}
          >{confirmLabel}</button
        >
      </div>
    </div>
  </div>
{/if}

<style>
  .confirm-backdrop {
    position: fixed;
    inset: 0;
    z-index: 40;
    display: grid;
    place-items: center;
    padding: 1rem;
    background: rgb(5 7 10 / 0.62);
    backdrop-filter: blur(10px);
  }

  .confirm-dialog {
    display: grid;
    gap: 1rem;
    width: min(24rem, 100%);
    padding: 1.1rem;
    background: var(--glass-surface);
    backdrop-filter: blur(var(--glass-blur)) saturate(var(--glass-saturate));
    box-shadow: var(--glass-highlight), var(--tile-shadow);
  }

  @media (prefers-reduced-transparency: reduce) {
    .confirm-dialog {
      background: var(--surface);
      backdrop-filter: none;
    }
  }

  p {
    margin: 0;
    color: var(--text);
    font-size: 0.92rem;
  }

  .confirm-actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
  }

  button {
    min-height: 2.2rem;
    padding: 0 0.9rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 4px);
    background: var(--surface);
    color: var(--text);
    font: inherit;
    cursor: pointer;
  }

  button:hover {
    border-color: var(--accent);
  }

  button.danger {
    border-color: var(--danger);
    color: var(--danger);
  }

  button.danger:hover {
    background: color-mix(in srgb, var(--danger) 12%, transparent);
  }
</style>
