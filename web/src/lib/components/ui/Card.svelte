<script lang="ts">
	import { cn } from '$utils/cn';
	import type { Snippet } from 'svelte';

	/**
	 * How big the surface is, which decides its corner radius.
	 *
	 * Radius is a function of size, not a house style. The 2rem corner that
	 * makes a full-width panel look generous turns a 56px inline banner into a
	 * lozenge, because the curve then eats most of the edge it is supposed to
	 * be softening. Three sizes is enough to keep that honest.
	 */
	export type CardSize = 'sm' | 'md' | 'lg';

	interface Props {
		class?: string;
		/** Adds a hover treatment. Use only when the whole card is interactive. */
		interactive?: boolean;
		padded?: boolean;
		/** sm: banners and nested panels. md: the default. lg: large surfaces. */
		size?: CardSize;
		children: Snippet;
	}

	let {
		class: className,
		interactive = false,
		padded = true,
		size = 'md',
		children
	}: Props = $props();

	const RADIUS: Record<CardSize, string> = {
		sm: 'rounded-card-sm',
		md: 'rounded-card',
		lg: 'rounded-panel'
	};

	// Padding tracks the radius: a generous corner needs room to breathe inside
	// it, or the content sits in the curve.
	const PADDING: Record<CardSize, string> = {
		sm: 'p-4',
		md: 'p-6',
		lg: 'p-7'
	};
</script>

<div
	class={cn(
		'border-border bg-surface border',
		RADIUS[size],
		padded && PADDING[size],
		interactive &&
			'hover:border-border-strong hover:bg-surface-raised transition-colors duration-150 ease-snappy',
		className
	)}
>
	{@render children()}
</div>
