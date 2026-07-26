<script lang="ts">
	import Seo from '$components/Seo.svelte';
	import ActivityFeed from '$components/dashboard/ActivityFeed.svelte';
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import ResourceView from '$components/dashboard/ResourceView.svelte';
	import SectionHeader from '$components/dashboard/SectionHeader.svelte';
	import { Skeleton } from '$components/ui';
	import { dashboardRepository } from '$lib/data';
	import { createResource } from '$utils/resource.svelte';

	const activity = createResource((signal) => dashboardRepository.listActivity(signal));
</script>

<Seo title="History" description="Everything that has run, and how it went." noindex />

<DashboardTopbar
	crumbs={[{ label: 'Dashboard', href: '/dashboard' }, { label: 'History' }]}
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
