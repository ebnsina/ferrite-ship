<script lang="ts">
	import type { Snippet } from 'svelte';

	/**
	 * Applies a theme to a subtree rather than the whole document.
	 *
	 * The semantic tokens are declared against `[data-theme]`, which matches any
	 * element — so wrapping a section re-declares the custom properties for just
	 * that subtree. This is what lets the dashboard stay light while the
	 * marketing site stays dark.
	 */
	interface Props {
		theme: 'light' | 'dark';
		class?: string;
		children: Snippet;
	}

	let { theme, class: className, children }: Props = $props();
</script>

<div data-theme={theme} class={className} style="background-color: var(--ui-canvas); color: var(--ui-content)">
	{@render children()}
</div>
