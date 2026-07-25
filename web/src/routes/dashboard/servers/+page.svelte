<script lang="ts">
	import CardSkeletonGrid from '$components/dashboard/CardSkeletonGrid.svelte';
	import DashboardSection from '$components/dashboard/DashboardSection.svelte';
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import ResourceView from '$components/dashboard/ResourceView.svelte';
	import ServerGrid from '$components/dashboard/ServerGrid.svelte';
	import StatusLegend from '$components/dashboard/StatusLegend.svelte';
	import { dashboardRepository } from '$lib/data';
	import { createResource } from '$utils/resource.svelte';
	import Server from '@lucide/svelte/icons/server';

	const servers = createResource((signal) => dashboardRepository.listServers(signal));
</script>

<svelte:head>
	<title>Servers · ferrite-ship</title>
</svelte:head>

<DashboardTopbar
	title="Servers"
	description="Every machine you have connected. Pick one to manage its files, apps and settings."
	icon={Server}
/>

<div class="px-6 py-8">
	<DashboardSection
		title="Connected servers"
		description="The bars show how much of each server is being used right now."
		icon={Server}
	>
		{#snippet aside()}
			<StatusLegend />
		{/snippet}

		<ResourceView resource={servers}>
			{#snippet pending()}
				<CardSkeletonGrid count={6} />
			{/snippet}

			{#snippet children(list)}
				<ServerGrid servers={list} />
			{/snippet}
		</ResourceView>
	</DashboardSection>
</div>
