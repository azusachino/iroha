<script lang="ts">
  import { onMount } from "svelte";
  import maplibregl from "maplibre-gl";
  import "maplibre-gl/dist/maplibre-gl.css";
  import type { RoutePoint } from "$lib/api";

  let {
    points,
    selectedIndex = null,
  }: { points: RoutePoint[]; selectedIndex?: number | null } = $props();

  let container: HTMLDivElement;
  let map: maplibregl.Map;
  let marker: maplibregl.Marker | undefined;

  // Key-free raster style backed by OpenStreetMap tiles (no API token needed).
  const style: maplibregl.StyleSpecification = {
    version: 8,
    sources: {
      osm: {
        type: "raster",
        tiles: ["https://tile.openstreetmap.org/{z}/{x}/{y}.png"],
        tileSize: 256,
        attribution: "© OpenStreetMap contributors",
      },
    },
    layers: [{ id: "osm", type: "raster", source: "osm" }],
  };

  onMount(() => {
    const coords = points
      .filter((p) => Number.isFinite(p.lat) && Number.isFinite(p.lon))
      .map((p) => [p.lon, p.lat] as [number, number]);

    map = new maplibregl.Map({
      container,
      style,
      center: coords.length ? coords[0] : [0, 0],
      zoom: coords.length ? 12 : 1,
      attributionControl: { compact: true },
    });
    map.addControl(
      new maplibregl.NavigationControl({ showCompass: false }),
      "top-right",
    );

    map.on("load", () => {
      if (coords.length < 2) return;

      map.addSource("route", {
        type: "geojson",
        data: {
          type: "Feature",
          properties: {},
          geometry: { type: "LineString", coordinates: coords },
        },
      });
      map.addLayer({
        id: "route-line",
        type: "line",
        source: "route",
        layout: { "line-join": "round", "line-cap": "round" },
        paint: { "line-color": "#4f8cff", "line-width": 4 },
      });
      marker = new maplibregl.Marker({ color: "#ff6b6b" })
        .setLngLat(coords[0])
        .addTo(map);

      // Fit the viewport to the full track.
      const bounds = coords.reduce(
        (b, c) => b.extend(c),
        new maplibregl.LngLatBounds(coords[0], coords[0]),
      );
      map.fitBounds(bounds, { padding: 32, duration: 0 });
    });

    return () => map.remove();
  });

  $effect(() => {
    if (!marker || selectedIndex == null) return;
    const point = points[selectedIndex];
    if (point && Number.isFinite(point.lat) && Number.isFinite(point.lon))
      marker.setLngLat([point.lon, point.lat]);
  });
</script>

<div class="map" bind:this={container}></div>
