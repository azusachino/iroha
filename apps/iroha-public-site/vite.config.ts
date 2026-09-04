import adapter from "@sveltejs/adapter-static";
import { sveltekit } from "@sveltejs/kit/vite";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";

const sharedPath = new URL("../../packages/iroha-shared/src", import.meta.url)
  .pathname;

export default defineConfig({
  resolve: {
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
  },
  plugins: [
    tailwindcss(),
    sveltekit({
      alias: { "@iroha/shared": sharedPath },
      compilerOptions: {
        runes: true,
      },
      // Self-hosted at the root (no GitHub Pages project-page subpath to
      // account for) -- default base path.
      // Every page is known and fetchable at build time (this is a single,
      // fully static snapshot) -- prerender the whole site instead of
      // shipping a client-rendered shell.
      prerender: { entries: ["*"] },
      adapter: adapter({ fallback: undefined }),
    }),
  ],
});
