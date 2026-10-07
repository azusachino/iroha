<script lang="ts">
  import { onMount } from "svelte";
  import { Copy, KeyRound } from "@lucide/svelte";
  import {
    issueIntakeCredential,
    listIntakeCredentials,
    revokeIntakeCredential,
    type IntakeCredential,
  } from "$lib/api";
  import { formatDate } from "$lib/format";
  import { createAsyncResource } from "$lib/asyncResource.svelte";
  import RetryNotice from "@iroha/shared/theme-ui/components/RetryNotice.svelte";
  import ConfirmDialog from "./ConfirmDialog.svelte";

  const credentialsResource = createAsyncResource<IntakeCredential[]>();
  let name = $state("primary");
  let issued = $state<{ name: string; token: string } | null>(null);
  let copied = $state(false);
  let busy = $state(false);
  let error = $state<string | null>(null);

  let revokeTarget = $state<IntakeCredential | null>(null);
  let confirmOpen = $state(false);

  // Device names become permanent provenance keys (iphone-hae:<name>), so
  // they are normalized as typed rather than rejected on submit.
  function normalizeDeviceName(value: string): string {
    return value
      .toLowerCase()
      .replace(/[\s_]+/g, "-")
      .replace(/[^a-z0-9-]/g, "");
  }

  const active = $derived(
    (credentialsResource.data ?? []).filter(
      (credential) => !credential.revoked_at,
    ),
  );

  async function load() {
    await credentialsResource.run(() => listIntakeCredentials());
  }

  async function issue(event: SubmitEvent) {
    event.preventDefault();
    busy = true;
    error = null;
    copied = false;
    try {
      const result = await issueIntakeCredential(name.trim());
      issued = { name: result.credential.name, token: result.token };
      await load();
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause);
    } finally {
      busy = false;
    }
  }

  async function copy() {
    if (!issued) return;
    await navigator.clipboard.writeText(issued.token);
    copied = true;
  }

  function askRevoke(credential: IntakeCredential) {
    revokeTarget = credential;
    confirmOpen = true;
  }

  async function revoke() {
    if (!revokeTarget) return;
    error = null;
    try {
      await revokeIntakeCredential(revokeTarget.id);
      await load();
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause);
    }
  }

  onMount(() => {
    void load();
  });
</script>

