import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [
		sveltekit({
			compilerOptions: {
				// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},

			// Read-only private viewer: a client-rendered SPA that talks to the
			// iroha-server read API at runtime. adapter-static with an SPA
			// fallback avoids any prerender/SSR dependency on a live backend.
			adapter: adapter({ fallback: 'index.html' })
		})
	]
});
