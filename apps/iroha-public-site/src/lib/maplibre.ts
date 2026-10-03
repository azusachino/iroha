// MapLibre's worker URL is indirect, so Vite needs an explicit worker asset.
// https://vite.dev/guide/features.html#import-with-query-suffixes
import { setWorkerUrl } from "maplibre-gl";
import workerUrl from "maplibre-gl/dist/maplibre-gl-worker.mjs?worker&url";

setWorkerUrl(workerUrl);
export * from "maplibre-gl";
