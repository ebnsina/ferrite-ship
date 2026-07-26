<script lang="ts">
	import { buttonClasses, type ButtonSize, type ButtonVariant } from './button-styles';
	import { cn } from '$utils/cn';
	import ChevronDown from '@lucide/svelte/icons/chevron-down';
	import type { LucideProps } from '@lucide/svelte';
	import type { Component } from 'svelte';

	export interface MenuItem {
		label: string;
		icon?: Component<LucideProps>;
		/** Navigates. Mutually exclusive with onSelect. */
		href?: string;
		onSelect?: () => void;
		/** Destructive items are separated and tinted. */
		danger?: boolean;
		disabled?: boolean;
		/** Starts a new group, drawn with a divider above it. */
		separated?: boolean;
	}

	interface Props {
		label: string;
		items: MenuItem[];
		variant?: ButtonVariant;
		size?: ButtonSize;
		align?: 'start' | 'end';
	}

	let { label, items, variant = 'secondary', size = 'sm', align = 'end' }: Props = $props();

	let open = $state(false);
	let container = $state<HTMLDivElement | null>(null);
	let trigger = $state<HTMLButtonElement | null>(null);
	let menu = $state<HTMLDivElement | null>(null);

	const enabled = $derived(items.filter((item) => !item.disabled));

	function focusItem(index: number) {
		const nodes = menu?.querySelectorAll<HTMLElement>('[role="menuitem"]:not([aria-disabled="true"])');
		if (!nodes?.length) return;
		const wrapped = (index + nodes.length) % nodes.length;
		nodes[wrapped]?.focus();
	}

	function currentIndex(): number {
		const nodes = [
			...(menu?.querySelectorAll<HTMLElement>('[role="menuitem"]:not([aria-disabled="true"])') ?? [])
		];
		return nodes.indexOf(document.activeElement as HTMLElement);
	}

	function close(returnFocus = true) {
		open = false;
		if (returnFocus) trigger?.focus();
	}

	function onTriggerKeydown(event: KeyboardEvent) {
		// Opening with an arrow key lands on the first or last item, which is
		// what makes the menu usable without touching the mouse at all.
		if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
			event.preventDefault();
			open = true;
			queueMicrotask(() => focusItem(event.key === 'ArrowDown' ? 0 : enabled.length - 1));
		}
	}

	function onMenuKeydown(event: KeyboardEvent) {
		switch (event.key) {
			case 'Escape':
				event.preventDefault();
				close();
				break;
			case 'ArrowDown':
				event.preventDefault();
				focusItem(currentIndex() + 1);
				break;
			case 'ArrowUp':
				event.preventDefault();
				focusItem(currentIndex() - 1);
				break;
			case 'Home':
				event.preventDefault();
				focusItem(0);
				break;
			case 'End':
				event.preventDefault();
				focusItem(enabled.length - 1);
				break;
			case 'Tab':
				// Tabbing out is a deliberate exit, so let it move on rather than
				// yanking focus back to the trigger.
				close(false);
				break;
		}
	}

	function select(item: MenuItem) {
		if (item.disabled) return;
		close();
		item.onSelect?.();
	}

	// Pointerdown rather than click: a click listener fires after the button
	// that opened a second menu has already toggled, which leaves both shut.
	$effect(() => {
		if (!open) return;

		const onPointerDown = (event: PointerEvent) => {
			if (!container?.contains(event.target as Node)) close(false);
		};
		document.addEventListener('pointerdown', onPointerDown);
		return () => document.removeEventListener('pointerdown', onPointerDown);
	});
</script>

<div class="relative" bind:this={container}>
	<button
		bind:this={trigger}
		type="button"
		class={buttonClasses(variant, size)}
		aria-haspopup="menu"
		aria-expanded={open}
		onclick={() => (open = !open)}
		onkeydown={onTriggerKeydown}
	>
		{label}
		<ChevronDown
			size={15}
			aria-hidden="true"
			class={cn('transition-transform duration-150', open && 'rotate-180')}
		/>
	</button>

	{#if open}
		<div
			bind:this={menu}
			role="menu"
			tabindex="-1"
			aria-label={label}
			onkeydown={onMenuKeydown}
			class={cn(
				'border-border bg-surface rounded-card-sm absolute z-50 mt-2 w-max min-w-56 border p-1.5 shadow-lg',
				align === 'end' ? 'right-0' : 'left-0'
			)}
		>
			{#each items as item (item.label)}
				{@const Icon = item.icon}
				{#if item.separated}
					<div class="bg-border my-1.5 h-px" role="separator"></div>
				{/if}

				{#if item.href && !item.disabled}
					<a
						role="menuitem"
						href={item.href}
						class={cn(
							'rounded-field flex items-center gap-2.5 px-3 py-2 text-sm whitespace-nowrap outline-none',
							'hover:bg-surface-sunken focus-visible:bg-surface-sunken',
							item.danger ? 'text-error' : 'text-content'
						)}
						onclick={() => close(false)}
					>
						{#if Icon}<Icon size={15} aria-hidden="true" class="shrink-0" />{/if}
						{item.label}
					</a>
				{:else}
					<button
						role="menuitem"
						type="button"
						aria-disabled={item.disabled}
						class={cn(
							'rounded-field flex w-full items-center gap-2.5 px-3 py-2 text-left text-sm whitespace-nowrap outline-none',
							item.disabled
								? 'text-content-subtle cursor-not-allowed'
								: 'hover:bg-surface-sunken focus-visible:bg-surface-sunken',
							item.danger && !item.disabled ? 'text-error' : 'text-content'
						)}
						onclick={() => select(item)}
					>
						{#if Icon}<Icon size={15} aria-hidden="true" class="shrink-0" />{/if}
						{item.label}
					</button>
				{/if}
			{/each}
		</div>
	{/if}
</div>
