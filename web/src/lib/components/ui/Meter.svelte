<script lang="ts">
	import { cn } from '$utils/cn';
	import { formatPercent } from '$utils/format';
	import type { LucideProps } from '@lucide/svelte';
	import type { Component } from 'svelte';
	import { TONE_CLASSES, toneForUsage } from './tone';

	interface Props {
		/** Names the measurement. Optional only when compact, where the column
		 *  heading names it instead — it is still used as the accessible name. */
		label?: string;
		/** 0–1. Values outside the range are clamped for display. */
		ratio: number;
		/** Right-aligned detail, e.g. "5.4 GB of 16 GB". */
		detail?: string;
		icon?: Component<LucideProps>;
		/** Drops the label row, for a table where the column already says what
		 *  this is and repeating it on every row would be noise. */
		compact?: boolean;
		class?: string;
	}

	let { label, ratio, detail, icon, compact = false, class: className }: Props = $props();

	const clamped = $derived(Math.min(Math.max(ratio, 0), 1));
	const tone = $derived(toneForUsage(clamped));
	const Icon = $derived(icon);
</script>

<div class={cn('space-y-1.5', className)}>
	{#if !compact}
		<div class="flex items-center justify-between gap-3 text-xs">
			<span class="text-content-muted flex items-center gap-1.5">
				{#if Icon}
					<Icon size={13} aria-hidden="true" />
				{/if}
				{label}
			</span>
			<span class="text-content" data-numeric>
				{detail ?? formatPercent(clamped)}
			</span>
		</div>
	{/if}

	<div
		class="bg-surface-sunken h-1.5 overflow-hidden rounded-pill"
		role="meter"
		aria-label={label ?? 'Usage'}
		aria-valuenow={Math.round(clamped * 100)}
		aria-valuemin={0}
		aria-valuemax={100}
	>
		<div
			class={cn(
				'h-full rounded-pill transition-[width] duration-500 ease-fluid',
				TONE_CLASSES[tone].fill
			)}
			style:width="{clamped * 100}%"
		></div>
	</div>

	{#if compact}
		<span class="text-content-muted block text-xs" data-numeric>
			{detail ?? formatPercent(clamped)}
		</span>
	{/if}
</div>
