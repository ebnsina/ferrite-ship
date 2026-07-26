<script lang="ts">
	import { cn } from '$utils/cn';
	import type { LucideProps } from '@lucide/svelte';
	import type { Component } from 'svelte';
	import { TONE_CLASSES, type Tone } from './tone';

	interface Props {
		icon: Component<LucideProps>;
		tone?: Tone | 'neutral';
		size?: 'sm' | 'md';
		/**
		 * A brand colour to tint with, as `#rrggbb`. Use only for a third
		 * party's own identity — never to say something is fine or broken, which
		 * is what `tone` is for. Takes precedence over `tone` when set.
		 */
		color?: string;
		class?: string;
	}

	let { icon, tone = 'neutral', size = 'md', color, class: className }: Props = $props();

	const Icon = $derived(icon);

	const SIZES = {
		// Component-tier override: a 2rem box takes the full tile radius almost
		// to a circle, so the small size gets its own proportional value.
		sm: { box: 'size-8 rounded-[0.75rem]', glyph: 15 },
		md: { box: 'size-10 rounded-tile', glyph: 18 }
	} as const;

	/*
	 * Brand colours are chosen to work on a press kit, not on our surfaces, so
	 * they are mixed rather than used flat. The background is a wash of the
	 * colour; the glyph is pulled a third of the way toward whatever the text
	 * colour is in the current theme.
	 *
	 * That last part is what makes ClickHouse's yellow legible: on white it
	 * darkens toward near-black and on dark it lightens, staying recognisably
	 * yellow either way. Used flat it would be invisible in light mode.
	 */
	const tinted = $derived(
		color
			? `--tile-bg: color-mix(in oklch, ${color} 15%, transparent); ` +
				`--tile-fg: color-mix(in oklch, ${color} 68%, var(--ui-content) 32%);`
			: undefined
	);
</script>

<span
	style={tinted}
	class={cn(
		'inline-flex shrink-0 items-center justify-center',
		SIZES[size].box,
		color
			? 'bg-(--tile-bg) text-(--tile-fg)'
			: tone === 'neutral'
				? 'bg-surface-sunken text-content-muted'
				: TONE_CLASSES[tone].soft,
		className
	)}
>
	<Icon size={SIZES[size].glyph} aria-hidden="true" />
</span>
