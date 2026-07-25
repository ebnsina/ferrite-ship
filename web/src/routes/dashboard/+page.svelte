<script lang="ts">
	import ActivityFeed from '$components/dashboard/ActivityFeed.svelte';
	import CardSkeletonGrid from '$components/dashboard/CardSkeletonGrid.svelte';
	import DashboardSection from '$components/dashboard/DashboardSection.svelte';
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import FleetStats from '$components/dashboard/FleetStats.svelte';
	import ResourceView from '$components/dashboard/ResourceView.svelte';
	import ServerGrid from '$components/dashboard/ServerGrid.svelte';
	import StatusLegend from '$components/dashboard/StatusLegend.svelte';
	import { Skeleton } from '$components/ui';
	import { dashboardRepository } from '$lib/data';
	import { createResource } from '$utils/resource.svelte';
	import History from '@lucide/svelte/icons/history';
	import LayoutDashboard from '@lucide/svelte/icons/layout-dashboard';
	import Server from '@lucide/svelte/icons/server';

	const servers = createResource((signal) => dashboardRepository.listServers(signal));
	const activity = createResource((signal) => dashboardRepository.listActivity(signal));
</script>

<svelte:head>
	<title>Overview · ferrite-ship</title>
</svelte:head>

<DashboardTopbar
	title="Overview"
	description="A quick look at how your servers are doing and what has happened recently."
	icon={LayoutDashboard}
/>

<div class="space-y-10 px-6 py-8">
	<ResourceView resource={servers}>
		{#snippet pending()}
			<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
				{#each Array.from({ length: 4 }, (_, index) => index) as index (index)}
					<Skeleton shape="card" class="h-28" />
				{/each}
			</div>
		{/snippet}

		{#snippet children(list)}
			<FleetStats servers={list} />
		{/snippet}
	</ResourceView>

	<DashboardSection
		title="Your servers"
		description="Each card shows how hard a server is working. The bars fill up as it gets busier."
		icon={Server}
	>
		{#snippet aside()}
			<StatusLegend />
		{/snippet}

		<ResourceView resource={servers}>
			{#snippet pending()}
				<CardSkeletonGrid count={3} />
			{/snippet}

			{#snippet children(list)}
				<ServerGrid servers={list} />
			{/snippet}
		</ResourceView>
	</DashboardSection>

	<DashboardSection
		title="Recent activity"
		description="Everything that has run lately, who started it, and whether it worked."
		icon={History}
	>
		<ResourceView resource={activity}>
			{#snippet pending()}
				<Skeleton shape="card" class="h-72" />
			{/snippet}

			{#snippet children(entries)}
				<ActivityFeed {entries} />
			{/snippet}
		</ResourceView>
	</DashboardSection>
</div>
