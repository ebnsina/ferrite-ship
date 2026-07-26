<script lang="ts">
	import { page } from '$app/state';
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import ResourceView from '$components/dashboard/ResourceView.svelte';
	import ServerTerminal from '$components/dashboard/ServerTerminal.svelte';
	import Seo from '$components/Seo.svelte';
	import { ButtonLink, Card, EmptyState, Skeleton } from '$components/ui';
	import { TONE_CLASSES, type Tone } from '$components/ui/tone';
	import { dashboardRepository } from '$lib/data';
	import type { TerminalStatus } from '$types/terminal';
	import { createResource } from '$utils/resource.svelte';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import FlaskConical from '@lucide/svelte/icons/flask-conical';

	const id = page.params.id ?? '';
	const server = createResource((signal) => dashboardRepository.getServer(id, signal));

	let status = $state<TerminalStatus>('connecting');

	const STATUS: Record<TerminalStatus, { label: string; tone: Tone }> = {
		connecting: { label: 'Connecting…', tone: 'info' },
		connected: { label: 'Connected', tone: 'ok' },
		closed: { label: 'Disconnected', tone: 'pending' },
		error: { label: 'Could not connect', tone: 'error' }
	};
</script>

<Seo title="Terminal" description="A shell on your server, in the browser." noindex />

<DashboardTopbar
	crumbs={[
		{ label: 'Dashboard', href: '/dashboard' },
		{ label: 'Servers', href: '/dashboard/servers' },
		{ label: server.data?.name ?? 'Server', href: `/dashboard/servers/${id}` },
		{ label: 'Terminal' }
	]}
/>

<div class="space-y-5 px-6 py-8">
	<ResourceView resource={server}>
		{#snippet pending()}
			<Skeleton shape="card" class="h-[60vh]" />
		{/snippet}

		{#snippet children(s)}
			{#if s.connectionKind === 'demo'}
				<Card size="lg" padded={false} class="overflow-hidden">
					<EmptyState
						icon={FlaskConical}
						title="There is no shell here"
						description="This is a simulated server, so there is no real machine to open a terminal on. Connect a real server to use this."
					>
						{#snippet action()}
							<ButtonLink href="/dashboard/servers/new" size="sm">Connect a real server</ButtonLink>
						{/snippet}
					</EmptyState>
				</Card>
			{:else}
				<div class="flex flex-wrap items-center justify-between gap-3">
					<div>
						<h1 class="text-content text-lg font-semibold tracking-tight">{s.name}</h1>
						<p class="text-content-muted mt-0.5 text-sm">
							You are signed in as <span class="font-machine">{s.hostname || s.ipAddress}</span>.
							Anything you type here runs on the real machine.
						</p>
					</div>

					<span
						class="rounded-pill px-2.5 py-1 text-xs font-medium {TONE_CLASSES[STATUS[status].tone]
							.soft}"
					>
						{STATUS[status].label}
					</span>
				</div>

				<ServerTerminal serverId={id} onStatusChange={(next) => (status = next)} />

				<div class="flex flex-wrap items-center justify-between gap-3">
					<p class="text-content-subtle text-xs">
						Closing this page ends the session. Nothing you type is recorded yet.
					</p>
					<ButtonLink href="/dashboard/servers/{id}" variant="secondary" size="sm">
						<ArrowLeft size={15} aria-hidden="true" />
						Back to server
					</ButtonLink>
				</div>
			{/if}
		{/snippet}
	</ResourceView>
</div>
