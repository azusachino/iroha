<script lang="ts">
  import type { RouteFeatureCollection } from "$lib/api";
  import RoutesMap from "$lib/components/RoutesMap.svelte";

  let {
    routes,
    loading,
    error,
    onLoad,
  }: {
    routes: RouteFeatureCollection | null;
    loading: boolean;
    error: string | null;
    onLoad: () => void;
  } = $props();
</script>

{#if loading}
  <p class="route-footprint-status">Loading route footprint…</p>
{:else if error}
  <div class="route-footprint-error">
    <p>Routes could not be loaded.</p>
    <button type="button" onclick={onLoad}>Try again</button>
  </div>
{:else if routes && routes.features.length}
  <RoutesMap data={routes} />
{:else}
  <p class="route-footprint-status">No routes recorded yet.</p>
{/if}

<style>
  .route-footprint-status,
  .route-footprint-error {
    color: var(--text-muted);
    font-size: 0.84rem;
    line-height: 1.5;
  }

  .route-footprint-error {
    display: grid;
    gap: 0.75rem;
    justify-items: start;
    padding: 1rem 0;
  }

  .route-footprint-error p {
    margin: 0;
  }

  button {
    border: 1px solid var(--border);
    border-radius: 999px;
    padding: 0.5rem 0.8rem;
    background: var(--surface);
    color: var(--text);
    font: inherit;
    font-size: 0.78rem;
    font-weight: 650;
    cursor: pointer;
  }

  button:hover {
    border-color: var(--accent);
    color: var(--accent);
  }
</style>
