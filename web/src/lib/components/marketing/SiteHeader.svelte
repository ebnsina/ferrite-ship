<script lang="ts">
	import Wordmark from '$components/brand/Wordmark.svelte';
	import { ButtonLink, ThemeToggle } from '$components/ui';
	import { marketingNav } from '$lib/content/navigation';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
</script>

<!-- No bottom border: the header sits over the hero's accent wash and blends. -->
<header class="supports-[backdrop-filter]:bg-canvas/50 bg-canvas/80 sticky top-0 z-40 backdrop-blur-xl">
	<div class="mx-auto flex h-16 max-w-6xl items-center gap-6 px-5">
		<Wordmark />

		<nav aria-label="Primary" class="hidden md:flex md:items-center md:gap-1">
			{#each marketingNav as item (item.href)}
				{@const Icon = item.icon}
				<a
					href={item.href}
					class="text-content-muted hover:bg-surface-raised hover:text-content flex items-center gap-2 rounded-tile px-3.5 py-2 text-sm transition-colors duration-150"
				>
					<Icon size={15} aria-hidden="true" />
					{item.label}
				</a>
			{/each}
		</nav>

		<div class="ml-auto flex items-center gap-2">
			<ThemeToggle />
			<ButtonLink href="/dashboard" size="sm">
				Open dashboard
				<ArrowRight size={15} aria-hidden="true" />
			</ButtonLink>
		</div>
	</div>
</header>
