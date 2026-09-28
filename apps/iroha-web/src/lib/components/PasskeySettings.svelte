<script lang="ts">
  import { onMount } from "svelte";
  import { KeyRound, Plus, Trash2 } from "@lucide/svelte";
  import {
    ApiError,
    deletePasskey,
    listPasskeys,
    reauthenticate,
    renamePasskey,
    type Passkey,
  } from "$lib/api";
  import { auth } from "$lib/auth.svelte";
  import { formatDate } from "$lib/format";
  import {
    PasskeyCancelled,
    passkeysSupported,
    registerPasskey,
  } from "$lib/passkey";

  // The server accepts a password confirmation for 5 minutes; mirror a
  // slightly shorter window so the UI asks again before it would fail.
  const REAUTH_MS = 4.5 * 60 * 1000;

  let passkeys = $state<Passkey[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);
  let notice = $state<string | null>(null);
  let confirmedUntil = 0;

  // A pending action waits for the password prompt to succeed.
  let pending = $state<null | { label: string; run: () => Promise<void> }>(
    null,
  );
  let password = $state("");
  let busy = $state(false);
  let newName = $state(defaultName());
  let editing = $state<string | null>(null);
  let editName = $state("");

  const supported = passkeysSupported();

  function defaultName(): string {
    if (typeof navigator === "undefined") return "Passkey";
    const ua = navigator.userAgent;
    if (/iPhone/.test(ua)) return "iPhone";
    if (/iPad/.test(ua)) return "iPad";
    if (/Android/.test(ua)) return "Android";
    if (/Mac OS X/.test(ua)) return "Mac";
    if (/Windows/.test(ua)) return "Windows";
    return "Passkey";
  }

  function describe(cause: unknown): string {
    if (cause instanceof PasskeyCancelled) return cause.message;
    if (cause instanceof ApiError) return cause.message;
    return "Something went wrong. Try again.";
  }

  async function load() {
    loading = true;
    try {
      passkeys = await listPasskeys();
    } catch (cause) {
      error = describe(cause);
    } finally {
      loading = false;
    }
  }

  // Runs action now if the password was confirmed recently; otherwise asks
  // for it first.
  function withConfirmation(label: string, action: () => Promise<void>) {
    error = null;
    notice = null;
    if (Date.now() < confirmedUntil) {
      void execute(action);
    } else {
      password = "";
      pending = { label, run: action };
    }
  }

  async function execute(action: () => Promise<void>) {
    busy = true;
    try {
      await action();
    } catch (cause) {
      if (cause instanceof ApiError && cause.code === "reauth_required") {
        confirmedUntil = 0;
      }
      error = describe(cause);
    } finally {
      busy = false;
    }
  }

  async function confirmPassword(event: SubmitEvent) {
    event.preventDefault();
    if (!pending) return;
    busy = true;
    error = null;
    try {
      await reauthenticate(password);
      confirmedUntil = Date.now() + REAUTH_MS;
      password = "";
      const action = pending.run;
      pending = null;
      busy = false;
      await execute(action);
    } catch (cause) {
      error =
        cause instanceof ApiError && cause.code === "invalid_credentials"
          ? "Wrong password."
          : describe(cause);
    } finally {
      busy = false;
    }
  }

  function add() {
    const name = newName.trim() || "Passkey";
    withConfirmation("add a passkey", async () => {
      const created = await registerPasskey(name);
      passkeys = [...passkeys, created];
      notice = `Added “${created.name}”. You can now sign in with it.`;
    });
  }

  function remove(passkey: Passkey) {
    withConfirmation(`remove “${passkey.name}”`, async () => {
      await deletePasskey(passkey.id);
      passkeys = passkeys.filter((item) => item.id !== passkey.id);
      notice = `Removed “${passkey.name}”.`;
    });
  }

  async function saveName(passkey: Passkey) {
    const name = editName.trim();
    if (!name || name === passkey.name) {
      editing = null;
      return;
    }
    await execute(async () => {
      await renamePasskey(passkey.id, name);
      passkeys = passkeys.map((item) =>
        item.id === passkey.id ? { ...item, name } : item,
      );
      editing = null;
    });
  }

  onMount(() => {
    void load();
  });
</script>

