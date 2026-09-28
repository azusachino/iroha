<script lang="ts">
  import { Eye, EyeOff } from "@lucide/svelte";
  import { ApiError } from "$lib/api";
  import { auth, loadSession, login, setup } from "$lib/auth.svelte";

  const isSetup = $derived(auth.status === "setup");
  let username = $state("");
  let password = $state("");
  let confirmPassword = $state("");
  let showPassword = $state(false);
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
            : cause.code === "rate_limited"
              ? "Too many attempts. Wait a minute and try again."
              : cause.message
          : "Couldn't reach Iroha. Check your connection and try again.";
    } finally {
      busy = false;
    }
  }
</script>

<main class="auth-screen">
  <div class="auth-column">
    <div class="brand">
      <img src="/favicon.svg" alt="" width="40" height="40" />
      <span>iroha</span>
    </div>
    <p class="tagline">Your personal data cockpit</p>

    {#if auth.status === "loading"}
      <p class="muted center" role="status">Checking your session…</p>
    {:else if auth.status === "error"}
      <div class="card">
        <p class="error" role="alert">Couldn't reach Iroha.</p>
        <button type="button" class="primary" onclick={() => void loadSession()}
          >Try again</button
        >
      </div>
    {:else}
      <form class="card" onsubmit={submit}>
        <header class="card-header">
          <h1>{isSetup ? "Welcome" : "Sign in"}</h1>
          <p class="muted">
            {isSetup
              ? "Create the owner account. Iroha has one owner, and this step runs only once."
              : "Sign in to continue to your cockpit."}
          </p>
        </header>

        {#if error}<p class="alert" role="alert">{error}</p>{/if}

        <label>
          Username
          <input
            name="username"
            autocomplete="username"
            autocapitalize="none"
            spellcheck="false"
            placeholder="Enter your username"
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
          <span class="password">
            <input
              name="password"
              type={showPassword ? "text" : "password"}
              autocomplete={isSetup ? "new-password" : "current-password"}
              placeholder="Enter your password"
              minlength={isSetup ? 12 : undefined}
              required
              bind:value={password}
            />
            <button
              type="button"
              class="reveal"
              aria-label={showPassword ? "Hide password" : "Show password"}
              aria-pressed={showPassword}
              onclick={() => (showPassword = !showPassword)}
            >
              {#if showPassword}<EyeOff size={17} />{:else}<Eye
                  size={17}
                />{/if}
            </button>
          </span>
        </label>

        {#if isSetup}
          <label>
            Confirm password
            <input
              name="confirm-password"
              type={showPassword ? "text" : "password"}
              autocomplete="new-password"
              minlength="12"
              required
              bind:value={confirmPassword}
            />
          </label>
          <small class="muted"
            >At least 12 characters. You can add a passkey after signing in.</small
          >
        {/if}

        <button type="submit" class="primary" disabled={busy}>
          {busy
            ? isSetup
              ? "Creating account…"
              : "Signing in…"
            : isSetup
              ? "Create account"
              : "Sign in"}
        </button>
      </form>
    {/if}
  </div>
</main>

<style>
  .auth-screen {
    display: grid;
    min-height: 100dvh;
    place-items: center;
    padding: 1.5rem;
  }

  .auth-column {
    display: grid;
    width: min(100%, 26rem);
    gap: 0.5rem;
  }

  .brand {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.6rem;
    font-size: 1.6rem;
    font-weight: 750;
    letter-spacing: -0.03em;
  }

  .tagline {
    margin: 0 0 1.25rem;
    color: var(--text-muted);
    text-align: center;
  }

  .card {
    display: grid;
    gap: 0.85rem;
    padding: 1.5rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    box-shadow: 0 12px 40px rgb(0 0 0 / 0.06);
  }

  .card-header {
    display: grid;
    gap: 0.35rem;
    margin-bottom: 0.25rem;
    text-align: center;
  }

  h1,
  p {
    margin: 0;
  }

  h1 {
    font-size: 1.35rem;
  }

  label {
    display: grid;
    gap: 0.3rem;
    font-size: 0.85rem;
    font-weight: 600;
  }

  input {
    width: 100%;
    min-height: 2.75rem;
    padding: 0 0.75rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 4px);
    background: var(--surface);
    color: var(--text);
    font: inherit;
    font-size: 1rem;
    font-weight: 400;
  }

  .password {
    position: relative;
    display: block;
  }

  .password input {
    padding-right: 2.75rem;
  }

  .reveal {
    position: absolute;
    top: 50%;
    right: 0.3rem;
    display: grid;
    width: 2.25rem;
    height: 2.25rem;
    place-items: center;
    border: 0;
    border-radius: calc(var(--radius) - 6px);
    background: none;
    color: var(--text-muted);
    cursor: pointer;
    transform: translateY(-50%);
  }

  .primary {
    min-height: 2.75rem;
    margin-top: 0.25rem;
    border: 1px solid var(--text);
    border-radius: calc(var(--radius) - 4px);
    background: var(--text);
    color: var(--surface);
    font: inherit;
    font-weight: 650;
    cursor: pointer;
  }

  .primary:disabled {
    cursor: default;
    opacity: 0.6;
  }

  input:focus-visible,
  button:focus-visible {
    outline: 2px solid var(--color-focus);
    outline-offset: 2px;
  }

  .alert {
    padding: 0.65rem 0.75rem;
    border: 1px solid color-mix(in srgb, var(--danger) 40%, transparent);
    border-radius: calc(var(--radius) - 4px);
    background: color-mix(in srgb, var(--danger) 8%, transparent);
    color: var(--danger);
    font-size: 0.85rem;
  }

  small,
  .muted {
    color: var(--text-muted);
    font-size: 0.8rem;
    font-weight: 400;
    line-height: 1.45;
  }

  .center {
    text-align: center;
  }

  .error {
    color: var(--danger);
  }
</style>
