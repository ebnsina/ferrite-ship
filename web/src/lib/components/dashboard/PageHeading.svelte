<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		title: string;
		/** The single number that matters most on this page. */
		metric: string;
		caption: string;
		actions?: Snippet;
	}

	let { title, metric, caption, actions }: Props = $props();
</script>

<div class="flex flex-wrap items-end justify-between gap-4">
	<div>
		<h1 class="text-content text-2xl font-semibold tracking-tight sm:text-3xl">{title}</h1>
		<p
			class="from-content to-accent mt-1 bg-gradient-to-r bg-clip-text text-3xl font-semibold tracking-tight text-transparent sm:text-4xl"
			data-numeric
		>
			{metric}
		</p>
		<p class="text-content-muted mt-1.5 text-sm">{caption}</p>
	</div>

	{#if actions}
		<div class="flex items-center gap-2">{@render actions()}</div>
	{/if}
</div>
