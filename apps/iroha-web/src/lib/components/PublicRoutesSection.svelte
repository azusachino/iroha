<script lang="ts">
  import { formatSport } from "$lib/format";
  import { cityLabel } from "$lib/public/showcase-helpers";
  import type { CityGroup } from "$lib/public/aggregate";
  import type { RouteFeature } from "$lib/public/types";
  import PanelFrame from "@iroha/shared/components/PanelFrame.svelte";

  let {
    routesLoading,
    filteredRoutes,
    mappedRoutes,
    cityGroups,
    maxRunCount,
    hasPendingLocations,
    selectedYear,
    sportFilter,
    cityFilter,
    onToggleCity,
    onClearCity,
    sectionEl = $bindable<HTMLElement | undefined>(undefined),
  }: {
    routesLoading: boolean;
    filteredRoutes: RouteFeature[];
    mappedRoutes: RouteFeature[];
    cityGroups: CityGroup[];
    maxRunCount: number;
    hasPendingLocations: boolean;
    selectedYear: string;
    sportFilter: string | null;
    cityFilter: string | null;
    onToggleCity: (city: string) => void;
    onClearCity: () => void;
    sectionEl?: HTMLElement;
  } = $props();
</script>

<section class="section-heading" bind:this={sectionEl}>
  <h2>Routes &amp; cities</h2>
  {#if hasPendingLocations}
    <p class="muted small">Some locations are waiting for geocoding.</p>
  {/if}
</section>

{#if routesLoading}
  <div class="routes-grid" aria-busy="true">
    <PanelFrame label="Route map">
      <div class="map-wrap map-loading">
        <p class="muted">Loading routes map…</p>
      </div>
    </PanelFrame>
    <PanelFrame label="Geography selection">
      <div class="cities cities-loading">
        <p class="muted">Loading geography data…</p>
      </div>
    </PanelFrame>
  </div>
{:else if filteredRoutes.length === 0}
  <p class="muted">No routes recorded yet.</p>
{:else}
  <div class="routes-grid">
    <PanelFrame label="Route map">
      <div class="map-wrap">
        {#await import("$lib/components/RoutesMap.svelte") then { default: RoutesMap }}
          <RoutesMap
            data={{ type: "FeatureCollection", features: mappedRoutes }}
          />
        {/await}
      </div>
    </PanelFrame>
    <PanelFrame label="Geography selection">
      <div class="cities">
        <p class="muted small">
          {selectedYear} · {sportFilter
            ? formatSport(sportFilter)
            : "All sports"} · {cityFilter
            ? cityLabel(cityFilter)
            : "All cities"} (geography only)
        </p>
        <div class="cities-head">
          <h3>
            {sportFilter
              ? `${formatSport(sportFilter)} cities`
              : "Cities visited"}
          </h3>
          {#if cityFilter}
            <button type="button" class="clear-filter" onclick={onClearCity}>
              Show all
            </button>
          {/if}
        </div>
        {#if cityGroups.length === 0}
          <p class="muted small">No route coordinates for this selection.</p>
        {:else}
          <div class="city-grid">
            {#each cityGroups as group (group.city)}
              {@const intensity = group.runCount / maxRunCount}
              <button
                type="button"
                class="city-card"
                class:selected={cityFilter === group.city}
                style={`--intensity: ${intensity}`}
                onclick={() => onToggleCity(group.city)}
              >
                <div class="city-name">
                  {cityLabel(group.city, group.status)}
                </div>
                <div class="city-sports">
                  {Array.from(group.sports)
                    .map((s) => formatSport(s))
                    .join(", ")}
                </div>
                <div class="city-count">
                  {group.runCount}
                  {group.runCount === 1 ? "run" : "runs"}
                  {#if group.count > group.runCount}
                    <span class="muted small">
                      (+{group.count - group.runCount} other)
                    </span>
                  {/if}
                </div>
              </button>
            {/each}
          </div>
        {/if}
      </div>
    </PanelFrame>
  </div>
{/if}

<style>
  .small {
    font-size: var(--type-label);
  }
  .section-heading {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 1rem;
    margin: 1.5rem 0 0.75rem;
  }
  .section-heading h2 {
    margin: 0;
    font-size: var(--type-title);
    letter-spacing: -0.02em;
  }
  .clear-filter {
    margin-top: 0.4rem;
    border: none;
    background: none;
    color: var(--accent);
    font-size: var(--type-label);
    cursor: pointer;
    text-decoration: underline;
  }
  .routes-grid {
    display: grid;
    grid-template-columns: 2fr 1fr;
    gap: 1rem;
  }
  .map-wrap {
    min-height: 20rem;
    padding: 0.5rem;
  }
  .map-loading,
  .cities-loading {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 18rem;
  }
  .cities {
    min-width: 0;
    max-height: 24rem;
    overflow-y: auto;
  }
  .cities-head {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 0.5rem;
    margin-bottom: 0.75rem;
  }
  .cities-head h3 {
    margin: 0;
    font-size: var(--type-label);
  }
  .city-grid {
    display: grid;
    gap: 0.5rem;
  }
  .city-card {
    text-align: left;
    padding: 0.6rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: 10px;
    background: color-mix(
      in srgb,
      var(--accent) calc(var(--intensity) * 25%),
      var(--surface-2)
    );
    color: var(--text);
    cursor: pointer;
  }
  .city-card.selected {
    outline: 2px solid var(--accent);
  }
  .city-name {
    font-weight: 700;
    font-size: var(--type-label);
  }
  .city-sports,
  .city-count {
    color: var(--text-muted);
    font-size: var(--type-caption);
    margin-top: 0.2rem;
  }
  @media (max-width: 768px) {
    .routes-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
