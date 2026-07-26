<script lang="ts">
	import { page } from '$app/state';
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import { ButtonLink } from '$components/ui';
	import { codeForStatus, describe } from '$lib/errors';
	import { formatNumber } from '$utils/format';
	import LayoutDashboard from '@lucide/svelte/icons/layout-dashboard';

	/**
	 * A dashboard-scoped error page. Without this, a bad URL under /dashboard
	 * falls through to the root error page, which renders outside the theme
	 * scope and flips the user from a light console to a dark full-page error.
	 */
	const fallback = $derived(describe(codeForStatus(page.status)));
	const heading = $derived(
		page.status === 404 ? 'We cannot find that page' : fallback.message
	);
</script>

<svelte:head>
	<title>{page.status} · ferrite-ship</title>
</svelte:head>

<DashboardTopbar crumbs={[{ label: 'Dashboard', href: '/dashboard' }, { label: 'Not found' }]} />

<div class="flex flex-1 items-center justify-center px-6 py-24">
	<div class="max-w-sm text-center">
		<p class="text-content-subtle text-4xl font-semibold" data-numeric>
			{formatNumber(page.status, { useGrouping: false })}
		</p>
		<h1 class="text-content mt-4 text-xl font-semibold tracking-tight">{heading}</h1>
		<p class="text-content-muted mt-2 text-sm">{fallback.action}</p>

		<div class="mt-7 flex justify-center">
			<ButtonLink href="/dashboard" size="sm">
				<LayoutDashboard size={15} aria-hidden="true" />
				Back to overview
			</ButtonLink>
		</div>
	</div>
</div>
