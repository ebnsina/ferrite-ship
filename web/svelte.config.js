import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/**
 * Marketing routes are prerendered to static HTML for SEO and instant loads.
 * The dashboard opts out (see src/routes/dashboard/+layout.ts) and is served
 * from the SPA fallback, because it is an authenticated client of the Go API.
 *
 * @type {import('@sveltejs/kit').Config}
 */
const config = {
	preprocess: vitePreprocess(),
	kit: {
		adapter: adapter({
			pages: 'build',
			assets: 'build',
			fallback: '200.html',
			precompress: false,
			strict: true
		}),
		alias: {
			$components: 'src/lib/components',
			$config: 'src/lib/config',
			$utils: 'src/lib/utils',
			$types: 'src/lib/types'
		}
	}
};

export default config;
