<script lang="ts">
	import { cn } from '$utils/cn';
	import type { Crumb } from '$types/breadcrumb';
	import ChevronDown from '@lucide/svelte/icons/chevron-down';

	interface Props {
		items: Crumb[];
	}

	let { items }: Props = $props();

	// Which crumb's menu is open, by index. One at a time.
	let open = $state<number | null>(null);

	$effect(() => {
		if (open === null) return;

		const close = () => (open = null);
		// Pointerdown rather than click: a click listener fires after the
		// button that opened another menu has toggled, leaving both shut.
		document.addEventListener('pointerdown', close);
		return () => document.removeEventListener('pointerdown', close);
	});
</script>

<nav aria-label="Breadcrumb">
	<ol class="flex items-center gap-2 text-sm">
		{#each items as item, index (item.label)}
			{@const last = index === items.length - 1}
			<li class="flex items-center gap-1.5">
				{#if item.href && !last}
					<a
						href={item.href}
						class="text-content-muted hover:text-content transition-colors duration-150"
					>
						{item.label}
					</a>
				{:else}
					<span class={last ? 'text-content font-medium' : 'text-content-muted'}>
						{item.label}
					</span>
				{/if}

				{#if item.siblings && item.siblings.length > 0}
					<div class="relative">
						<button
							type="button"
							aria-haspopup="menu"
							aria-expanded={open === index}
							aria-label="Go somewhere else from {item.label}"
							onpointerdown={(event) => event.stopPropagation()}
							onclick={() => (open = open === index ? null : index)}
							class="text-content-subtle hover:bg-surface-sunken hover:text-content inline-flex size-5 items-center justify-center rounded-[0.375rem] transition-colors duration-150"
						>
							<ChevronDown
								size={13}
								aria-hidden="true"
								class={cn('transition-transform duration-150', open === index && 'rotate-180')}
							/>
						</button>

						{#if open === index}
							<div
								role="menu"
								aria-label={item.label}
								class="border-border bg-surface rounded-card-sm absolute left-0 z-50 mt-2 w-max min-w-44 border p-1.5 shadow-lg"
							>
								{#each item.siblings as sibling (sibling.href)}
									<a
										role="menuitem"
										href={sibling.href}
										class="text-content-muted hover:bg-surface-sunken hover:text-content rounded-field block px-3 py-2 text-sm whitespace-nowrap transition-colors duration-150"
									>
										{sibling.label}
									</a>
								{/each}
							</div>
						{/if}
					</div>
				{/if}

				{#if !last}
					<span class="text-content-subtle" aria-hidden="true">/</span>
				{/if}
			</li>
		{/each}
	</ol>
</nav>
