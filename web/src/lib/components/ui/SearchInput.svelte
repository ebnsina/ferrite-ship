<script lang="ts">
	import Search from '@lucide/svelte/icons/search';

	interface Props {
		placeholder?: string;
		value?: string;
	}

	let { placeholder = 'Search', value = $bindable('') }: Props = $props();

	let input = $state<HTMLInputElement | null>(null);

	// "/" focuses search, the convention users already know from other tools.
	function onKeydown(event: KeyboardEvent) {
		if (event.key !== '/' || event.metaKey || event.ctrlKey || event.altKey) return;

		const active = document.activeElement;
		const typing =
			active instanceof HTMLInputElement ||
			active instanceof HTMLTextAreaElement ||
			(active instanceof HTMLElement && active.isContentEditable);
		if (typing) return;

		event.preventDefault();
		input?.focus();
	}
</script>

<svelte:window onkeydown={onKeydown} />

<div class="relative w-full max-w-xs">
	<Search
		size={15}
		aria-hidden="true"
		class="text-content-subtle pointer-events-none absolute top-1/2 left-3 -translate-y-1/2"
	/>
	<input
		bind:this={input}
		bind:value
		type="search"
		{placeholder}
		aria-label={placeholder}
		class="border-border bg-surface-sunken text-content placeholder:text-content-subtle focus-visible:border-accent-border h-9 w-full rounded-field border py-2 pr-10 pl-9 text-sm outline-none transition-colors duration-150"
	/>
	<kbd
		class="border-border text-content-subtle pointer-events-none absolute top-1/2 right-2.5 -translate-y-1/2 rounded border px-1.5 py-0.5 text-[0.6875rem]"
	>
		/
	</kbd>
</div>
