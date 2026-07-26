<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import RemoveToolDialog from '$components/dashboard/RemoveToolDialog.svelte';
	import SectionHeader from '$components/dashboard/SectionHeader.svelte';
	import ToolCard from '$components/dashboard/ToolCard.svelte';
	import ToolsSkeleton from '$components/dashboard/ToolsSkeleton.svelte';
	import Seo from '$components/Seo.svelte';
	import { ButtonLink, Card, ErrorState } from '$components/ui';
	import { dashboardRepository } from '$lib/data';
	import { toolsClient, type Tool } from '$lib/data/tools';
	import { byCategory } from '$lib/domain/tools';
	import { toAppError, type AppError } from '$lib/errors';
	import { createResource } from '$utils/resource.svelte';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';

	const serverId = page.params.id ?? '';
	const server = createResource((signal) => dashboardRepository.getServer(serverId, signal));

	let tools = $state<Tool[]>([]);
	let loading = $state(true);
	let error = $state<AppError | null>(null);

	let actionError = $state<string | null>(null);
	let removing = $state<Tool | null>(null);
	let removeBusy = $state(false);

	async function load() {
		loading = true;
		error = null;

		try {
			tools = await toolsClient.list(serverId);
		} catch (cause) {
			error = toAppError(cause);
		} finally {
			loading = false;
		}
	}

	async function install(tool: Tool) {
		actionError = null;

		try {
			const job = await toolsClient.install(serverId, tool.id);
			// Straight to the log: installing pulls an image and can take
			// minutes, and watching it happen is far better than a spinner that
			// says nothing.
			await goto(`/dashboard/jobs/${job.id}`);
		} catch (cause) {
			actionError = toAppError(cause).message;
		}
	}

	async function remove(purge: boolean) {
		if (!removing) return;
		removeBusy = true;
		actionError = null;

		try {
			const job = await toolsClient.remove(serverId, removing.id, purge);
			await goto(`/dashboard/jobs/${job.id}`);
		} catch (cause) {
			actionError = toAppError(cause).message;
			removeBusy = false;
			removing = null;
		}
	}

	// A simulated server has no machine behind it and no public address, so
	// saying why up front beats letting someone press Install and be refused.
	function unavailableReason(tool: Tool): string | undefined {
		const kind = server.data?.connectionKind;
		if (kind !== 'demo') return undefined;
		if (tool.ports.some((port) => port.public)) {
			return 'This needs a server people can reach from the internet. Connect a real one to try it.';
		}
		return 'This is a simulated server, so there is nothing real to install on.';
	}

	const groups = $derived(byCategory(tools));
	const installedCount = $derived(tools.filter((tool) => tool.status === 'ready').length);

	$effect(() => {
		void load();
	});
</script>

<Seo
	title="Tools"
	description="Install databases and other software on your server."
	noindex
/>

<DashboardTopbar
	crumbs={[
		{ label: 'Dashboard', href: '/dashboard' },
		{ label: 'Servers', href: '/dashboard/servers' },
		{ label: server.data?.name ?? 'Server', href: `/dashboard/servers/${serverId}` },
		{ label: 'Tools' }
	]}
/>

<div class="space-y-6 px-6 py-8">
	<div class="flex flex-wrap items-start justify-between gap-3">
		<div>
			<h1 class="text-content text-lg font-semibold tracking-tight">Tools</h1>
			<p class="text-content-muted mt-0.5 text-sm">
				{#if loading}
					Checking what is on this server…
				{:else if installedCount === 0}
					Nothing installed yet. Pick something below and we will set it up for you.
				{:else}
					{installedCount === 1 ? '1 tool is' : `${installedCount} tools are`} running here.
				{/if}
			</p>
		</div>

		<ButtonLink href="/dashboard/servers/{serverId}" variant="secondary" size="sm">
			<ArrowLeft size={15} aria-hidden="true" />
			Back to server
		</ButtonLink>
	</div>

	{#if actionError}
		<Card size="sm" class="border-error/40 bg-error-soft">
			<p class="text-error text-sm">{actionError}</p>
		</Card>
	{/if}

	{#if error}
		<Card padded={false} class="overflow-hidden">
			<ErrorState {error} onRetry={load} />
		</Card>
	{:else if loading}
		<ToolsSkeleton />
	{:else}
		{#each groups as group (group.category)}
			<section class="space-y-4">
				<SectionHeader
					title={group.category}
					description={group.category === 'Databases'
						? 'Where your application keeps its information.'
						: group.category === 'Caching'
							? 'Temporary storage that makes things feel fast.'
							: 'Everything else you might want running.'}
				/>

				<div class="space-y-4">
					{#each group.tools as tool (tool.id)}
						<ToolCard
							{tool}
							{serverId}
							unavailable={unavailableReason(tool)}
							onInstall={install}
							onRemove={(chosen) => (removing = chosen)}
						/>
					{/each}
				</div>
			</section>
		{/each}
	{/if}
</div>

{#if removing}
	<RemoveToolDialog
		open={true}
		tool={removing}
		serverName={server.data?.name ?? 'this server'}
		busy={removeBusy}
		onConfirm={remove}
		onCancel={() => (removing = null)}
	/>
{/if}
