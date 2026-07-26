<script lang="ts">
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import ResourceView from '$components/dashboard/ResourceView.svelte';
	import SectionHeader from '$components/dashboard/SectionHeader.svelte';
	import ToolsSkeleton from '$components/dashboard/ToolsSkeleton.svelte';
	import Seo from '$components/Seo.svelte';
	import { ButtonLink, Card, EmptyState, IconTile, Skeleton, StatusPill } from '$components/ui';
	import { dashboardRepository } from '$lib/data';
	import { toolsClient } from '$lib/data/tools';
	import { SERVER_STATUS } from '$lib/domain/status';
	import { byCategory, reachability, toolIcon } from '$lib/domain/tools';
	import { createResource } from '$utils/resource.svelte';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import Globe from '@lucide/svelte/icons/globe';
	import Lock from '@lucide/svelte/icons/lock';
	import Server from '@lucide/svelte/icons/server';

	// The catalogue is the same wherever you install it; what differs is which
	// server you put it on. So this page answers "what can I have?" and then
	// hands over to the server that will run it.
	const catalog = createResource((signal) => toolsClient.catalog(signal));
	const servers = createResource((signal) => dashboardRepository.listServers(signal));

	// Only real servers: there is nothing behind a simulated one to install on.
	const installable = $derived((servers.data ?? []).filter((s) => s.connectionKind === 'ssh'));
</script>

<Seo
	title="Tools"
	description="Databases, caches and streaming you can add to any server."
	noindex
/>

<DashboardTopbar crumbs={[{ label: 'Dashboard', href: '/dashboard' }, { label: 'Tools' }]} />

<div class="space-y-8 px-6 py-8">
	<SectionHeader
		title="Tools"
		description="Software we install and keep running for you. Pick a server to add one to."
	/>

	<section class="space-y-4">
		<h2 class="text-content text-sm font-medium">Choose a server</h2>

		<ResourceView resource={servers}>
			{#snippet pending()}
				<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
					<Skeleton shape="card" class="h-28" />
					<Skeleton shape="card" class="h-28" />
				</div>
			{/snippet}

			{#snippet children()}
				{#if installable.length === 0}
					<Card padded={false} class="overflow-hidden">
						<EmptyState
							icon={Server}
							title="No servers to install on yet"
							description="Connect a real server and it will show up here, ready to have things added to it."
						>
							{#snippet action()}
								<ButtonLink href="/dashboard/servers/new" variant="secondary" size="sm">
									Connect a server
								</ButtonLink>
							{/snippet}
						</EmptyState>
					</Card>
				{:else}
					<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
						{#each installable as server (server.id)}
							<a href="/dashboard/servers/{server.id}/tools" class="rounded-card block">
								<Card interactive class="h-full">
									<div class="flex items-start justify-between gap-3">
										<div class="flex min-w-0 items-start gap-3">
											<IconTile icon={Server} tone={SERVER_STATUS[server.status].tone} size="sm" />
											<div class="min-w-0">
												<p class="text-content truncate text-sm font-medium">{server.name}</p>
												<p class="text-content-subtle font-machine mt-0.5 truncate text-xs">
													{server.ipAddress}
												</p>
											</div>
										</div>
										<ArrowRight size={15} aria-hidden="true" class="text-content-subtle mt-1.5" />
									</div>
									<div class="mt-4">
										<StatusPill status={SERVER_STATUS[server.status]} />
									</div>
								</Card>
							</a>
						{/each}
					</div>
				{/if}
			{/snippet}
		</ResourceView>
	</section>

	<section class="space-y-6">
		<h2 class="text-content text-sm font-medium">What you can install</h2>

		<ResourceView resource={catalog}>
			{#snippet pending()}
				<ToolsSkeleton />
			{/snippet}

			{#snippet children(tools)}
				{#each byCategory(tools) as group (group.category)}
					<div class="space-y-3">
						<h3 class="text-content-muted text-xs font-medium tracking-wide uppercase">
							{group.category}
						</h3>

						<div class="grid gap-4 md:grid-cols-2">
							{#each group.tools as tool (tool.id)}
								{@const Icon = toolIcon(tool.icon)}
								{@const isPublic = tool.ports.some((port) => port.public)}
								<Card>
									<div class="flex items-start gap-4">
										<IconTile icon={Icon} />
										<div class="min-w-0">
											<div class="flex flex-wrap items-center gap-x-2">
												<h4 class="text-content text-sm font-medium">{tool.name}</h4>
												<span class="text-content-subtle text-xs">{tool.version}</span>
											</div>
											<p class="text-content-muted mt-1.5 text-sm leading-relaxed">
												{tool.summary}
											</p>
											<p class="text-content-subtle mt-2 flex items-center gap-1.5 text-xs">
												{#if isPublic}
													<Globe size={13} aria-hidden="true" />
												{:else}
													<Lock size={13} aria-hidden="true" />
												{/if}
												{reachability(tool).label}
											</p>
										</div>
									</div>
								</Card>
							{/each}
						</div>
					</div>
				{/each}
			{/snippet}
		</ResourceView>
	</section>
</div>
