<script lang="ts">
	import { Button, ThemeToggle } from '$components/ui';
	import type { LucideProps } from '@lucide/svelte';
	import CirclePlus from '@lucide/svelte/icons/circle-plus';
	import type { Component, Snippet } from 'svelte';

	interface Props {
		title: string;
		/** One plain sentence saying what this page is for. Not optional by design. */
		description: string;
		icon: Component<LucideProps>;
		actions?: Snippet;
	}

	let { title, description, icon, actions }: Props = $props();

	const Icon = $derived(icon);
</script>

<div class="border-border/70 border-b">
	<div class="flex flex-wrap items-center gap-4 px-6 py-5">
		<span class="bg-accent-soft text-accent flex size-10 shrink-0 items-center justify-center rounded-tile">
			<Icon size={18} aria-hidden="true" />
		</span>

		<div class="min-w-0 flex-1">
			<h1 class="text-content truncate text-lg font-semibold tracking-tight">{title}</h1>
			<p class="text-content-muted mt-0.5 text-sm">{description}</p>
		</div>

		<div class="flex items-center gap-2">
			{#if actions}
				{@render actions()}
			{/if}
			<ThemeToggle />
			<Button size="sm">
				<CirclePlus size={15} aria-hidden="true" />
				Connect a server
			</Button>
		</div>
	</div>
</div>
