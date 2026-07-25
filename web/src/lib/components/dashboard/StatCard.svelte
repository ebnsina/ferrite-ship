<script lang="ts">
	import { Card } from '$components/ui';
	import { TONE_CLASSES, type Tone } from '$components/ui/tone';
	import type { LucideProps } from '@lucide/svelte';
	import type { Component } from 'svelte';

	interface Props {
		label: string;
		value: string;
		hint?: string;
		icon: Component<LucideProps>;
		tone?: Tone;
	}

	let { label, value, hint, icon, tone = 'pending' }: Props = $props();

	const Icon = $derived(icon);
</script>

<Card>
	<div class="flex items-start justify-between gap-4">
		<div class="min-w-0">
			<p class="text-content-muted text-xs font-medium">{label}</p>
			<p class="text-content mt-2 text-2xl font-semibold tracking-tight" data-numeric>{value}</p>
			{#if hint}
				<p class="text-content-subtle mt-1 text-xs leading-snug">{hint}</p>
			{/if}
		</div>
		<span class="rounded-tile p-2.5 {TONE_CLASSES[tone].soft}">
			<Icon size={17} aria-hidden="true" />
		</span>
	</div>
</Card>
