<script lang="ts">
  import { Eye, EyeOff, KeyRound, Lock, X } from "@lucide/svelte";
  import { ApiError } from "$lib/api";
  import { auth, login, passkeyLogin, setup } from "$lib/auth.svelte";
  import { PasskeyCancelled, passkeysSupported } from "$lib/passkey";

  let { open = $bindable(false) }: { open?: boolean } = $props();

  let dialog = $state<HTMLDialogElement>();
  const isSetup = $derived(auth.status === "setup");
  let username = $state("");
  let password = $state("");
  let confirmPassword = $state("");
  let showPassword = $state(false);
  let showPasswordForm = $state(false);
  let busy = $state(false);
  let error = $state<string | null>(null);

  const offerPasskey = $derived(
    !isSetup && auth.passkeysEnabled && passkeysSupported(),
  );

  $effect(() => {
    if (!dialog) return;
    if (open && !dialog.open) {
      error = null;
      password = "";
      confirmPassword = "";
      showPasswordForm = !offerPasskey;
      dialog.showModal();
    } else if (!open && dialog.open) {
      dialog.close();
    }
  });

  function close() {
    open = false;
  }

  async function usePasskey() {
    error = null;
    busy = true;
    try {
      await passkeyLogin();
      open = false;
    } catch (cause) {
      error =
        cause instanceof PasskeyCancelled
          ? null
          : cause instanceof ApiError
            ? cause.message
            : "Couldn't sign in with a passkey. Try again or use your password.";
    } finally {
      busy = false;
    }
  }

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
      open = false;
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

<dialog
  bind:this={dialog}
  class="auth-dialog"
  onclose={() => (open = false)}
  aria-labelledby="auth-dialog-title"
