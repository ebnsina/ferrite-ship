<script lang="ts">
	import '../app.css';
	import BrandMeta from '$components/BrandMeta.svelte';
	import type { Snippet } from 'svelte';

	interface Props {
		children: Snippet;
	}

	let { children }: Props = $props();
</script>

<BrandMeta />

{@render children()}
