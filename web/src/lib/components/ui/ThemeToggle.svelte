<script lang="ts">
	import Moon from '@lucide/svelte/icons/moon';
	import Sun from '@lucide/svelte/icons/sun';

	type Theme = 'dark' | 'light';
	const STORAGE_KEY = 'ferrite-theme';

	// Initialised from the DOM, which app.html has already resolved before paint.
	let theme = $state<Theme>('dark');

	$effect(() => {
		const current = document.documentElement.dataset.theme;
		theme = current === 'light' ? 'light' : 'dark';
	});

	function toggle() {
		theme = theme === 'dark' ? 'light' : 'dark';
		document.documentElement.dataset.theme = theme;
		try {
			localStorage.setItem(STORAGE_KEY, theme);
		} catch {
			// Storage blocked — the choice simply will not persist across reloads.
		}
	}
</script>

<button
	type="button"
	onclick={toggle}
	class="text-content-muted hover:bg-surface-raised hover:text-content inline-flex h-9 w-9 items-center justify-center rounded-tile transition-colors duration-150"
	aria-label={theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}
>
	{#if theme === 'dark'}
		<Sun size={16} aria-hidden="true" />
	{:else}
		<Moon size={16} aria-hidden="true" />
	{/if}
</button>
