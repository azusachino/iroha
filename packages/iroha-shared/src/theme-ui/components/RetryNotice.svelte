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
    gap: 1rem;
    border: 1px solid color-mix(in srgb, var(--sport-run) 45%, var(--border));
    padding: 1rem;
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
    padding: 0.45rem 0.75rem;
    background: transparent;
    color: inherit;
    font: inherit;
    font-size: 0.75rem;
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
