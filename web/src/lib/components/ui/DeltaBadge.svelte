<script lang="ts">
	import { formatPercent } from '$utils/format';
	import ArrowDown from '@lucide/svelte/icons/arrow-down';
	import ArrowUp from '@lucide/svelte/icons/arrow-up';
	import { TONE_CLASSES } from './tone';

	interface Props {
		/** Change as a ratio: 0.15 is +15%. */
		ratio: number;
		/** Whether a rise is good news. Decides which direction reads as green. */
		higherIsBetter?: boolean;
	}

	let { ratio, higherIsBetter = true }: Props = $props();

	const rising = $derived(ratio >= 0);
	const good = $derived(rising === higherIsBetter);
	const tone = $derived(ratio === 0 ? 'pending' : good ? 'ok' : 'warn');
</script>

<span
	class="inline-flex items-center gap-0.5 rounded-pill px-1.5 py-0.5 text-xs font-medium {TONE_CLASSES[
		tone
	].soft}"
>
	{#if ratio !== 0}
		{#if rising}
			<ArrowUp size={11} aria-hidden="true" />
		{:else}
			<ArrowDown size={11} aria-hidden="true" />
		{/if}
	{/if}
	<span data-numeric>{formatPercent(Math.abs(ratio))}</span>
</span>
