<script lang="ts">
	import { cn } from '$utils/cn';
	import type { LucideProps } from '@lucide/svelte';
	import type { Component } from 'svelte';
	import { TONE_CLASSES, type Tone } from './tone';

	interface Props {
		icon: Component<LucideProps>;
		tone?: Tone | 'neutral';
		size?: 'sm' | 'md';
		class?: string;
	}

	let { icon, tone = 'neutral', size = 'md', class: className }: Props = $props();

	const Icon = $derived(icon);

	const SIZES = {
		// Component-tier override: a 2rem box takes the full tile radius almost
		// to a circle, so the small size gets its own proportional value.
		sm: { box: 'size-8 rounded-[0.75rem]', glyph: 15 },
		md: { box: 'size-10 rounded-tile', glyph: 18 }
	} as const;
</script>

<span
	class={cn(
		'inline-flex shrink-0 items-center justify-center',
		SIZES[size].box,
		tone === 'neutral' ? 'bg-surface-sunken text-content-muted' : TONE_CLASSES[tone].soft,
		className
	)}
>
	<Icon size={SIZES[size].glyph} aria-hidden="true" />
</span>