>
  <div class="auth-dialog-content">
    <header class="dialog-header">
      <div class="dialog-title-row">
        <h2 id="auth-dialog-title">
          {isSetup ? "Create owner account" : "Sign in to Iroha"}
        </h2>
        <button
          type="button"
          class="icon-btn close-btn"
          aria-label="Close dialog"
          onclick={close}
        >
          <X size={18} />
        </button>
      </div>
      <p class="muted">
        {isSetup
          ? "Create the initial owner account. This step runs only once."
          : "Sign in as the owner to unlock the full personal cockpit."}
      </p>
    </header>

    {#if error}
      <div class="alert" role="alert">{error}</div>
    {/if}

    {#if offerPasskey}
      <div class="passkey-section">
        <button
          type="button"
          class="passkey-primary-btn"
          disabled={busy}
          onclick={() => void usePasskey()}
        >
          <KeyRound size={20} />
          <span>Sign in with a passkey</span>
        </button>
      </div>
    {/if}

    {#if offerPasskey && !showPasswordForm && !isSetup}
      <div class="divider">
        <span>or</span>
      </div>
      <button
        type="button"
        class="secondary-toggle-btn"
        onclick={() => (showPasswordForm = true)}
      >
        <Lock size={15} />
        <span>Use username & password</span>
      </button>
    {/if}

    {#if showPasswordForm || isSetup || !offerPasskey}
      {#if offerPasskey && !isSetup}
        <div class="divider">
          <span>or use password</span>
        </div>
      {/if}

      <form class="password-form" onsubmit={submit}>
        <label>
          <span>Username</span>
          <input
            name="username"
            type="text"
            autocomplete="username"
            autocapitalize="none"
            spellcheck="false"
            placeholder="Username"
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
          <span>Password</span>
          <span class="password-input-wrap">
            <input
              name="password"
              type={showPassword ? "text" : "password"}
              autocomplete={isSetup ? "new-password" : "current-password"}
              placeholder="Password"
              minlength={isSetup ? 12 : undefined}
              required
              bind:value={password}
            />
            <button
              type="button"
              class="reveal-btn"
              aria-label={showPassword ? "Hide password" : "Show password"}
              onclick={() => (showPassword = !showPassword)}
            >
              {#if showPassword}<EyeOff size={16} />{:else}<Eye
                  size={16}
                />{/if}
            </button>
          </span>
        </label>

        {#if isSetup}
          <label>
            <span>Confirm password</span>
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
            >At least 12 characters. You can register passkeys once signed in.</small
          >
        {/if}

        <button type="submit" class="submit-btn" disabled={busy}>
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
</dialog>

<style>
  .auth-dialog {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    padding: 0;
    width: min(92vw, 24rem);
    box-shadow: 0 16px 48px rgb(0 0 0 / 0.35);
  }

  .auth-dialog::backdrop {
    background: rgb(0 0 0 / 0.6);
    backdrop-filter: blur(4px);
  }

  .auth-dialog-content {
    display: grid;
    gap: 1.1rem;
    padding: 1.5rem;
  }

  .dialog-header {
    display: grid;
    gap: 0.35rem;
  }

  .dialog-title-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
  }

  h2 {
    margin: 0;
    font-size: 1.25rem;
    font-weight: 700;
  }

  .muted {
    margin: 0;
    font-size: 0.85rem;
    color: var(--text-muted);
  }

  .icon-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 2rem;
    height: 2rem;
    border: 0;
    border-radius: 50%;
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
  }

  .icon-btn:hover {
    color: var(--text);
    background: var(--surface-2);
  }

  .alert {
    padding: 0.65rem 0.85rem;
    border: 1px solid color-mix(in srgb, var(--danger) 40%, var(--border));
    border-radius: calc(var(--radius) - 4px);
    background: color-mix(in srgb, var(--danger) 12%, var(--surface));
    color: var(--danger);
    font-size: 0.85rem;
  }

  .passkey-section {
    display: grid;
  }

  .passkey-primary-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 0.65rem;
    width: 100%;
    min-height: 2.85rem;
    padding: 0 1.25rem;
    border: 1px solid var(--accent);
    border-radius: calc(var(--radius) - 2px);
    background: var(--accent);
    color: var(--color-canvas, #090d16);
    font-size: 0.95rem;
    font-weight: 700;
    cursor: pointer;
    box-shadow: 0 4px 14px color-mix(in srgb, var(--accent) 30%, transparent);
    transition:
      transform var(--motion-micro),
      box-shadow var(--motion-micro);
  }

  .passkey-primary-btn:hover:not(:disabled) {
    box-shadow: 0 6px 20px color-mix(in srgb, var(--accent) 45%, transparent);
  }

  .passkey-primary-btn:disabled,
  .submit-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .divider {
    position: relative;
    text-align: center;
    margin: 0.2rem 0;
  }

  .divider::before {
    content: "";
    position: absolute;
    top: 50%;
    left: 0;
    right: 0;
    border-top: 1px solid var(--border);
  }

  .divider span {
    position: relative;
    padding: 0 0.65rem;
    background: var(--surface);
    color: var(--text-muted);
    font-size: 0.78rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .secondary-toggle-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    min-height: 2.4rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 2px);
    background: var(--surface-2);
    color: var(--text);
    font-size: 0.88rem;
    font-weight: 600;
    cursor: pointer;
  }

  .secondary-toggle-btn:hover {
    border-color: color-mix(in srgb, var(--accent) 40%, var(--border));
  }

  .password-form {
    display: grid;
    gap: 0.85rem;
  }

  label {
    display: grid;
    gap: 0.35rem;
    font-size: 0.84rem;
    font-weight: 600;
  }

  input {
    width: 100%;
    min-height: 2.5rem;
    padding: 0 0.75rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 4px);
    background: var(--surface-2);
    color: var(--text);
    font: inherit;
    font-size: 0.92rem;
  }

  .password-input-wrap {
    position: relative;
    display: flex;
    align-items: center;
  }

  .password-input-wrap input {
    padding-right: 2.5rem;
  }

  .reveal-btn {
    position: absolute;
    right: 0.4rem;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 2rem;
    height: 2rem;
    border: 0;
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
  }

  .reveal-btn:hover {
    color: var(--text);
  }

  .submit-btn {
    min-height: 2.5rem;
    margin-top: 0.25rem;
    border: 1px solid var(--border);
    border-radius: calc(var(--radius) - 4px);
    background: var(--surface-2);
    color: var(--text);
    font-size: 0.92rem;
    font-weight: 650;
    cursor: pointer;
  }

  .submit-btn:hover:not(:disabled) {
    border-color: var(--accent);
    color: var(--accent);
  }
</style>
