<script lang="ts">
	import type { StatusPresentation } from '$lib/domain/status';
	import { cn } from '$utils/cn';
	import { TONE_CLASSES } from './tone';

	interface Props {
		status: StatusPresentation;
		/** Hide the text label in dense contexts; the icon and title carry it. */
		compact?: boolean;
		class?: string;
	}

	let { status, compact = false, class: className }: Props = $props();

	const Icon = $derived(status.icon);
</script>

<span
	class={cn(
		'inline-flex items-center gap-1.5 rounded-pill px-2.5 py-1 text-xs font-medium',
		TONE_CLASSES[status.tone].soft,
		className
	)}
	title={compact ? status.label : undefined}
>
	<Icon size={13} aria-hidden="true" class={status.animated ? 'animate-spin' : undefined} />
	{#if compact}
		<span class="sr-only">{status.label}</span>
	{:else}
		{status.label}
	{/if}
</span>
