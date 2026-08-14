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
  const osmMaxZoom = 19;
  const validPoints = $derived(
    points.filter(
      (point) => Number.isFinite(point.lat) && Number.isFinite(point.lon),
    ),
  );
  const mapLabel = $derived(
    validPoints.length
      ? `Interactive route map with ${validPoints.length} recorded GPS fixes`
      : "Interactive route map with no recorded GPS fixes",
  );

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

  onMount(() => {
    const coords = points
      .filter((p) => Number.isFinite(p.lat) && Number.isFinite(p.lon))
      .map((p) => [p.lon, p.lat] as [number, number]);

    map = new maplibregl.Map({
      container,
      style,
      center: coords.length ? coords[0] : [0, 0],
      zoom: coords.length ? 12 : 1,
      maxZoom: osmMaxZoom,
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
      map.fitBounds(bounds, { padding: 32, duration: 0, maxZoom: osmMaxZoom });
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

<section class="map-shell" aria-label={mapLabel}>
  <div
    class="map"
    bind:this={container}
    role="region"
    aria-label={mapLabel}
    aria-describedby="route-map-help"
  ></div>
  <p id="route-map-help" class="visually-hidden">
    Use the map zoom and pan controls to inspect the route. The exact recorded
    coordinates are available in the route data table below.
  </p>
  <details class="map-data">
    <summary>View route coordinates</summary>
    <div class="table-wrap">
      <table>
        <caption>Recorded route coordinates</caption>
        <thead>
          <tr>
            <th scope="col">Fix</th>
            <th scope="col">Recorded at</th>
            <th scope="col">Latitude</th>
            <th scope="col">Longitude</th>
            <th scope="col">Elevation</th>
          </tr>
        </thead>
        <tbody>
          {#each validPoints as point}
            <tr>
              <th scope="row">{point.seq}</th>
              <td>{point.ts ?? "—"}</td>
              <td>{point.lat.toFixed(6)}</td>
              <td>{point.lon.toFixed(6)}</td>
              <td>
                {point.elevation_m == null
                  ? "—"
                  : `${point.elevation_m.toFixed(1)} m`}
              </td>
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
  }
  .map {
    width: 100%;
    min-height: 18rem;
    overflow: hidden;
    border-radius: var(--radius);
  }
  .map-data {
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
    min-width: 42rem;
    border-collapse: collapse;
    color: var(--text);
    font-variant-numeric: tabular-nums;
  }
  th,
  td {
    padding: 0.35rem 0.5rem;
    border-bottom: 1px solid var(--border);
    text-align: right;
  }
  th:first-child,
  td:first-child,
  th:nth-child(2),
  td:nth-child(2) {
    text-align: left;
  }
  thead th {
    color: var(--text-muted);
    font-weight: 650;
  }
</style>
