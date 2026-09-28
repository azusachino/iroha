<script lang="ts">
  import { KeyRound } from "@lucide/svelte";
  import { ApiError } from "$lib/api";
  import { auth, loadSession, login, setup } from "$lib/auth.svelte";

  const isSetup = $derived(auth.status === "setup");
  let username = $state("");
  let password = $state("");
  let confirmPassword = $state("");
  let busy = $state(false);
  let error = $state<string | null>(null);

  async function submit(event: SubmitEvent) {
    event.preventDefault();
    error = null;
    if (isSetup && password !== confirmPassword) {
      error = "The two passwords don't match.";
      return;
    }
    busy = true;
    try {
      if (isSetup) await setup(username.trim(), password);
      else await login(username.trim(), password);
      password = "";
      confirmPassword = "";
    } catch (cause) {
      error =
        cause instanceof ApiError
          ? cause.code === "invalid_credentials"
            ? "Wrong username or password."
            : cause.message
          : "Couldn't reach Iroha. Check your connection and try again.";
    } finally {
      busy = false;
    }
  }
</script>

<main class="auth-screen">
  {#if auth.status === "loading"}
    <p class="muted" role="status">Checking your session…</p>
  {:else if auth.status === "error"}
    <div class="auth-card">
      <p class="error" role="alert">Couldn't reach Iroha.</p>
      <button type="button" onclick={() => void loadSession()}>Try again</button
      >
    </div>
  {:else}
    <form class="auth-card" onsubmit={submit}>
      <p class="eyebrow"><KeyRound size={14} /> iroha</p>
      <h1>{isSetup ? "Create the owner account" : "Log in"}</h1>
      {#if isSetup}
        <p class="muted">
          This runs once. Iroha has a single owner, and this screen closes for
          good after the account exists.
        </p>
      {/if}
      <label>
        Username
        <input
          name="username"
          autocomplete="username"
          autocapitalize="none"
          spellcheck="false"
          required
          bind:value={username}
        />
      </label>
      {#if isSetup}
        <small class="muted"
          >Lowercase letters, digits, dots, dashes, or underscores.</small
        >
      {/if}
      <label>
        Password
        <input
          name="password"
          type="password"
          autocomplete={isSetup ? "new-password" : "current-password"}
          minlength={isSetup ? 12 : undefined}
          required
          bind:value={password}
        />
      </label>
      {#if isSetup}
        <label>
          Confirm password
          <input
            name="confirm-password"
            type="password"
            autocomplete="new-password"
            minlength="12"
            required
            bind:value={confirmPassword}
          />
        </label>
        <small class="muted">At least 12 characters.</small>
      {/if}
      {#if error}<p class="error" role="alert">{error}</p>{/if}
      <button type="submit" disabled={busy}>
        {busy
          ? isSetup
            ? "Creating…"
            : "Logging in…"
          : isSetup
            ? "Create account"
            : "Log in"}
      </button>
    </form>
  {/if}
</main>

<style>
  .auth-screen {
    display: grid;
    min-height: 100dvh;
    place-items: center;
    padding: 1.5rem;
  }

  .auth-card {
    display: grid;
    width: min(100%, 24rem);
    gap: 0.75rem;
    padding: 1.5rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
  }

  h1,
  p {
    margin: 0;
  }

  h1 {
    font-size: 1.5rem;
    letter-spacing: -0.03em;
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

  label {
    display: grid;
    gap: 0.3rem;
    font-size: 0.85rem;
  }

  input {
    min-height: 2.75rem;
    padding: 0 0.75rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 4px);
    background: var(--surface);
    color: var(--text);
    font: inherit;
    font-size: 1rem;
  }

  input:focus-visible,
  button:focus-visible {
    outline: 2px solid var(--color-focus);
    outline-offset: 2px;
  }

  button {
    min-height: 2.75rem;
    border: 1px solid var(--accent);
    border-radius: calc(var(--radius) - 4px);
    background: var(--accent);
    color: var(--surface);
    font: inherit;
    font-weight: 650;
    cursor: pointer;
  }

  button:disabled {
    cursor: default;
    opacity: 0.6;
  }

  small,
  .muted {
    color: var(--text-muted);
    font-size: 0.78rem;
    line-height: 1.45;
  }

  .error {
    color: var(--danger);
    font-size: 0.85rem;
  }
</style>