<div class="passkeys">
  <div class="intro">
    <KeyRound size={18} />
    <div>
      <strong>Passkeys</strong>
      <p>
        Sign in with Face ID, Touch ID, or a security key instead of your
        password. Your password keeps working.
      </p>
    </div>
  </div>

  {#if !auth.passkeysEnabled}
    <p class="muted">Passkeys aren't configured on this server.</p>
  {:else if !supported}
    <p class="muted">This browser doesn't support passkeys.</p>
  {:else}
    {#if loading}
      <p class="muted" role="status">Loading passkeys…</p>
    {:else if passkeys.length === 0}
      <p class="muted">No passkeys yet.</p>
    {:else}
      <ul>
        {#each passkeys as passkey (passkey.id)}
          <li>
            {#if editing === passkey.id}
              <form
                class="rename"
                onsubmit={(event) => {
                  event.preventDefault();
                  void saveName(passkey);
                }}
              >
                <input
                  aria-label="Passkey name"
                  maxlength="64"
                  bind:value={editName}
                />
                <button type="submit" disabled={busy}>Save</button>
                <button type="button" onclick={() => (editing = null)}
                  >Cancel</button
                >
              </form>
            {:else}
              <div>
                <strong>{passkey.name}</strong>
                <small
                  >Added {formatDate(passkey.created_at)} · {passkey.last_used_at
                    ? `last used ${formatDate(passkey.last_used_at)}`
                    : "never used"}</small
                >
              </div>
              <div class="row-actions">
                <button
                  type="button"
                  onclick={() => {
                    editing = passkey.id;
                    editName = passkey.name;
                  }}>Rename</button
                >
                <button
                  type="button"
                  class="danger"
                  aria-label={`Remove ${passkey.name}`}
                  disabled={busy}
                  onclick={() => remove(passkey)}><Trash2 size={15} /></button
                >
              </div>
            {/if}
          </li>
        {/each}
      </ul>
    {/if}

    {#if pending}
      <form class="confirm" onsubmit={confirmPassword}>
        <label>
          Enter your password to {pending.label}
          <input
            type="password"
            autocomplete="current-password"
            required
            bind:value={password}
          />
        </label>
        <div class="row-actions">
          <button type="submit" class="primary" disabled={busy}
            >{busy ? "Checking…" : "Continue"}</button
          >
          <button type="button" onclick={() => (pending = null)}>Cancel</button>
        </div>
      </form>
    {:else}
      <form
        class="add"
        onsubmit={(event) => {
          event.preventDefault();
          add();
        }}
      >
        <label>
          Name for the new passkey
          <input maxlength="64" bind:value={newName} />
        </label>
        <button type="submit" class="primary" disabled={busy}>
          <Plus size={15} /> Add a passkey
        </button>
      </form>
    {/if}
  {/if}

  {#if notice}<p class="notice" role="status">{notice}</p>{/if}
  {#if error}<p class="error" role="alert">{error}</p>{/if}
</div>

<style>
  .passkeys {
    display: grid;
    gap: 0.9rem;
  }

  .intro {
    display: flex;
    gap: 0.6rem;
    color: var(--text);
  }

  .intro p,
  p {
    margin: 0.15rem 0 0;
    color: var(--text-muted);
    font-size: 0.82rem;
    line-height: 1.45;
  }

  ul {
    display: grid;
    gap: 0;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  li {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    padding: 0.65rem 0;
    border-top: 1px solid var(--border);
  }

  li > div:first-child {
    display: grid;
    gap: 0.15rem;
    min-width: 0;
  }

  small,
  .muted {
    color: var(--text-muted);
    font-size: 0.78rem;
  }

  .row-actions {
    display: flex;
    gap: 0.4rem;
  }

  form.add,
  form.confirm {
    display: grid;
    gap: 0.6rem;
    padding-top: 0.75rem;
    border-top: 1px solid var(--border);
  }

  form.rename {
    display: flex;
    width: 100%;
    gap: 0.4rem;
  }

  label {
    display: grid;
    gap: 0.3rem;
    font-size: 0.85rem;
  }

  input {
    min-height: 2.5rem;
    padding: 0 0.7rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 4px);
    background: var(--surface);
    color: var(--text);
    font: inherit;
    font-size: 1rem;
  }

  form.rename input {
    flex: 1;
  }

  button {
    display: inline-flex;
    min-height: 2.4rem;
    align-items: center;
    justify-content: center;
    gap: 0.35rem;
    padding: 0 0.75rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 4px);
    background: var(--surface);
    color: var(--text);
    font: inherit;
    font-size: 0.85rem;
    cursor: pointer;
  }

  button.primary {
    justify-self: start;
    border-color: var(--text);
    background: var(--text);
    color: var(--surface);
    font-weight: 650;
  }

  button.danger {
    color: var(--danger);
  }

  button:disabled {
    cursor: default;
    opacity: 0.55;
  }

  button:focus-visible,
  input:focus-visible {
    outline: 2px solid var(--color-focus);
    outline-offset: 2px;
  }

  .notice {
    color: var(--color-positive);
  }

  .error {
    color: var(--danger);
  }
</style>
