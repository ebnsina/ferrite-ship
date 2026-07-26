<script lang="ts">
	import type { Crumb } from '$types/breadcrumb';

	interface Props {
		items: Crumb[];
	}

	let { items }: Props = $props();
</script>

<nav aria-label="Breadcrumb">
	<ol class="flex items-center gap-2 text-sm">
		{#each items as item, index (item.label)}
			{@const last = index === items.length - 1}
			<li class="flex items-center gap-2">
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

				{#if !last}
					<span class="text-content-subtle" aria-hidden="true">/</span>
				{/if}
			</li>
		{/each}
	</ol>
</nav>
