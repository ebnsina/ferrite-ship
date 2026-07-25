<script lang="ts">
	import { cn } from '$utils/cn';
	import type { Snippet } from 'svelte';

	interface Props {
		class?: string;
		/** Adds a hover treatment. Use only when the whole card is interactive. */
		interactive?: boolean;
		padded?: boolean;
		children: Snippet;
	}

	let { class: className, interactive = false, padded = true, children }: Props = $props();
</script>

<div
	class={cn(
		'border-border bg-surface rounded-card border',
		padded && 'p-6',
		interactive &&
			'hover:border-border-strong hover:bg-surface-raised transition-colors duration-150 ease-snappy',
		className
	)}
>
	{@render children()}
</div>
