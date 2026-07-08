<script lang="ts">
	import { onMount } from 'svelte';
	import maplibregl from 'maplibre-gl';
	import 'maplibre-gl/dist/maplibre-gl.css';
	import type { RouteFeatureCollection } from '$lib/api';

	let { data }: { data: RouteFeatureCollection } = $props();

	let container: HTMLDivElement;

	// Key-free raster style backed by OpenStreetMap tiles (no API token needed).
	// Reuses the same style as RouteMap.svelte for visual consistency.
	const style: maplibregl.StyleSpecification = {
		version: 8,
		sources: {
			osm: {
				type: 'raster',
				tiles: ['https://tile.openstreetmap.org/{z}/{x}/{y}.png'],
				tileSize: 256,
				attribution: '© OpenStreetMap contributors'
			}
		},
		layers: [{ id: 'osm', type: 'raster', source: 'osm' }]
	};

	onMount(() => {
		const allCoords = data.features.flatMap((f) => f.geometry.coordinates);

		const map = new maplibregl.Map({
			container,
			style,
			center: allCoords.length ? allCoords[0] : [0, 0],
			zoom: allCoords.length ? 10 : 1,
			attributionControl: { compact: true }
		});
		map.addControl(new maplibregl.NavigationControl({ showCompass: false }), 'top-right');

		map.on('load', () => {
			if (allCoords.length === 0) return;

			map.addSource('routes', {
				type: 'geojson',
				data
			});
			// Semi-transparent, thin lines: overlapping routes build up into a
			// denser "heat" effect where the same paths are run often.
			map.addLayer({
				id: 'routes-line',
				type: 'line',
				source: 'routes',
				layout: { 'line-join': 'round', 'line-cap': 'round' },
				paint: { 'line-color': '#4f8cff', 'line-width': 1.5, 'line-opacity': 0.4 }
			});

			// Fit the viewport to every route.
			const bounds = allCoords.reduce(
				(b, c) => b.extend(c as [number, number]),
				new maplibregl.LngLatBounds(allCoords[0] as [number, number], allCoords[0] as [number, number])
			);
			map.fitBounds(bounds, { padding: 32, duration: 0 });
		});

		return () => map.remove();
	});
</script>

<div class="map" bind:this={container}></div>
