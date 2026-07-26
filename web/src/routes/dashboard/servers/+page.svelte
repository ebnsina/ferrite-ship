<script lang="ts">
	import Seo from '$components/Seo.svelte';
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import ResourceView from '$components/dashboard/ResourceView.svelte';
	import SectionHeader from '$components/dashboard/SectionHeader.svelte';
	import ServerTable from '$components/dashboard/ServerTable.svelte';
	import { ButtonLink, Card, EmptyState, SearchInput, Skeleton } from '$components/ui';
	import { dashboardRepository } from '$lib/data';
	import { SERVER_STATUS } from '$lib/domain/status';
	import type { ServerStatus } from '$types/server';
	import { formatNumber } from '$utils/format';
	import { createResource } from '$utils/resource.svelte';
	import CirclePlus from '@lucide/svelte/icons/circle-plus';
	import ServerIcon from '@lucide/svelte/icons/server';

	const servers = createResource((signal) => dashboardRepository.listServers(signal));

	let query = $state('');
	let onlyStatus = $state<ServerStatus | 'all'>('all');

	// Counts come from the whole fleet, not the filtered view: a filter that
	// says "3" and then shows 3 tells you nothing about what you filtered out.
	const counts = $derived.by(() => {
		const tally = new Map<ServerStatus, number>();
		for (const server of servers.data ?? []) {
			tally.set(server.status, (tally.get(server.status) ?? 0) + 1);
		}
		return tally;
	});

	const visible = $derived.by(() => {
		const needle = query.trim().toLowerCase();

		return (servers.data ?? []).filter((server) => {
			if (onlyStatus !== 'all' && server.status !== onlyStatus) return false;
			if (!needle) return true;
			return (
				server.name.toLowerCase().includes(needle) ||
				server.ipAddress.toLowerCase().includes(needle) ||
				server.region.toLowerCase().includes(needle)
			);
		});
	});

	// Only offer a filter for a state something is actually in. A row of
	// buttons that all say zero is a row of dead ends.
	const filters = $derived(
		(['online', 'degraded', 'provisioning', 'offline', 'unknown'] as ServerStatus[]).filter(
			(status) => (counts.get(status) ?? 0) > 0
		)
	);
</script>

<Seo title="Servers" description="Every machine you have connected." noindex />

<DashboardTopbar crumbs={[{ label: 'Dashboard', href: '/dashboard' }, { label: 'Servers' }]} />

<div class="space-y-5 px-6 py-8">
	<div class="flex flex-wrap items-end justify-between gap-4">
		<SectionHeader title="Servers" description="Every machine you have connected." />
		<ButtonLink href="/dashboard/servers/new" size="sm">
			<CirclePlus size={15} aria-hidden="true" />
			Connect a server
		</ButtonLink>
	</div>

	<ResourceView resource={servers}>
		{#snippet pending()}
			<Skeleton shape="card" class="h-96" />
		{/snippet}

		{#snippet children(list)}
			{#if list.length === 0}
				<Card size="lg" padded={false} class="overflow-hidden">
					<EmptyState
						icon={ServerIcon}
						title="No servers yet"
						description="Connect a machine and it will show up here within a few seconds."
					>
						{#snippet action()}
							<ButtonLink href="/dashboard/servers/new" size="sm">
								Connect your first server
							</ButtonLink>
						{/snippet}
					</EmptyState>
				</Card>
			{:else}
				<div class="flex flex-wrap items-center gap-3">
					<SearchInput bind:value={query} placeholder="Search by name, address or location" />

					<div class="flex flex-wrap items-center gap-1.5">
						<button
							type="button"
							onclick={() => (onlyStatus = 'all')}
							class="rounded-pill px-3 py-1 text-xs font-medium transition-colors duration-150
								{onlyStatus === 'all'
								? 'bg-surface-inverse text-content-inverse'
								: 'text-content-muted hover:bg-surface-sunken'}"
						>
							All {formatNumber(list.length)}
						</button>

						{#each filters as status (status)}
							<button
								type="button"
								onclick={() => (onlyStatus = status)}
								class="rounded-pill px-3 py-1 text-xs font-medium transition-colors duration-150
									{onlyStatus === status
									? 'bg-surface-inverse text-content-inverse'
									: 'text-content-muted hover:bg-surface-sunken'}"
							>
								{SERVER_STATUS[status].label}
								{formatNumber(counts.get(status) ?? 0)}
							</button>
						{/each}
					</div>

					<p class="text-content-subtle ml-auto text-xs">
						Showing {formatNumber(visible.length)} of {formatNumber(list.length)}
					</p>
				</div>

				{#if visible.length === 0}
					<Card size="lg" padded={false} class="overflow-hidden">
						<EmptyState
							icon={ServerIcon}
							title="Nothing matches that"
							description="Try a different name, address or state."
						/>
					</Card>
				{:else}
					<ServerTable servers={visible} />
				{/if}
			{/if}
		{/snippet}
	</ResourceView>
</div>
