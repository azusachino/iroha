<script lang="ts">
  import { tick } from "svelte";
  let {
    message,
    onRetry,
    retryLabel = "Try again",
    focusTarget,
  }: {
    message: string;
    onRetry: () => void | Promise<void>;
    retryLabel?: string;
    focusTarget?: HTMLElement;
  } = $props();

  async function retry(event: MouseEvent): Promise<void> {
    const origin = event.currentTarget as HTMLElement;
    await onRetry();
    await tick();
    if (!focusTarget?.isConnected) return;
    // Do not take focus back if the user moved to another control while waiting.
    if (
      document.activeElement !== document.body &&
      document.activeElement !== origin
    )
      return;
    const retryButton = [
      ...focusTarget.querySelectorAll<HTMLButtonElement>("button"),
    ].find((button) => button.dataset.retryLabel === retryLabel);
    (retryButton ?? focusTarget).focus({ preventScroll: true });
  }
</script>

<div class="retry-notice" role="alert">
  <p>{message}</p>
  <button type="button" data-retry-label={retryLabel} onclick={retry}
    >{retryLabel}</button
  >
</div>

<style>
  .retry-notice {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-4);
    border: 1px solid color-mix(in srgb, var(--sport-run) 45%, var(--border));
    padding: var(--space-4);
    color: var(--sport-run);
  }

  .retry-notice p {
    margin: 0;
    overflow-wrap: anywhere;
  }

  button {
    flex: 0 0 auto;
    border: 1px solid currentColor;
    border-radius: var(--radius);
    padding: var(--space-2) var(--space-3);
    background: transparent;
    color: inherit;
    font: inherit;
    font-size: var(--type-caption);
    cursor: pointer;
  }

  button:hover {
    background: color-mix(in srgb, currentColor 10%, transparent);
  }

  @media (max-width: 640px) {
    .retry-notice {
      align-items: stretch;
      flex-direction: column;
    }
  }
</style>
