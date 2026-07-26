<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import ActivityFeed from '$components/dashboard/ActivityFeed.svelte';
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import ResourceView from '$components/dashboard/ResourceView.svelte';
	import SectionHeader from '$components/dashboard/SectionHeader.svelte';
	import Seo from '$components/Seo.svelte';
	import {
		Button,
		Card,
		ConfirmDialog,
		IconTile,
		Meter,
		Skeleton,
		StatusPill
	} from '$components/ui';
	import { dashboardCommands } from '$lib/data/commands';
	import { dashboardRepository } from '$lib/data';
	import { SERVER_STATUS } from '$lib/domain/status';
	import { toAppError } from '$lib/errors';
	import { usageRatio } from '$types/server';
	import { formatBytes, formatDuration, formatRelativeTime } from '$utils/format';
	import { createResource } from '$utils/resource.svelte';
	import Cpu from '@lucide/svelte/icons/cpu';
	import HardDrive from '@lucide/svelte/icons/hard-drive';
	import MemoryStick from '@lucide/svelte/icons/memory-stick';
	import Play from '@lucide/svelte/icons/play';
	import Search from '@lucide/svelte/icons/search';
	import Server from '@lucide/svelte/icons/server';
	import Trash2 from '@lucide/svelte/icons/trash-2';

	const id = page.params.id ?? '';

	const server = createResource((signal) => dashboardRepository.getServer(id, signal));
	const jobs = createResource((signal) => dashboardRepository.listServerJobs(id, signal));

	let starting = $state(false);
	let removing = $state(false);
	let confirmOpen = $state(false);
	let actionError = $state<string | null>(null);

	async function start(dryRun: boolean) {
		if (starting) return;
		starting = true;
		actionError = null;

		try {
			const job = await dashboardCommands.startBaseline(id, { dryRun });
			await goto(`/dashboard/jobs/${job.id}`);
		} catch (cause) {
			actionError = toAppError(cause).message;
			starting = false;
		}
	}

	async function remove() {
		if (removing) return;
		removing = true;
		actionError = null;

		try {
			await dashboardCommands.removeServer(id);
			await goto('/dashboard/servers');
		} catch (cause) {
			actionError = toAppError(cause).message;
			removing = false;
			confirmOpen = false;
		}
	}
</script>

<Seo title="Server" description="Everything about one connected server." noindex />

<DashboardTopbar
	crumbs={[
		{ label: 'Dashboard', href: '/dashboard' },
		{ label: 'Servers', href: '/dashboard/servers' },
		{ label: server.data?.name ?? 'Server' }
	]}
/>

<div class="space-y-8 px-6 py-8">
	<ResourceView resource={server}>
		{#snippet pending()}
			<Skeleton shape="card" class="h-44" />
		{/snippet}

		{#snippet children(s)}
			{@const status = SERVER_STATUS[s.status]}
			{@const isOffline = s.status === 'offline'}

			<div class="flex flex-wrap items-start justify-between gap-4">
				<div class="flex items-start gap-4">
					<IconTile icon={Server} tone={status.tone} />
					<div class="min-w-0">
						<h1 class="text-content text-xl font-semibold tracking-tight">{s.name}</h1>
						<p class="text-content-muted mt-1 text-sm">
							{s.region} · {s.operatingSystem || 'Not checked yet'}
						</p>
						<p class="text-content-subtle font-machine mt-1 text-xs">{s.ipAddress}</p>
					</div>
				</div>

				<div class="flex flex-wrap items-center gap-2">
					<StatusPill {status} />
					<Button variant="ghost" size="sm" onclick={() => start(true)} disabled={starting}>
						<Search size={15} aria-hidden="true" />
						Check
					</Button>
					<Button variant="secondary" size="sm" onclick={() => start(false)} disabled={starting}>
						<Play size={15} aria-hidden="true" />
						{starting ? 'Starting…' : 'Set up'}
					</Button>
					<Button variant="ghost" size="sm" onclick={() => (confirmOpen = true)}>
						<Trash2 size={15} aria-hidden="true" />
						Remove
					</Button>
				</div>
			</div>

			{#if actionError}
				<p class="text-error text-sm">{actionError}</p>
			{/if}

			<div class="grid gap-4 lg:grid-cols-3">
				<Card class="lg:col-span-2">
					<h2 class="text-content text-sm font-medium">How hard it is working</h2>
					<div class="mt-5 space-y-4">
						<Meter label="Processor" icon={Cpu} ratio={s.cpuUsage} />
						<Meter
							label="Memory"
							icon={MemoryStick}
							ratio={usageRatio(s.memory)}
							detail="{formatBytes(s.memory.usedBytes)} of {formatBytes(s.memory.totalBytes)}"
						/>
						<Meter
							label="Storage"
							icon={HardDrive}
							ratio={usageRatio(s.disk)}
							detail="{formatBytes(s.disk.usedBytes)} of {formatBytes(s.disk.totalBytes)}"
						/>
					</div>
				</Card>

				<Card>
					<h2 class="text-content text-sm font-medium">Details</h2>
					<dl class="mt-5 space-y-3 text-sm">
						<div class="flex justify-between gap-3">
							<dt class="text-content-muted">Address</dt>
							<dd class="text-content font-machine truncate">{s.ipAddress}</dd>
						</div>
						<div class="flex justify-between gap-3">
							<dt class="text-content-muted">Name on the machine</dt>
							<dd class="text-content truncate">{s.hostname || '—'}</dd>
						</div>
						<div class="flex justify-between gap-3">
							<dt class="text-content-muted">{isOffline ? 'Last seen' : 'Running for'}</dt>
							<dd class="text-content truncate">
								{isOffline
									? formatRelativeTime(s.lastSeenAt, { style: 'short' })
									: formatDuration(s.uptimeMs)}
							</dd>
						</div>
						<div class="flex justify-between gap-3">
							<dt class="text-content-muted">Connected by</dt>
							<dd class="text-content truncate">
								{s.connectionKind === 'demo' ? 'Simulated' : 'SSH'}
							</dd>
						</div>
					</dl>
				</Card>
			</div>

			<ConfirmDialog
				bind:open={confirmOpen}
				title="Remove {s.name}?"
				description="This forgets the server here. The machine itself keeps running, and anything already set up on it stays exactly as it is."
				confirmLabel="Remove it"
				danger
				busy={removing}
				onConfirm={remove}
				onCancel={() => (confirmOpen = false)}
			>
				<p class="text-content-muted text-xs leading-relaxed">
					Its history and stored sign-in details are deleted too. To manage it again you would
					connect it afresh.
				</p>
			</ConfirmDialog>
		{/snippet}
	</ResourceView>

	<section class="space-y-4">
		<SectionHeader title="What has run here" description="Setup runs and checks for this server." />

		<ResourceView resource={jobs}>
			{#snippet pending()}
				<Skeleton shape="card" class="h-56" />
			{/snippet}

			{#snippet children(entries)}
				<ActivityFeed {entries} />
			{/snippet}
		</ResourceView>
	</section>
</div>
