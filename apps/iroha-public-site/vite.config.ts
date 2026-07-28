import adapter from "@sveltejs/adapter-static";
import { sveltekit } from "@sveltejs/kit/vite";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";

// GitHub Pages serves this as a project page at azusachino.github.io/iroha/,
// not a root user site, so every asset/data URL needs the /iroha prefix in
// production. Empty by default so `vite dev`/`vite preview` work at the root
// locally; the Pages deploy workflow sets BASE_PATH=/iroha at build time.
const rawBase = process.env.BASE_PATH ?? "";
const base = (
  rawBase === "" || rawBase.startsWith("/") ? rawBase : `/${rawBase}`
) as "" | `/${string}`;

export default defineConfig({
  plugins: [
    tailwindcss(),
    sveltekit({
      compilerOptions: {
        runes: true,
      },
      paths: { base },
      // Every page is known and fetchable at build time (this is a single,
      // fully static snapshot) -- prerender the whole site instead of
      // shipping a client-rendered shell.
      prerender: { entries: ["*"] },
      adapter: adapter({ fallback: undefined }),
    }),
  ],
});
