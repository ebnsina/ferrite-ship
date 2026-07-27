<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import ActivityFeed from '$components/dashboard/ActivityFeed.svelte';
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import ResourceView from '$components/dashboard/ResourceView.svelte';
	import SectionHeader from '$components/dashboard/SectionHeader.svelte';
	import Seo from '$components/Seo.svelte';
	import {
		ButtonLink,
		Card,
		ConfirmDialog,
		IconTile,
		Menu,
		Meter,
		Skeleton,
		StatusPill,
		type MenuItem
	} from '$components/ui';
	import { dashboardCommands } from '$lib/data/commands';
	import { dashboardRepository } from '$lib/data';
	import { SERVER_STATUS } from '$lib/domain/status';
	import { toAppError } from '$lib/errors';
	import { usageRatio } from '$types/server';
	import { formatBytes, formatDuration, formatRelativeTime } from '$utils/format';
	import { createResource } from '$utils/resource.svelte';
	import Boxes from '@lucide/svelte/icons/boxes';
	import Cog from '@lucide/svelte/icons/cog';
	import Cpu from '@lucide/svelte/icons/cpu';
	import FolderOpen from '@lucide/svelte/icons/folder-open';
	import HardDrive from '@lucide/svelte/icons/hard-drive';
	import MemoryStick from '@lucide/svelte/icons/memory-stick';
	import Package from '@lucide/svelte/icons/package';
	import Play from '@lucide/svelte/icons/play';
	import RotateCw from '@lucide/svelte/icons/rotate-cw';
	import Search from '@lucide/svelte/icons/search';
	import Server from '@lucide/svelte/icons/server';
	import SquareTerminal from '@lucide/svelte/icons/square-terminal';
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

	// Built from the server's own state: telling someone to "set up" a machine
	// they set up last week is meaningless, so the label follows the facts.
	const actions = $derived.by((): MenuItem[] => {
		const isSetUp = Boolean(server.data?.setUpAt);

		return [
			{
				label: isSetUp ? 'Re-run setup' : 'Set up',
				icon: isSetUp ? RotateCw : Play,
				disabled: starting,
				onSelect: () => void start(false)
			},
			{
				label: 'Check without changing anything',
				icon: Search,
				disabled: starting,
				onSelect: () => void start(true)
			},
			{
				label: 'Remove from Ferrite Ship',
				icon: Trash2,
				danger: true,
				separated: true,
				onSelect: () => (confirmOpen = true)
			}
		];
	});
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
			<div class="space-y-8">
				<div class="flex items-start gap-4">
					<Skeleton class="size-10 shrink-0" />
					<div class="space-y-2">
						<Skeleton class="h-6 w-40" />
						<Skeleton class="h-3 w-56" />
						<Skeleton class="h-3 w-28" />
					</div>
				</div>
				<div class="grid gap-4 lg:grid-cols-3">
					<Skeleton shape="card" class="h-56 lg:col-span-2" />
					<Skeleton shape="card" class="h-56" />
				</div>
			</div>
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

				<!--
					Places on the left, things you do on the right.

					These used to be seven controls in one row, which read as a
					wall rather than a set of choices — and it put "Remove" the
					same distance from the cursor as "Files". Navigation stays
					visible because it is where you are going next; everything
					that changes the machine is one deliberate click away.
				-->
				<div class="flex flex-wrap items-center gap-2">
					<StatusPill {status} />

					{#if s.connectionKind === 'ssh'}
						<ButtonLink href="/dashboard/servers/{id}/apps" variant="ghost" size="sm">
							<Boxes size={15} aria-hidden="true" />
							Apps
						</ButtonLink>
						<ButtonLink href="/dashboard/servers/{id}/tools" variant="ghost" size="sm">
							<Package size={15} aria-hidden="true" />
							Tools
						</ButtonLink>
						<ButtonLink href="/dashboard/servers/{id}/services" variant="ghost" size="sm">
							<Cog size={15} aria-hidden="true" />
							Services
						</ButtonLink>
						<ButtonLink href="/dashboard/servers/{id}/files" variant="ghost" size="sm">
							<FolderOpen size={15} aria-hidden="true" />
							Files
						</ButtonLink>
						<ButtonLink href="/dashboard/servers/{id}/terminal" variant="ghost" size="sm">
							<SquareTerminal size={15} aria-hidden="true" />
							Terminal
						</ButtonLink>
					{/if}

					<Menu label={starting ? 'Starting…' : 'Actions'} items={actions} />
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
						<!-- A bar going red tells you there is a problem and nothing
						     about what to do, which leaves the terminal as the only
						     way forward. This is the way out, and it sits where the
						     problem is noticed. -->
						<a
							href="/dashboard/servers/{id}/storage"
							class="text-content-muted hover:text-content inline-flex items-center gap-1.5 text-xs transition-colors duration-150"
						>
							<Search size={13} aria-hidden="true" />
							See what is using it
						</a>
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
							<dt class="text-content-muted">Set up</dt>
							<dd class="text-content truncate">
								{s.setUpAt ? formatRelativeTime(s.setUpAt, { style: 'short' }) : 'Not yet'}
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
