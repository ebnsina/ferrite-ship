<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import LogoMark from '$components/brand/LogoMark.svelte';
	import { dashboardNavGroups } from '$lib/content/navigation';
	import { dashboardRepository } from '$lib/data';
	import {
		activeSection,
		serverIdFromPath,
		serverSectionHref,
		serverSections
	} from '$lib/domain/server-nav';
	import { cn } from '$utils/cn';
	import { authClient } from '$lib/data/auth';
	import CirclePlus from '@lucide/svelte/icons/circle-plus';
	import LogOut from '@lucide/svelte/icons/log-out';
	import PanelLeft from '@lucide/svelte/icons/panel-left';

	let collapsed = $state(false);
	let email = $state<string | null>(null);
	let signingOut = $state(false);

	async function signOut() {
		if (signingOut) return;
		signingOut = true;
		try {
			await authClient.logout();
		} finally {
			// Leave regardless: if the call failed the cookie may still be gone,
			// and stranding someone on a dashboard they cannot use is worse.
			await goto('/login');
		}
	}

	$effect(() => {
		void authClient
			.status()
			.then((status) => (email = status.email ?? null))
			.catch(() => (email = null));
	});

	function isActive(href: string, pathname: string): boolean {
		return href === '/dashboard' ? pathname === href : pathname.startsWith(href);
	}

	/*
	 * The server you are looking at gets its own section.
	 *
	 * Without it the sidebar showed six links regardless of where you were,
	 * while the six pages belonging to a server were reachable only from that
	 * server's header — so moving from its files to its terminal meant going
	 * back first. It also left the sidebar mostly empty, which is the visible
	 * symptom of the same thing: the navigation was not describing where you
	 * actually were.
	 */
	const currentServerId = $derived(serverIdFromPath(page.url.pathname));
	const current = $derived(currentServerId ? activeSection(page.url.pathname, currentServerId) : null);

	let serverName = $state<string | null>(null);

	$effect(() => {
		const id = currentServerId;
		if (!id) {
			serverName = null;
			return;
		}

		// The name is nice to have, not load-bearing: the id is enough to
		// navigate, so a failure here leaves the links working.
		void dashboardRepository
			.getServer(id)
			.then((server) => (serverName = server.name))
			.catch(() => (serverName = null));
	});
</script>

<aside
	class={cn(
		'border-border/70 hidden shrink-0 border-r transition-[width] duration-200 ease-snappy lg:block',
		collapsed ? 'w-[4.5rem]' : 'w-60'
	)}
>
	<div class="sticky top-0 flex h-dvh flex-col">
		<div class="flex h-16 items-center gap-2 px-4">
			<a href="/" class="flex items-center" aria-label="Ferrite Ship home">
				<LogoMark size={30} />
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
			{#if currentServerId}
				<div>
					{#if !collapsed}
						<h2 class="text-content-subtle truncate px-3 pb-2 text-xs font-medium">
							{serverName ?? 'This server'}
						</h2>
					{/if}

					<ul class="space-y-0.5">
						{#each serverSections as section (section.path)}
							{@const href = serverSectionHref(currentServerId, section)}
							{@const active = current?.path === section.path}
							{@const Icon = section.icon}
							<li>
								<a
									{href}
									aria-current={active ? 'page' : undefined}
									title={collapsed ? section.label : section.description}
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
										<span class="truncate">{section.label}</span>
									{/if}
								</a>
							</li>
						{/each}
					</ul>
				</div>
			{/if}

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

		<div class="border-border/70 space-y-0.5 border-t p-3">
			{#if email && !collapsed}
				<p class="text-content-subtle truncate px-3 pb-1 text-xs" title={email}>{email}</p>
			{/if}

			<button
				type="button"
				onclick={signOut}
				disabled={signingOut}
				title={collapsed ? 'Sign out' : undefined}
				class={cn(
					'text-content-muted hover:bg-surface-sunken hover:text-content flex w-full items-center gap-3 rounded-tile px-3 py-2 text-sm transition-colors duration-150',
					collapsed && 'justify-center px-0'
				)}
			>
				<LogOut size={17} aria-hidden="true" class="shrink-0" />
				{#if !collapsed}
					<span class="truncate">{signingOut ? 'Signing out…' : 'Sign out'}</span>
				{/if}
			</button>

			<a
				href="/dashboard/servers/new"
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
