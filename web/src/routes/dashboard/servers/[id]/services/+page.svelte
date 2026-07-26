<script lang="ts">
	import { page } from '$app/state';
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import ServiceLogs from '$components/dashboard/ServiceLogs.svelte';
	import ServiceTable from '$components/dashboard/ServiceTable.svelte';
	import ServiceTableSkeleton from '$components/dashboard/ServiceTableSkeleton.svelte';
	import Seo from '$components/Seo.svelte';
	import { ButtonLink, Card, ErrorState, SearchInput } from '$components/ui';
	import { dashboardRepository } from '$lib/data';
	import { servicesClient, type ServiceAction, type ServiceUnit } from '$lib/data/services';
	import { toAppError, type AppError } from '$lib/errors';
	import { formatNumber } from '$utils/format';
	import { createResource } from '$utils/resource.svelte';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';

	const serverId = page.params.id ?? '';
	const server = createResource((signal) => dashboardRepository.getServer(serverId, signal));

	let units = $state<ServiceUnit[]>([]);
	let loading = $state(true);
	let error = $state<AppError | null>(null);

	let query = $state('');
	let runningOnly = $state(true);
	let busyUnit = $state<string | null>(null);
	let actionError = $state<string | null>(null);
	let logsFor = $state<string | null>(null);

	async function load() {
		loading = true;
		error = null;

		try {
			units = await servicesClient.list(serverId);
		} catch (cause) {
			error = toAppError(cause);
		} finally {
			loading = false;
		}
	}

	async function act(unit: ServiceUnit, action: ServiceAction) {
		busyUnit = unit.name;
		actionError = null;

		try {
			await servicesClient.perform(serverId, unit.name, action);
			// systemd takes a moment to settle, so re-read rather than guessing
			// what the new state should be.
			await new Promise((resolve) => setTimeout(resolve, 600));
			await load();
		} catch (cause) {
			actionError = toAppError(cause).message;
		} finally {
			busyUnit = null;
		}
	}

	// A stock Ubuntu box has ~175 units and most are one-shot boot tasks that
	// ran once and finished. Showing everything by default buries the handful
	// anyone actually cares about.
	const visible = $derived.by(() => {
		const needle = query.trim().toLowerCase();

		return units.filter((unit) => {
			if (runningOnly && unit.active !== 'active' && unit.active !== 'failed') return false;
			if (!needle) return true;
			return (
				unit.name.toLowerCase().includes(needle) ||
				unit.description.toLowerCase().includes(needle)
			);
		});
	});

	const failing = $derived(units.filter((unit) => unit.active === 'failed').length);

	$effect(() => {
		void load();
	});
</script>

<Seo title="Services" description="See what is running on your server and control it." noindex />

<DashboardTopbar
	crumbs={[
		{ label: 'Dashboard', href: '/dashboard' },
		{ label: 'Servers', href: '/dashboard/servers' },
		{ label: server.data?.name ?? 'Server', href: `/dashboard/servers/${serverId}` },
		{ label: 'Services' }
	]}
/>

<div class="space-y-5 px-6 py-8">
	{#if logsFor}
		<ServiceLogs {serverId} unit={logsFor} onClose={() => (logsFor = null)} />
	{:else}
		<div class="flex flex-wrap items-center justify-between gap-3">
			<div>
				<h1 class="text-content text-lg font-semibold tracking-tight">Services</h1>
				<p class="text-content-muted mt-0.5 text-sm">
					{#if loading}
						Reading what is on this server…
					{:else}
						{formatNumber(visible.length)} of {formatNumber(units.length)} shown{#if failing > 0}
							· <span class="text-error">{formatNumber(failing)} not working</span>
						{/if}
					{/if}
				</p>
			</div>

			<ButtonLink href="/dashboard/servers/{serverId}" variant="secondary" size="sm">
				<ArrowLeft size={15} aria-hidden="true" />
				Back to server
			</ButtonLink>
		</div>

		<div class="flex flex-wrap items-center gap-4">
			<SearchInput bind:value={query} placeholder="Search services" />

			<label class="text-content-muted flex items-center gap-2 text-sm">
				<input type="checkbox" bind:checked={runningOnly} />
				Hide things that are not running
			</label>
		</div>

		{#if actionError}
			<Card class="border-error/40 bg-error-soft">
				<p class="text-error text-sm">{actionError}</p>
			</Card>
		{/if}

		{#if error}
			<Card padded={false} class="overflow-hidden">
				<ErrorState {error} onRetry={load} />
			</Card>
		{:else if loading}
			<ServiceTableSkeleton />
		{:else}
			<ServiceTable
				units={visible}
				{busyUnit}
				onAction={act}
				onViewLogs={(unit) => (logsFor = unit.name)}
			/>
		{/if}
	{/if}
</div>
