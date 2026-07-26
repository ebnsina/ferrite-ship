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
	import { formatNumber } from '$utils/format';
	import { createResource } from '$utils/resource.svelte';
	import CalendarDays from '@lucide/svelte/icons/calendar-days';
	import CirclePlus from '@lucide/svelte/icons/circle-plus';
	import SlidersHorizontal from '@lucide/svelte/icons/sliders-horizontal';

	const servers = createResource((signal) => dashboardRepository.listServers(signal));
	const activity = createResource((signal) => dashboardRepository.listActivity(signal));
	const metrics = createResource((signal) => dashboardRepository.listMetrics(signal));

	// The headline reports something we actually measure. An uptime percentage
	// would need history we do not collect yet, and a made-up figure on the
	// first screen would undermine everything under it.
	const total = $derived(metrics.data?.find((m) => m.id === 'servers')?.value ?? null);
	const online = $derived(metrics.data?.find((m) => m.id === 'online')?.value ?? null);

	const headline = $derived(
		total === null || online === null ? '—' : `${formatNumber(online)} of ${formatNumber(total)}`
	);

	const heading = $derived(
		total === 0
			? 'No servers yet'
			: total !== null && online === total
				? 'Everything is looking good'
				: 'Some servers need a look'
	);
</script>

<svelte:head>
	<title>Overview · ferrite-ship</title>
</svelte:head>

<DashboardTopbar crumbs={[{ label: 'Dashboard', href: '/dashboard' }, { label: 'Overview' }]} unread={2} />

<div class="space-y-8 px-6 py-8">
	<PageHeading
		title={heading}
		metric={headline}
		caption="of your servers are running fine right now"
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

		<a
			href="/dashboard/servers/new"
			class="text-content-muted hover:text-content flex items-center gap-2 text-sm transition-colors duration-150"
		>
			<CirclePlus size={16} aria-hidden="true" />
			Connect a server
		</a>
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
