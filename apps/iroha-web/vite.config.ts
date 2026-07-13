import adapter from "@sveltejs/adapter-static";
import { sveltekit } from "@sveltejs/kit/vite";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";

export default defineConfig({
  server: {
    // Reachable over Tailscale/LAN via `make web-dev` (binds 0.0.0.0). Allow
    // the machine's MagicDNS short name and any *.ts.net FQDN; IPs are always
    // allowed. Add more hosts here if you reach it by another name.
    allowedHosts: ["harus-macmini", ".ts.net"],
    // Proxy API calls to the local iroha-server, so a remote browser only
    // ever talks to this dev origin — no CORS and no per-host API base. The
    // server stays bound to localhost; Vite forwards from the same machine.
    proxy: {
      "/api": "http://127.0.0.1:8080",
      "/public": "http://127.0.0.1:8080",
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

      // Read-only private viewer: a client-rendered SPA that talks to the
      // iroha-server read API at runtime. adapter-static with an SPA
      // fallback avoids any prerender/SSR dependency on a live backend.
      adapter: adapter({ fallback: "index.html" }),
    }),
  ],
});
