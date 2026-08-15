<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    loading,
    ready = true,
    preserveLayout = false,
    label,
    children,
  }: {
    loading: boolean;
    ready?: boolean;
    preserveLayout?: boolean;
    label: string;
    children: Snippet;
  } = $props();
</script>

{#if !ready && !preserveLayout}
  <div class="loading-surface" role="status" aria-live="polite">
    <span class="loading-mark" aria-hidden="true"></span>
    <span>{label}</span>
  </div>
{:else}
  <div
    class="data-surface"
    class:updating={loading}
    class:pending={!ready && loading}
    aria-busy={loading}
  >
    <div
      class="data-content"
      aria-hidden={!ready && loading}
      inert={!ready && loading}
    >
      {@render children()}
    </div>
    {#if !ready && loading}
      <div class="loading-overlay" role="status" aria-live="polite">
        <span class="loading-mark" aria-hidden="true"></span>
        <span>{label}</span>
      </div>
    {:else if loading}
      <span class="update-status" role="status" aria-live="polite"
        >Updating…</span
      >
    {/if}
  </div>
{/if}

<style>
  .loading-surface {
    display: grid;
    min-height: 22rem;
    place-items: center;
    gap: 0.7rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    color: var(--text-muted);
    font-size: 0.82rem;
  }
  .loading-mark {
    width: 2.8rem;
    height: 0.25rem;
    overflow: hidden;
    background: color-mix(in srgb, var(--accent) 18%, var(--border));
  }
  .loading-mark::after {
    display: block;
    width: 45%;
    height: 100%;
    background: var(--accent);
    content: "";
    animation: loading-sweep 1.1s ease-in-out infinite;
  }
  .data-surface {
    position: relative;
    min-width: 0;
  }
  .data-surface.pending {
    min-height: 16rem;
  }
  .data-content {
    min-width: 0;
  }
  .loading-overlay {
    position: absolute;
    inset: 0;
    z-index: 2;
    display: grid;
    place-content: center;
    gap: 0.7rem;
    min-height: 16rem;
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: color-mix(in srgb, var(--bg) 82%, transparent);
    color: var(--text-muted);
    font-size: 0.82rem;
    pointer-events: none;
  }
  .update-status {
    position: absolute;
    top: 0.35rem;
    right: 0.35rem;
    z-index: 2;
    padding: 0.3rem 0.55rem;
    border: 1px solid color-mix(in srgb, var(--accent) 38%, var(--border));
    border-radius: 999px;
    background: color-mix(in srgb, var(--surface) 92%, transparent);
    color: var(--accent);
    font-size: 0.68rem;
    font-weight: 650;
  }
  @keyframes loading-sweep {
    from {
      transform: translateX(-120%);
    }
    to {
      transform: translateX(340%);
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .loading-mark::after {
      animation: none;
    }
  }
</style>
