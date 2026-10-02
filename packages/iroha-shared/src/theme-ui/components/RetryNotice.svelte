<script lang="ts">
  let { message, onRetry }: { message: string; onRetry: () => void } = $props();
</script>

<div class="retry-notice" role="alert">
  <p>{message}</p>
  <button type="button" onclick={onRetry}>Try again</button>
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