<section class="panel" aria-labelledby="intake-title">
  <header>
    <div>
      <p class="eyebrow"><KeyRound size={14} /> Health Auto Export</p>
      <h2 id="intake-title">Intake tokens</h2>
    </div>
    {#if credentialsResource.ready}<span>{active.length} active</span>{/if}
  </header>
  <p class="panel-note">
    Each device that uploads needs its own token, sent by Health Auto Export as
    <code>Authorization: Bearer &lt;token&gt;</code>. To rotate, issue a new
    token with the same name, update both automations, then revoke the old one.
  </p>

  {#if issued}
    <div class="issued" role="status">
      <strong>Token for “{issued.name}”</strong>
      <small
        >Copy it into Health Auto Export now. Iroha can't show it again.</small
      >
      <code class="token">{issued.token}</code>
      <div class="row">
        <button type="button" onclick={() => void copy()}>
          <Copy size={14} />
          {copied ? "Copied" : "Copy token"}
        </button>
        <button type="button" onclick={() => (issued = null)}>Done</button>
      </div>
    </div>
  {/if}

  <form class="row" onsubmit={issue}>
    <label>
      Device name
      <input
        name="device"
        autocapitalize="none"
        spellcheck="false"
        pattern="[a-z0-9][a-z0-9\-]*"
        title="Lowercase letters, digits, and dashes, starting with a letter or digit"
        aria-describedby="device-hint"
        required
        value={name}
        oninput={(event) => {
          const input = event.currentTarget;
          name = normalizeDeviceName(input.value);
          input.value = name;
        }}
      />
      <small id="device-hint"
        >Lowercase letters, digits, and dashes, like <code>harus-phone</code>.
        Keep the same name when you rotate the token.</small
      >
    </label>
    <button type="submit" disabled={busy}
      >{busy ? "Issuing…" : "Issue token"}</button
    >
  </form>

  {#if credentialsResource.error}<RetryNotice
      message={`Could not load intake tokens: ${credentialsResource.error}`}
      retryLabel="Retry intake tokens"
      onRetry={load}
    />{/if}
  {#if error}<p class="error" role="alert">{error}</p>{/if}

  {#if credentialsResource.loading}
    <p class="muted" role="status">
      {credentialsResource.ready
        ? "Updating intake tokens…"
        : "Loading intake tokens…"}
    </p>
  {/if}
  {#if credentialsResource.ready && credentialsResource.data?.length}
    <ul class="credential-list">
      {#each credentialsResource.data ?? [] as credential (credential.id)}
        <li class:revoked={credential.revoked_at}>
          <div>
            <strong>{credential.name}</strong>
            <small>
              Issued {formatDate(credential.created_at)} · {credential.revoked_at
                ? `revoked ${formatDate(credential.revoked_at)}`
                : credential.last_used_at
                  ? `last used ${formatDate(credential.last_used_at)}`
                  : "never used"}
            </small>
          </div>
          {#if !credential.revoked_at}
            <button type="button" onclick={() => askRevoke(credential)}
              >Revoke</button
            >
          {/if}
        </li>
      {/each}
    </ul>
  {:else if credentialsResource.ready}
    <p class="muted">No tokens yet. Intake is refused until you issue one.</p>
  {/if}
</section>

<ConfirmDialog
  bind:open={confirmOpen}
  message={`Revoke the token for “${revokeTarget?.name ?? ""}”? Uploads using it fail immediately.`}
  confirmLabel="Revoke"
  onConfirm={() => void revoke()}
/>

<style>
  .panel {
    display: grid;
    min-width: 0;
    align-content: start;
    gap: 1rem;
    padding: 1rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
  }

  .panel > header {
    display: flex;
    align-items: end;
    justify-content: space-between;
    gap: 1rem;
  }

  .panel > header > span,
  small,
  .muted,
  .panel-note {
    color: var(--text-muted);
    font-size: 0.76rem;
    line-height: 1.45;
  }

  h2,
  p {
    margin: 0;
  }

  h2 {
    font-size: 1.25rem;
  }

  .eyebrow {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    color: var(--accent);
    font-size: 0.68rem;
    font-weight: 750;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  .row {
    display: flex;
    flex-wrap: wrap;
    align-items: end;
    gap: 0.5rem;
  }

  label {
    display: grid;
    flex: 1 1 10rem;
    gap: 0.3rem;
    font-size: 0.8rem;
  }

  input {
    min-height: 2.4rem;
    padding: 0 0.7rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 4px);
    background: var(--surface);
    color: var(--text);
    font: inherit;
    font-size: 1rem;
  }

  button {
    display: inline-flex;
    min-height: 2.4rem;
    align-items: center;
    gap: 0.35rem;
    padding: 0 0.8rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 4px);
    background: var(--surface);
    color: var(--text);
    font: inherit;
    cursor: pointer;
  }

  button:hover {
    border-color: var(--accent);
    color: var(--accent);
  }

  button:disabled {
    cursor: default;
    opacity: 0.5;
  }

  .issued {
    display: grid;
    gap: 0.5rem;
    padding: 0.85rem;
    border: 1px solid var(--accent);
    border-radius: calc(var(--radius) - 4px);
  }

  .token {
    padding: 0.6rem;
    overflow-wrap: anywhere;
    border-radius: calc(var(--radius) - 6px);
    background: var(--surface-muted, var(--border));
    font-size: 0.85rem;
    user-select: all;
  }

  .credential-list {
    display: grid;
    gap: 0.45rem;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .credential-list li {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    padding: 0.65rem 0;
    border-top: 1px solid var(--border);
  }

  .credential-list li > div {
    display: grid;
    gap: 0.2rem;
  }

  .credential-list li.revoked strong {
    color: var(--text-muted);
    text-decoration: line-through;
  }

  .error {
    color: var(--danger);
    font-size: 0.85rem;
  }
</style>
