<script lang="ts">
  import type { Snippet } from "svelte";
  import type { AsyncResource } from "$lib/asyncResource.svelte";

  // Takes the AsyncResource(s) a route is rendering directly, rather than
  // loading/ready booleans a caller computes by hand -- that used to let
  // every route reinvent (and often get wrong) its own "is this ready"
  // logic. See asyncResource.svelte.ts.
  let {
    resource,
    preserveLayout = false,
    label,
    children,
  }: {
    resource: AsyncResource<unknown> | AsyncResource<unknown>[];
    preserveLayout?: boolean;
    label: string;
    children: Snippet;
  } = $props();

  const resources = $derived(Array.isArray(resource) ? resource : [resource]);
  const loading = $derived(resources.some((r) => r.loading));
  // Each resource's own `ready` only ever goes false -> true once and stays
  // true (see asyncResource.svelte.ts), so this conjunction is sticky too --
  // no separate latch needed here.
  const ready = $derived(resources.every((r) => r.ready));
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
    top: 0.65rem;
    right: 0.65rem;
    z-index: 2;
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.4rem 0.65rem;
    border: 1px solid color-mix(in srgb, var(--accent) 38%, var(--border));
    border-radius: 999px;
    background: var(--surface);
    color: var(--accent);
    font-size: 0.68rem;
    font-weight: 650;
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
