<script lang="ts">
	import type { LucideProps } from '@lucide/svelte';
	import Inbox from '@lucide/svelte/icons/inbox';
	import type { Component, Snippet } from 'svelte';

	interface Props {
		title: string;
		description: string;
		icon?: Component<LucideProps>;
		/** Optional call to action. */
		action?: Snippet;
	}

	let { title, description, icon = Inbox, action }: Props = $props();

	const Icon = $derived(icon);
</script>

<div class="flex flex-col items-center px-6 py-16 text-center">
	<div class="bg-surface-raised text-content-subtle rounded-tile p-4">
		<Icon size={22} aria-hidden="true" />
	</div>
	<h3 class="text-content mt-5 text-base font-medium">{title}</h3>
	<p class="text-content-muted mt-1.5 max-w-sm text-sm">{description}</p>
	{#if action}
		<div class="mt-6">{@render action()}</div>
	{/if}
</div>
