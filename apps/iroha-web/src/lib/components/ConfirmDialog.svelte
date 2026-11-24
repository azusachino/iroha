<script lang="ts">
  import { dialogLightDismiss } from "$lib/dialog";

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

  let dialog = $state<HTMLDialogElement>();
  let cancelButton = $state<HTMLButtonElement>();

  $effect(() => {
    if (!dialog) return;
    if (open && !dialog.open) {
      dialog.returnValue = "";
      dialog.showModal();
      cancelButton?.focus();
    } else if (!open && dialog.open) {
      dialog.close();
    }
  });

  function handleClose() {
    const confirmed = dialog?.returnValue === "confirm";
    if (dialog) dialog.returnValue = "";
    open = false;
    if (confirmed) onConfirm();
  }
</script>

<dialog
  bind:this={dialog}
  class="confirm-dialog tile"
  role="alertdialog"
  aria-label={message}
  closedby="any"
  use:dialogLightDismiss
  onclose={handleClose}
>
  <p>{message}</p>
  <form method="dialog" class="confirm-actions">
    <button type="submit" value="cancel" bind:this={cancelButton}>
      {cancelLabel}
    </button>
    <button type="submit" value="confirm" class="danger">
      {confirmLabel}
    </button>
  </form>
</dialog>

<style>
  .confirm-dialog {
    margin: auto;
    width: min(24rem, calc(100% - 2rem));
    padding: 1.1rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    box-shadow: var(--tile-shadow);
  }

  .confirm-dialog[open] {
    display: grid;
    gap: 1rem;
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
