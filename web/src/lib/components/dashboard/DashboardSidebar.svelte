<script lang="ts">
	import { page } from '$app/state';
	import LogoMark from '$components/brand/LogoMark.svelte';
	import { dashboardNavGroups } from '$lib/content/navigation';
	import { cn } from '$utils/cn';
	import CirclePlus from '@lucide/svelte/icons/circle-plus';
	import PanelLeft from '@lucide/svelte/icons/panel-left';

	let collapsed = $state(false);

	function isActive(href: string, pathname: string): boolean {
		return href === '/dashboard' ? pathname === href : pathname.startsWith(href);
	}
</script>

<aside
	class={cn(
		'border-border/70 hidden shrink-0 border-r transition-[width] duration-200 ease-snappy lg:block',
		collapsed ? 'w-[4.5rem]' : 'w-60'
	)}
>
	<div class="sticky top-0 flex h-dvh flex-col">
		<div class="flex h-16 items-center gap-2 px-4">
			<a href="/" class="flex items-center gap-2.5 overflow-hidden" aria-label="ferrite-ship home">
				<LogoMark size={30} />
				{#if !collapsed}
					<span class="text-content truncate text-[0.95rem] font-semibold tracking-tight">
						ferrite<span class="text-content-subtle">·</span>ship
					</span>
				{/if}
			</a>

			{#if !collapsed}
				<button
					type="button"
					onclick={() => (collapsed = true)}
					class="text-content-subtle hover:bg-surface-raised hover:text-content ml-auto inline-flex size-8 items-center justify-center rounded-tile transition-colors duration-150"
					aria-label="Collapse sidebar"
				>
					<PanelLeft size={16} aria-hidden="true" />
				</button>
			{/if}
		</div>

		{#if collapsed}
			<button
				type="button"
				onclick={() => (collapsed = false)}
				class="text-content-subtle hover:bg-surface-raised hover:text-content mx-auto inline-flex size-8 items-center justify-center rounded-tile transition-colors duration-150"
				aria-label="Expand sidebar"
			>
				<PanelLeft size={16} aria-hidden="true" />
			</button>
		{/if}

		<nav aria-label="Dashboard" class="flex-1 space-y-6 overflow-y-auto px-3 py-4">
			{#each dashboardNavGroups as group (group.label)}
				<div>
					{#if !collapsed}
						<h2 class="text-content-subtle px-3 pb-2 text-xs font-medium">{group.label}</h2>
					{/if}

					<ul class="space-y-0.5">
						{#each group.items as item (item.href)}
							{@const active = isActive(item.href, page.url.pathname)}
							{@const Icon = item.icon}
							<li>
								<a
									href={item.href}
									aria-current={active ? 'page' : undefined}
									title={collapsed ? item.label : undefined}
									class={cn(
										'flex items-center gap-3 rounded-tile px-3 py-2 text-sm transition-colors duration-150',
										collapsed && 'justify-center px-0',
										active
											? 'bg-surface-sunken text-content font-medium'
											: 'text-content-muted hover:bg-surface-sunken hover:text-content'
									)}
								>
									<Icon size={17} aria-hidden="true" class="shrink-0" />
									{#if !collapsed}
										<span class="truncate">{item.label}</span>
									{/if}
								</a>
							</li>
						{/each}
					</ul>
				</div>
			{/each}
		</nav>

		<div class="border-border/70 border-t p-3">
			<a
				href="/dashboard/servers"
				title={collapsed ? 'Connect a server' : undefined}
				class={cn(
					'text-content-muted hover:bg-surface-sunken hover:text-content flex items-center gap-3 rounded-tile px-3 py-2 text-sm transition-colors duration-150',
					collapsed && 'justify-center px-0'
				)}
			>
				<CirclePlus size={17} aria-hidden="true" class="shrink-0" />
				{#if !collapsed}
					<span class="truncate">Connect a server</span>
				{/if}
			</a>
		</div>
	</div>
</aside>
