import adapter from "@sveltejs/adapter-static";
import { sveltekit } from "@sveltejs/kit/vite";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";

const sharedPath = new URL("../../packages/iroha-shared/src", import.meta.url)
  .pathname;

// iroha-server that `vite dev` and `vite preview` forward /public/v1 to.
const apiTarget = process.env.IROHA_DEV_API_TARGET ?? "http://127.0.0.1:8080";

export default defineConfig({
  resolve: {
    // Shared and host imports must use one ECharts/zrender state registry.
    dedupe: ["echarts", "zrender"],
    alias: {
      "@iroha/shared": sharedPath,
    },
  },
  build: {
    // MapLibre and ECharts are core to the public route/detail experience and
    // together produce one intentionally large vendor chunk.
    chunkSizeWarningLimit: 1700,
  },
  server: {
    allowedHosts: ["harus-macmini", "harus-mini", ".ts.net"],
    proxy: { "/public": apiTarget },
  },
  preview: {
    proxy: { "/public": apiTarget },
  },
  plugins: [
    tailwindcss(),
    sveltekit({
      compilerOptions: {
        runes: true,
      },
      alias: {
        $lib: "src/lib",
        "@iroha/shared": sharedPath,
      },
      // Self-hosted at the root (no GitHub Pages project-page subpath to
      // account for) -- default base path.
      // The main page loads live data in the browser from /public/v1, so
      // the build is a client-rendered shell with an SPA fallback.
      adapter: adapter({ fallback: "index.html" }),
    }),
  ],
});
