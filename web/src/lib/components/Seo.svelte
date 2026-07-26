<script lang="ts">
	import { page } from '$app/state';
	import { env } from '$config/env';

	const SITE_NAME = 'Ferrite Ship';
	const DEFAULT_IMAGE = '/og.png';

	interface Props {
		/** Page title. The site name is appended unless `bare` is set. */
		title: string;
		description: string;
		/** Absolute path for the canonical URL. Defaults to the current route. */
		path?: string;
		/** Keep the title exactly as given — for the home page. */
		bare?: boolean;
		/** Authenticated pages should never be indexed. */
		noindex?: boolean;
		image?: string;
	}

	let { title, description, path, bare = false, noindex = false, image }: Props = $props();

	const fullTitle = $derived(bare ? title : `${title} · ${SITE_NAME}`);

	// Built from the configured origin rather than the request, so the canonical
	// URL is right behind a proxy and identical across prerender and runtime.
	const canonical = $derived(`${env.appUrl}${path ?? page.url.pathname}`.replace(/\/$/, '') || env.appUrl);
	const imageUrl = $derived(`${env.appUrl}${image ?? DEFAULT_IMAGE}`);
</script>

<svelte:head>
	<title>{fullTitle}</title>
	<meta name="description" content={description} />
	<link rel="canonical" href={canonical} />

	{#if noindex}
		<meta name="robots" content="noindex, nofollow" />
	{:else}
		<meta name="robots" content="index, follow" />
	{/if}

	<!-- Open Graph -->
	<meta property="og:type" content="website" />
	<meta property="og:site_name" content={SITE_NAME} />
	<meta property="og:title" content={fullTitle} />
	<meta property="og:description" content={description} />
	<meta property="og:url" content={canonical} />
	<meta property="og:image" content={imageUrl} />
	<meta property="og:image:width" content="1200" />
	<meta property="og:image:height" content="630" />
	<meta property="og:image:alt" content="Ferrite Ship — run your server from your browser" />
	<meta property="og:locale" content="en_GB" />

	<!-- Twitter / X -->
	<meta name="twitter:card" content="summary_large_image" />
	<meta name="twitter:title" content={fullTitle} />
	<meta name="twitter:description" content={description} />
	<meta name="twitter:image" content={imageUrl} />
</svelte:head>
