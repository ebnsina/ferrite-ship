<script lang="ts">
	import ActivityFeed from '$components/dashboard/ActivityFeed.svelte';
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import ResourceView from '$components/dashboard/ResourceView.svelte';
	import SectionHeader from '$components/dashboard/SectionHeader.svelte';
	import { Skeleton } from '$components/ui';
	import { dashboardRepository } from '$lib/data';
	import { createResource } from '$utils/resource.svelte';

	const activity = createResource((signal) => dashboardRepository.listActivity(signal));
</script>

<svelte:head>
	<title>History · ferrite-ship</title>
</svelte:head>

<DashboardTopbar
	crumbs={[{ label: 'Dashboard', href: '/dashboard' }, { label: 'History' }]}
	unread={2}
/>

<div class="space-y-6 px-6 py-8">
	<SectionHeader
		title="History"
		description="Everything that has run, who started it, and whether it worked."
	/>

	<ResourceView resource={activity}>
		{#snippet pending()}
			<Skeleton shape="card" class="h-96" />
		{/snippet}

		{#snippet children(entries)}
			<ActivityFeed {entries} />
		{/snippet}
	</ResourceView>
</div>
