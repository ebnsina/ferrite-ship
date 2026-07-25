<script lang="ts">
	import type { LucideProps } from '@lucide/svelte';
	import type { Component, Snippet } from 'svelte';

	interface Props {
		title: string;
		/** Says what this section shows and how to read it. */
		description: string;
		icon: Component<LucideProps>;
		/** Optional trailing content, e.g. a legend. */
		aside?: Snippet;
		children: Snippet;
	}

	let { title, description, icon, aside, children }: Props = $props();

	const Icon = $derived(icon);
</script>

<section class="space-y-4">
	<div class="flex flex-wrap items-end justify-between gap-3">
		<div>
			<h2 class="text-content flex items-center gap-2 text-sm font-medium">
				<Icon size={16} aria-hidden="true" class="text-content-muted" />
				{title}
			</h2>
			<p class="text-content-subtle mt-1 text-xs">{description}</p>
		</div>
		{#if aside}
			{@render aside()}
		{/if}
	</div>

	{@render children()}
</section>
