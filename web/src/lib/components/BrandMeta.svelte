<script lang="ts">
	import { BRAND } from '$lib/theme/brand.generated';

	/*
	 * The colours the browser itself uses — the pinned-tab tint and the chrome
	 * around the page.
	 *
	 * These live here rather than in app.html because app.html is a static file
	 * and cannot read a token. Hand-written there, they kept a lime mask-icon
	 * through two brand changes. Rendered from the generated module, they
	 * cannot fall behind.
	 */
</script>

<svelte:head>
	<link rel="mask-icon" href="/favicon.svg" color={BRAND.mark} />
	<!-- Marketing is dark and the dashboard light, so the browser chrome
	     follows the reader's scheme rather than one fixed colour. -->
	<meta name="theme-color" media="(prefers-color-scheme: dark)" content={BRAND.canvasDark} />
	<meta name="theme-color" media="(prefers-color-scheme: light)" content={BRAND.canvasLight} />
</svelte:head>
