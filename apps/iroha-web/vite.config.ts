import adapter from "@sveltejs/adapter-static";
import { sveltekit } from "@sveltejs/kit/vite";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig, loadEnv } from "vite";

const sharedPath = new URL("../../packages/iroha-shared/src", import.meta.url)
  .pathname;

// iroha-server that `vite dev` and `vite preview` forward /api to.
// loadEnv reads the process environment without needing Node's type
// definitions in this app.
const apiTarget =
  loadEnv("", ".", "IROHA_DEV_").IROHA_DEV_API_TARGET ??
  "http://127.0.0.1:8080";

export default defineConfig({
  resolve: {
    // Shared and host imports must use one ECharts/zrender state registry.
    dedupe: ["echarts", "zrender"],
    alias: {
      "@iroha/shared": sharedPath,
    },
  },
  build: {
    // MapLibre and ECharts are core to the private route/detail experience and
    // together produce one intentionally large vendor chunk.
    chunkSizeWarningLimit: 1400,
  },
  server: {
    // Reachable over Tailscale/LAN via `make web-dev` (binds 0.0.0.0). Allow
    // the machine's MagicDNS short name and any *.ts.net FQDN; IPs are always
    // allowed. Add more hosts here if you reach it by another name.
    allowedHosts: ["harus-macmini", "harus-mini", ".ts.net"],
    // Proxy API calls to the local iroha-server, so a remote browser only
    // ever talks to this dev origin — no CORS and no per-host API base. The
    // server stays bound to localhost; Vite forwards from the same machine.
    proxy: {
      "/api": apiTarget,
    },
  },
  // The preview server proxies the same way, so browser checks stay
  // same-origin and the HttpOnly session cookie reaches the API.
  preview: {
    proxy: {
      "/api": apiTarget,
    },
  },
  plugins: [
    tailwindcss(),
    sveltekit({
      compilerOptions: {
        // Force runes mode for the project, except for libraries. Can be removed in svelte 6.
        runes: ({ filename }) =>
          filename.split(/[/\\]/).includes("node_modules") ? undefined : true,
      },

      alias: {
        $lib: "src/lib",
        "@iroha/shared": sharedPath,
      },

      // Read-only private viewer: a client-rendered SPA that talks to the
      // iroha-server read API at runtime. adapter-static with an SPA
      // fallback avoids any prerender/SSR dependency on a live backend.
      adapter: adapter({ fallback: "index.html" }),
    }),
  ],
});
