<script lang="ts">
	import SiteFooter from '$components/marketing/SiteFooter.svelte';
	import SiteHeader from '$components/marketing/SiteHeader.svelte';
	import type { Snippet } from 'svelte';

	interface Props {
		children: Snippet;
	}

	let { children }: Props = $props();
</script>

<div class="flex min-h-dvh flex-col">
	<SiteHeader />
	<main class="flex-1">
		{@render children()}
	</main>
	<SiteFooter />
</div>
