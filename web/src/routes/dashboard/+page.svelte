<script lang="ts">
	import Seo from '$components/Seo.svelte';
	import BoardSkeleton from '$components/dashboard/BoardSkeleton.svelte';
	import DashboardGreeting from '$components/dashboard/DashboardGreeting.svelte';
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import ResourceView from '$components/dashboard/ResourceView.svelte';
	import SectionHeader from '$components/dashboard/SectionHeader.svelte';
	import ServerBoard from '$components/dashboard/ServerBoard.svelte';
	import StatStrip from '$components/dashboard/StatStrip.svelte';
	import StatStripSkeleton from '$components/dashboard/StatStripSkeleton.svelte';
	import { ButtonLink } from '$components/ui';
	import { dashboardRepository } from '$lib/data';
	import { createResource } from '$utils/resource.svelte';
	import CirclePlus from '@lucide/svelte/icons/circle-plus';

	/*
	 * This page answers one question: how are my servers right now.
	 *
	 * It used to also carry the recent-activity feed, which is a different
	 * question — what happened lately — and which already has a page of its
	 * own. Two feeds of the same rows in two places is two things to keep in
	 * step and one more screen to scroll past.
	 */
	const servers = createResource((signal) => dashboardRepository.listServers(signal));
	const metrics = createResource((signal) => dashboardRepository.listMetrics(signal));
</script>

<Seo title="Overview" description="How your servers are doing right now." noindex />

<DashboardTopbar crumbs={[{ label: 'Dashboard', href: '/dashboard' }, { label: 'Overview' }]} />

<div class="space-y-8 px-6 py-8">
	<DashboardGreeting>
		{#snippet actions()}
			<!-- A real action rather than the date-range and filter controls that
			     used to sit here, which were never wired to anything. -->
			<ButtonLink href="/dashboard/servers/new" size="sm">
				<CirclePlus size={15} aria-hidden="true" />
				Connect a server
			</ButtonLink>
		{/snippet}
	</DashboardGreeting>

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
	</section>
</div>
