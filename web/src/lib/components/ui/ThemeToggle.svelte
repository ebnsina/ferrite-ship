<script lang="ts">
	import type { Theme } from '$lib/theme/theme.svelte';
	import Moon from '@lucide/svelte/icons/moon';
	import Sun from '@lucide/svelte/icons/sun';

	interface Props {
		theme: Theme;
		onToggle: () => void;
	}

	let { theme, onToggle }: Props = $props();
</script>

<button
	type="button"
	onclick={onToggle}
	class="text-content-muted hover:bg-surface-raised hover:text-content inline-flex size-9 items-center justify-center rounded-tile transition-colors duration-150"
	aria-label={theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}
>
	{#if theme === 'dark'}
		<Sun size={16} aria-hidden="true" />
	{:else}
		<Moon size={16} aria-hidden="true" />
	{/if}
</button>
