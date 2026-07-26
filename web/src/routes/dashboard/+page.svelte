<script lang="ts">
	import ActivityFeed from '$components/dashboard/ActivityFeed.svelte';
	import BoardSkeleton from '$components/dashboard/BoardSkeleton.svelte';
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import PageHeading from '$components/dashboard/PageHeading.svelte';
	import ResourceView from '$components/dashboard/ResourceView.svelte';
	import SectionHeader from '$components/dashboard/SectionHeader.svelte';
	import ServerBoard from '$components/dashboard/ServerBoard.svelte';
	import StatStrip from '$components/dashboard/StatStrip.svelte';
	import StatStripSkeleton from '$components/dashboard/StatStripSkeleton.svelte';
	import { Button, Skeleton } from '$components/ui';
	import { dashboardRepository } from '$lib/data';
	import { formatPercent } from '$utils/format';
	import { createResource } from '$utils/resource.svelte';
	import CalendarDays from '@lucide/svelte/icons/calendar-days';
	import CirclePlus from '@lucide/svelte/icons/circle-plus';
	import SlidersHorizontal from '@lucide/svelte/icons/sliders-horizontal';

	const servers = createResource((signal) => dashboardRepository.listServers(signal));
	const activity = createResource((signal) => dashboardRepository.listActivity(signal));
	const metrics = createResource((signal) => dashboardRepository.listMetrics(signal));

	const uptime = $derived(metrics.data?.find((metric) => metric.id === 'uptime') ?? null);
</script>

<svelte:head>
	<title>Overview · ferrite-ship</title>
</svelte:head>

<DashboardTopbar crumbs={[{ label: 'Dashboard', href: '/dashboard' }, { label: 'Overview' }]} unread={2} />

<div class="space-y-8 px-6 py-8">
	<PageHeading
		title="Everything is looking good"
		metric={uptime ? formatPercent(uptime.value, { fractionDigits: 1 }) : '—'}
		caption="of the time your servers were up and running this week"
	>
		{#snippet actions()}
			<Button variant="secondary" size="sm">
				<CalendarDays size={15} aria-hidden="true" />
				Last 7 days
			</Button>
			<Button variant="secondary" size="sm">
				<SlidersHorizontal size={15} aria-hidden="true" />
				Filters
			</Button>
		{/snippet}
	</PageHeading>

	<ResourceView resource={metrics}>
		{#snippet pending()}
			<StatStripSkeleton />
		{/snippet}

		{#snippet children(list)}
			<StatStrip metrics={list} />
		{/snippet}
	</ResourceView>

	<section class="space-y-5">
		<SectionHeader
			title="Your servers"
			description="Grouped by how they are doing. The bar shows how full each one's storage is."
			linkHref="/dashboard/servers"
		/>

		<ResourceView resource={servers}>
			{#snippet pending()}
				<BoardSkeleton />
			{/snippet}

			{#snippet children(list)}
				<ServerBoard servers={list} />
			{/snippet}
		</ResourceView>

		<button
			type="button"
			class="text-content-muted hover:text-content flex items-center gap-2 text-sm transition-colors duration-150"
		>
			<CirclePlus size={16} aria-hidden="true" />
			Connect a server
		</button>
	</section>

	<section class="space-y-5">
		<SectionHeader
			title="Recent activity"
			description="What has run lately, who started it, and whether it worked."
			linkHref="/dashboard/activity"
		/>

		<ResourceView resource={activity}>
			{#snippet pending()}
				<Skeleton shape="card" class="h-72" />
			{/snippet}

			{#snippet children(entries)}
				<ActivityFeed {entries} />
			{/snippet}
		</ResourceView>
	</section>
</div>
