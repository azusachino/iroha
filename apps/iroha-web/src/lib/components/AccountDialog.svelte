<script lang="ts">
  import { X } from "@lucide/svelte";
  import { Tabs } from "bits-ui";
  import PasskeySettings from "./PasskeySettings.svelte";
  import { ApiError } from "$lib/api";
  import { auth, setDisplayName } from "$lib/auth.svelte";
  import { dialogLightDismiss } from "$lib/dialog";

  let { open = $bindable(false) }: { open?: boolean } = $props();

  let dialog: HTMLDialogElement;
  let tab = $state<string>("profile");
  let displayName = $state("");
  let saving = $state(false);
  let message = $state<{ ok: boolean; text: string } | null>(null);

  $effect(() => {
    if (!dialog) return;
    if (open && !dialog.open) {
      displayName = auth.displayName;
      message = null;
      dialog.showModal();
    } else if (!open && dialog.open) {
      dialog.close();
    }
  });

  async function saveProfile(event: SubmitEvent) {
    event.preventDefault();
    saving = true;
    message = null;
    try {
      await setDisplayName(displayName);
      message = { ok: true, text: "Saved." };
    } catch (cause) {
      message = {
        ok: false,
        text:
          cause instanceof ApiError
            ? cause.message
            : "Couldn't save. Try again.",
      };
    } finally {
      saving = false;
    }
  }
</script>

<dialog
  bind:this={dialog}
  class="account-dialog"
  closedby="any"
  use:dialogLightDismiss
  aria-labelledby="account-title"
  onclose={() => (open = false)}
>
  <header>
    <h2 id="account-title">Account settings</h2>
    <button
      type="button"
      class="icon"
      aria-label="Close account settings"
      onclick={() => (open = false)}
    >
      <X size={18} />
    </button>
  </header>

  <Tabs.Root bind:value={tab} loop>
    <Tabs.List class="account-tabs" aria-label="Account settings sections">
      <Tabs.Trigger value="profile" class="account-tab">Profile</Tabs.Trigger>
      <Tabs.Trigger value="security" class="account-tab">Security</Tabs.Trigger>
    </Tabs.List>

    <Tabs.Content value="profile">
      <form class="panel" onsubmit={saveProfile}>
        <label>
          Username
          <input value={auth.username} disabled />
        </label>
        <label>
          Display name
          <input
            name="display-name"
            maxlength="64"
            placeholder={auth.username}
            bind:value={displayName}
          />
        </label>
        <small>
          Shown in the menu instead of your username. Leave empty to use the
          username.
        </small>
        {#if message}
          <p class:error={!message.ok} role={message.ok ? "status" : "alert"}>
            {message.text}
          </p>
        {/if}
        <button type="submit" class="primary" disabled={saving}>
          {saving ? "Saving…" : "Save"}
        </button>
      </form>
    </Tabs.Content>

    <Tabs.Content value="security">
      {#if open && tab === "security"}
        <div class="panel">
          <PasskeySettings />
        </div>
      {/if}
    </Tabs.Content>
  </Tabs.Root>
</dialog>

<style>
  .account-dialog {
    width: min(100% - 2rem, 30rem);
    /* The app's reset zeroes margins; a modal dialog centres on auto. */
    margin: auto;
    padding: 0;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
  }

  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 1rem 1rem 0.5rem 1.25rem;
  }

  h2 {
    margin: 0;
    font-size: 1.15rem;
  }

  .account-dialog :global(.account-tabs) {
    display: flex;
    gap: 0.25rem;
    padding: 0 1.25rem;
    border-bottom: 1px solid var(--border);
  }

  .account-dialog :global(.account-tab) {
    padding: 0.6rem 0.75rem;
    border: 0;
    border-bottom: 2px solid transparent;
    background: none;
    color: var(--text-muted);
    font: inherit;
    cursor: pointer;
  }

  .account-dialog :global(.account-tab[data-state="active"]) {
    border-bottom-color: var(--accent);
    color: var(--text);
    font-weight: 650;
  }

  .panel {
    display: grid;
    gap: 0.75rem;
    padding: 1.25rem;
  }

  label {
    display: grid;
    gap: 0.3rem;
    font-size: 0.85rem;
  }

  input {
    min-height: 2.6rem;
    padding: 0 0.75rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 4px);
    background: var(--surface);
    color: var(--text);
    font: inherit;
    font-size: 1rem;
  }

  input:disabled {
    color: var(--text-muted);
  }

  small {
    color: var(--text-muted);
    font-size: 0.78rem;
  }

  p {
    margin: 0;
    font-size: 0.85rem;
  }

  .error {
    color: var(--danger);
  }

  .icon {
    display: grid;
    width: 2.2rem;
    height: 2.2rem;
    place-items: center;
    border: 0;
    border-radius: 50%;
    background: none;
    color: var(--text-muted);
    cursor: pointer;
  }

  .primary {
    justify-self: start;
    min-height: 2.5rem;
    padding: 0 1rem;
    border: 1px solid var(--text);
    border-radius: calc(var(--radius) - 4px);
    background: var(--text);
    color: var(--surface);
    font: inherit;
    font-weight: 650;
    cursor: pointer;
  }

  button:focus-visible,
  input:focus-visible {
    outline: 2px solid var(--color-focus);
    outline-offset: 2px;
  }
</style>
