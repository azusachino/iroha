<script lang="ts">
  import { onMount } from "svelte";
  import maplibregl from "maplibre-gl";
  import "maplibre-gl/dist/maplibre-gl.css";
  import type { FeatureCollection } from "geojson";
  import type { RouteFeatureCollection } from "$lib/api";

  let { data }: { data: RouteFeatureCollection } = $props();

  let container: HTMLDivElement;
  let map: maplibregl.Map | null = null;
  let loaded = $state(false);
  const osmMaxZoom = 19;
  const pointCount = $derived(
    data.features.reduce(
      (count, feature) => count + feature.geometry.coordinates.length,
      0,
    ),
  );
  const mapLabel = $derived(
    `Interactive route map with ${data.features.length} routes and ${pointCount} recorded coordinates`,
  );

  // Key-free raster style backed by OpenStreetMap tiles (no API token needed).
  // Reuses the same style as RouteMap.svelte for visual consistency.
  const style: maplibregl.StyleSpecification = {
    version: 8,
    sources: {
      osm: {
        type: "raster",
        tiles: ["https://tile.openstreetmap.org/{z}/{x}/{y}.png"],
        tileSize: 256,
        maxzoom: osmMaxZoom,
        attribution: "© OpenStreetMap contributors",
      },
    },
    layers: [{ id: "osm", type: "raster", source: "osm" }],
  };

  // Draw (or redraw) the current feature set and fit the viewport to it. Runs
  // on first load and again whenever `data` changes (e.g. a year or city
  // filter), so the map stays in sync with the surrounding page.
  function render(fc: RouteFeatureCollection) {
    if (!map || !loaded) return;
    const coords = fc.features.flatMap((f) => f.geometry.coordinates);

    const src = map.getSource("routes") as maplibregl.GeoJSONSource | undefined;
    if (src) {
      src.setData(fc as unknown as FeatureCollection);
    } else {
      map.addSource("routes", {
        type: "geojson",
        data: fc as unknown as FeatureCollection,
      });
      // Semi-transparent, thin lines: overlapping routes build up into a
      // denser "heat" effect where the same paths are run often.
      map.addLayer({
        id: "routes-line",
        type: "line",
        source: "routes",
        layout: { "line-join": "round", "line-cap": "round" },
        paint: {
          "line-color": "#4f8cff",
          "line-width": 1.5,
          "line-opacity": 0.4,
        },
      });
    }

    if (coords.length === 0) return;
    const bounds = coords.reduce(
      (b, c) => b.extend(c as [number, number]),
      new maplibregl.LngLatBounds(
        coords[0] as [number, number],
        coords[0] as [number, number],
      ),
    );
    const reducedMotion = window.matchMedia(
      "(prefers-reduced-motion: reduce)",
    ).matches;
    map.fitBounds(bounds, {
      padding: 32,
      duration: reducedMotion ? 0 : 300,
      maxZoom: osmMaxZoom,
    });
  }

  onMount(() => {
    map = new maplibregl.Map({
      container,
      style,
      center: [0, 0],
      zoom: 1,
      maxZoom: osmMaxZoom,
      attributionControl: { compact: true },
    });
    map.addControl(
      new maplibregl.NavigationControl({ showCompass: false }),
      "top-right",
    );
    map.on("load", () => {
      loaded = true;
    });

    return () => {
      map?.remove();
      map = null;
      loaded = false;
    };
  });

  // Re-render whenever the data or the load state changes.
  $effect(() => {
    const fc = data;
    if (loaded) render(fc);
  });
</script>

<section class="map-shell" aria-label={mapLabel}>
  <div
    class="map"
    bind:this={container}
    role="region"
    aria-label={mapLabel}
    aria-describedby="routes-map-help"
  ></div>
  <p id="routes-map-help" class="visually-hidden">
    Use the map zoom and pan controls to inspect route density. A route summary
    table follows for keyboard and screen-reader access.
  </p>
  <details class="map-data">
    <summary>View route summaries</summary>
    <div class="table-wrap">
      <table>
        <caption>Recorded route summaries</caption>
        <thead>
          <tr>
            <th scope="col">Route</th>
            <th scope="col">Sport</th>
            <th scope="col">Year</th>
            <th scope="col">Place</th>
            <th scope="col">Coordinates</th>
          </tr>
        </thead>
        <tbody>
          {#each data.features as feature, index}
            <tr>
              <th scope="row">Route {index + 1}</th>
              <td>{feature.properties.sport_type}</td>
              <td>{feature.properties.year}</td>
              <td>{feature.properties.city ?? "—"}</td>
              <td>{feature.geometry.coordinates.length}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </details>
</section>

<style>
  .map-shell {
    display: grid;
    gap: 0.55rem;
    min-width: 0;
  }
  .map {
    width: 100%;
    min-height: 18rem;
    overflow: hidden;
    border-radius: var(--radius);
  }
  .map-data {
    min-width: 0;
    color: var(--text-muted);
    font-size: 0.78rem;
  }
  .map-data summary {
    width: fit-content;
    cursor: pointer;
  }
  .table-wrap {
    margin-top: 0.5rem;
    overflow-x: auto;
  }
  table {
    width: 100%;
    min-width: min(34rem, 100%);
    border-collapse: collapse;
    color: var(--text);
    font-variant-numeric: tabular-nums;
  }
  th,
  td {
    padding: 0.35rem 0.5rem;
    border-bottom: 1px solid var(--border);
    text-align: left;
  }
  thead th {
    color: var(--text-muted);
    font-weight: 650;
  }
</style>
