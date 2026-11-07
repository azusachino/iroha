<script lang="ts">
  import type { Snippet } from "svelte";
  import type { DesignLanguage } from "../theme/themes";
  import { RefreshCw } from "@lucide/svelte";

  let {
    title,
    eyebrow = "Period",
    ariaLabel = "Period controls",
    appearance = "grapher",
    onRefresh,
    refreshing = false,
    children,
  }: {
    title?: string;
    eyebrow?: string;
    ariaLabel?: string;
    appearance?: DesignLanguage;
    onRefresh?: () => void;
    refreshing?: boolean;
    children: Snippet;
  } = $props();
</script>

<section
  class="period-toolbar"
  data-appearance={appearance}
  aria-label={ariaLabel}
>
  <div class="period-slot">
    {@render children()}
    {#if onRefresh}
      <button
        type="button"
        class="period-refresh-btn"
        aria-label="Refresh view"
        onclick={onRefresh}
        disabled={refreshing}
      >
        <RefreshCw size={14} class={refreshing ? "spin" : ""} />
        <span>Refresh</span>
      </button>
    {/if}
  </div>
</section>

<style>
  .period-toolbar {
    display: flex;
    min-width: 0;
    min-height: 3.25rem;
    align-items: center;
    justify-content: flex-end;
    gap: var(--space-3);
    padding: var(--space-2) var(--space-4);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
  }

  .period-toolbar[data-appearance="grapher"] {
    border-radius: 2px;
  }

  .period-slot {
    display: flex;
    min-width: 0;
    width: 100%;
    align-items: center;
    justify-content: flex-end;
    gap: var(--space-2);
  }

  .period-refresh-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    height: 2rem;
    padding: 0 0.65rem;
    border: 1px solid var(--border);
    border-radius: var(--radius, 4px);
    background: var(--surface-2, transparent);
    color: var(--text-muted);
    font: inherit;
    font-size: 0.78rem;
    font-weight: 600;
    cursor: pointer;
    white-space: nowrap;
    transition: color 0.15s ease, border-color 0.15s ease, background 0.15s ease;
  }

  .period-refresh-btn:hover:not(:disabled) {
    color: var(--text);
    border-color: var(--accent);
    background: color-mix(in srgb, var(--accent) 8%, var(--surface-2));
  }

  .period-refresh-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  :global(.period-refresh-btn .spin) {
    animation: period-spin 0.8s linear infinite;
  }

  @keyframes period-spin {
    from {
      transform: rotate(0deg);
    }
    to {
      transform: rotate(360deg);
    }
  }

  @media (max-width: 768px) {
    .period-toolbar {
      align-items: stretch;
      flex-direction: column;
      padding: var(--space-3);
    }

    .period-slot {
      width: 100%;
      justify-content: stretch;
      flex-wrap: wrap;
    }

    .period-slot :global(.period-controls) {
      width: 100%;
    }

    .period-refresh-btn {
      width: 100%;
      justify-content: center;
    }
  }
</style>
