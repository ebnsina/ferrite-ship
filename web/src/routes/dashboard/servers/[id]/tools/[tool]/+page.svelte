<script lang="ts">
	import { page } from '$app/state';
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import ToolConnection from '$components/dashboard/ToolConnection.svelte';
	import ToolBackups from '$components/dashboard/ToolBackups.svelte';
	import ToolConsole from '$components/dashboard/ToolConsole.svelte';
	import Seo from '$components/Seo.svelte';
	import { ButtonLink, Card, ErrorState, IconTile, Skeleton, StatusPill } from '$components/ui';
	import { dashboardRepository } from '$lib/data';
	import { toolsClient, type Tool } from '$lib/data/tools';
	import { reachability, toolIcon, toolStatus } from '$lib/domain/tools';
	import { toAppError, type AppError } from '$lib/errors';
	import { createResource } from '$utils/resource.svelte';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import Globe from '@lucide/svelte/icons/globe';
	import Lock from '@lucide/svelte/icons/lock';

	const serverId = page.params.id ?? '';
	const toolId = page.params.tool ?? '';

	const server = createResource((signal) => dashboardRepository.getServer(serverId, signal));

	let tool = $state<Tool | null>(null);
	let loading = $state(true);
	let error = $state<AppError | null>(null);

	async function load() {
		loading = true;
		error = null;

		try {
			// The list endpoint carries both the catalogue entry and what this
			// server has done with it, which is exactly what this page needs;
			// asking for one tool would be a second shape to keep in step.
			const tools = await toolsClient.list(serverId);
			tool = tools.find((candidate) => candidate.id === toolId) ?? null;
		} catch (cause) {
			error = toAppError(cause);
		} finally {
			loading = false;
		}
	}

	const status = $derived(tool ? toolStatus(tool.status) : null);
	const ready = $derived(tool?.status === 'ready');

	$effect(() => {
		void toolId;
		void load();
	});
</script>

<Seo title={tool?.name ?? 'Tool'} description="Connect to and query this tool." noindex />

<DashboardTopbar
	crumbs={[
		{ label: 'Dashboard', href: '/dashboard' },
		{ label: 'Servers', href: '/dashboard/servers' },
		{ label: server.data?.name ?? 'Server', href: `/dashboard/servers/${serverId}` },
		{ label: 'Tools', href: `/dashboard/servers/${serverId}/tools` },
		{ label: tool?.name ?? 'Tool' }
	]}
/>

<div class="space-y-6 px-6 py-8">
	{#if loading}
		<div class="flex items-start gap-4">
			<Skeleton class="size-10 shrink-0" />
			<div class="space-y-2">
				<Skeleton class="h-5 w-40" />
				<Skeleton class="h-3 w-64" />
			</div>
		</div>
		<Skeleton shape="card" class="h-64" />
	{:else if error}
		<Card size="lg" padded={false} class="overflow-hidden">
			<ErrorState {error} onRetry={load} />
		</Card>
	{:else if !tool}
		<Card size="lg">
			<p class="text-content text-sm">We do not know a tool by that name.</p>
			<p class="text-content-muted mt-1 text-sm">It may have been renamed or removed.</p>
		</Card>
	{:else}
		{@const Icon = toolIcon(tool.icon)}
		{@const isPublic = tool.ports.some((port) => port.public)}

		<div class="flex flex-wrap items-start justify-between gap-4">
			<div class="flex items-start gap-4">
				<IconTile icon={Icon} color={tool.accent} />
				<div class="min-w-0">
					<div class="flex flex-wrap items-center gap-x-2 gap-y-1">
						<h1 class="text-content text-xl font-semibold tracking-tight">{tool.name}</h1>
						<span class="text-content-subtle text-xs">{tool.version}</span>
						{#if status}
							<StatusPill {status} />
						{/if}
					</div>
					<p class="text-content-muted mt-1 max-w-xl text-sm leading-relaxed">{tool.summary}</p>
					<p class="text-content-subtle mt-2 flex items-center gap-1.5 text-xs">
						{#if isPublic}
							<Globe size={13} aria-hidden="true" />
						{:else}
							<Lock size={13} aria-hidden="true" />
						{/if}
						{reachability(tool, server.data?.domain).detail}
					</p>
				</div>
			</div>

			<ButtonLink href="/dashboard/servers/{serverId}/tools" variant="secondary" size="sm">
				<ArrowLeft size={15} aria-hidden="true" />
				All tools
			</ButtonLink>
		</div>

		{#if !ready}
			<Card size="lg">
				<p class="text-content text-sm">
					{#if tool.status === 'failed'}
						Setting this up did not finish, so there is nothing to connect to yet.
					{:else if tool.status}
						This is still being worked on. The details appear once it is running.
					{:else}
						This is not installed on this server yet.
					{/if}
				</p>
				<div class="mt-4">
					<ButtonLink href="/dashboard/servers/{serverId}/tools" size="sm">
						Go and install it
					</ButtonLink>
				</div>
			</Card>
		{:else}
			{#if tool.hasConsole}
				<Card size="lg">
					<ToolConsole {serverId} {tool} />
				</Card>
			{/if}

			<section class="space-y-3">
				<h2 class="text-content text-sm font-medium">Connect from your own machine</h2>
				<ToolConnection {serverId} toolId={tool.id} />
			</section>

			{#if tool.keepsData}
				<Card size="lg">
					<ToolBackups {serverId} {tool} />
				</Card>
			{/if}
		{/if}
	{/if}
</div>
