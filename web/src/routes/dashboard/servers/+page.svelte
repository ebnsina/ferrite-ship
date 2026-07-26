<script lang="ts">
	import Seo from '$components/Seo.svelte';
	import BoardSkeleton from '$components/dashboard/BoardSkeleton.svelte';
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import ResourceView from '$components/dashboard/ResourceView.svelte';
	import SectionHeader from '$components/dashboard/SectionHeader.svelte';
	import ServerBoard from '$components/dashboard/ServerBoard.svelte';
	import { ButtonLink } from '$components/ui';
	import { dashboardRepository } from '$lib/data';
	import { createResource } from '$utils/resource.svelte';
	import CirclePlus from '@lucide/svelte/icons/circle-plus';

	const servers = createResource((signal) => dashboardRepository.listServers(signal));
</script>

<Seo title="Servers" description="Every machine you have connected." noindex />

<DashboardTopbar crumbs={[{ label: 'Dashboard', href: '/dashboard' }, { label: 'Servers' }]} />

<div class="space-y-6 px-6 py-8">
	<div class="flex flex-wrap items-end justify-between gap-4">
		<SectionHeader
			title="Servers"
			description="Every machine you have connected, grouped by how it is doing."
		/>
		<ButtonLink href="/dashboard/servers/new" size="sm">
			<CirclePlus size={15} aria-hidden="true" />
			Connect a server
		</ButtonLink>
	</div>

	<ResourceView resource={servers}>
		{#snippet pending()}
			<BoardSkeleton />
		{/snippet}

		{#snippet children(list)}
			<ServerBoard servers={list} />
		{/snippet}
	</ResourceView>
</div>
