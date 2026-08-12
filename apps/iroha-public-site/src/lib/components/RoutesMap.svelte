<script lang="ts">
  import { onMount } from "svelte";
  import maplibregl from "maplibre-gl";
  import "maplibre-gl/dist/maplibre-gl.css";
  import type { FeatureCollection } from "geojson";
  import type { RouteFeatureCollection } from "$lib/types";

  let { data }: { data: RouteFeatureCollection } = $props();

  let container: HTMLDivElement;
  let map: maplibregl.Map | null = null;
  let loaded = $state(false);
  const osmMaxZoom = 19;

  // Key-free raster style backed by OpenStreetMap tiles (no API token needed).
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
  // on first load and again whenever `data` changes (e.g. a year filter), so
  // the map stays in sync with the surrounding page.
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
    map.fitBounds(bounds, { padding: 32, duration: 300, maxZoom: osmMaxZoom });
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

<div class="map" bind:this={container}></div>

<style>
  .map {
    width: 100%;
    height: 100%;
    min-height: 18rem;
    border-radius: var(--radius);
    overflow: hidden;
  }
</style>
