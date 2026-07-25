<script lang="ts">
	import { page } from '$app/state';
	import Wordmark from '$components/brand/Wordmark.svelte';
	import { dashboardNav } from '$lib/content/navigation';
	import { cn } from '$utils/cn';
	import CirclePlus from '@lucide/svelte/icons/circle-plus';

	function isActive(href: string, pathname: string): boolean {
		return href === '/dashboard' ? pathname === href : pathname.startsWith(href);
	}
</script>

<aside class="border-border/70 bg-surface/40 hidden w-64 shrink-0 border-r lg:block">
	<div class="sticky top-0 flex h-dvh flex-col">
		<div class="flex h-16 items-center px-5">
			<Wordmark href="/" />
		</div>

		<nav aria-label="Dashboard" class="flex-1 space-y-1 px-3 py-4">
			{#each dashboardNav as item (item.href)}
				{@const active = isActive(item.href, page.url.pathname)}
				{@const Icon = item.icon}
				<a
					href={item.href}
					aria-current={active ? 'page' : undefined}
					class={cn(
						'flex items-start gap-3 rounded-tile px-3 py-2.5 transition-colors duration-150',
						active
							? 'bg-accent-soft text-accent'
							: 'text-content-muted hover:bg-surface-raised hover:text-content'
					)}
				>
					<Icon size={17} aria-hidden="true" class="mt-0.5 shrink-0" />
					<span class="min-w-0">
						<span class={cn('block text-sm', active && 'font-medium')}>{item.label}</span>
						{#if item.hint}
							<span class="text-content-subtle block truncate text-xs">{item.hint}</span>
						{/if}
					</span>
				</a>
			{/each}
		</nav>

		<div class="border-border/70 border-t p-4">
			<a
				href="/dashboard/servers"
				class="text-content-muted hover:bg-surface-raised hover:text-content flex items-center gap-3 rounded-tile px-3 py-2.5 text-sm transition-colors duration-150"
			>
				<CirclePlus size={17} aria-hidden="true" />
				Connect a server
			</a>
		</div>
	</div>
</aside>
