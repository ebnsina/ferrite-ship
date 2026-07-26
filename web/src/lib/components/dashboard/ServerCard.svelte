<script lang="ts">
	import { goto } from '$app/navigation';
	import { Button, ButtonLink, IconTile, StatusPill } from '$components/ui';
	import { TONE_CLASSES, toneForUsage } from '$components/ui/tone';
	import { SERVER_STATUS } from '$lib/domain/status';
	import { usageRatio, type ManagedServer } from '$types/server';
	import { dashboardCommands } from '$lib/data/commands';
	import { toAppError } from '$lib/errors';
	import { formatBytes, formatDuration, formatRelativeTime } from '$utils/format';
	import Play from '@lucide/svelte/icons/play';
	import Search from '@lucide/svelte/icons/search';
	import Server from '@lucide/svelte/icons/server';
	import SquareTerminal from '@lucide/svelte/icons/square-terminal';

	interface Props {
		server: ManagedServer;
	}

	let { server }: Props = $props();

	const status = $derived(SERVER_STATUS[server.status]);
	const diskRatio = $derived(usageRatio(server.disk));
	const diskTone = $derived(toneForUsage(diskRatio));
	const isOffline = $derived(server.status === 'offline');

	let starting = $state(false);
	let startError = $state<string | null>(null);

	async function start(dryRun: boolean) {
		if (starting) return;
		starting = true;
		startError = null;

		try {
			const job = await dashboardCommands.startBaseline(server.id, { dryRun });
			await goto(`/dashboard/jobs/${job.id}`);
		} catch (cause) {
			startError = toAppError(cause).message;
			starting = false;
		}
	}
</script>

<article class="border-border bg-surface rounded-card border p-5">
	<div class="flex items-start justify-between gap-3">
		<IconTile icon={Server} tone={status.tone} size="sm" />
		<StatusPill {status} />
	</div>

	<h3 class="mt-4 truncate text-sm font-medium">
		<a
			href="/dashboard/servers/{server.id}"
			class="text-content hover:text-accent transition-colors duration-150"
		>
			{server.name}
		</a>
	</h3>
	<p class="text-content-subtle mt-1 truncate text-xs">
		{server.region} · {server.operatingSystem}
	</p>

	<dl class="mt-4 space-y-1 text-xs">
		<div class="flex justify-between gap-2">
			<dt class="text-content-subtle">{isOffline ? 'Last seen' : 'Running for'}</dt>
			<dd class="text-content-muted truncate">
				{isOffline
				? formatRelativeTime(server.lastSeenAt, { style: 'short' })
				: formatDuration(server.uptimeMs)}
			</dd>
		</div>
		<div class="flex justify-between gap-2">
			<dt class="text-content-subtle">Storage</dt>
			<dd class="text-content-muted truncate" data-numeric>
				{formatBytes(server.disk.usedBytes)} of {formatBytes(server.disk.totalBytes)}
			</dd>
		</div>
	</dl>

	<div
		class="bg-surface-sunken mt-3 h-1.5 overflow-hidden rounded-pill"
		role="meter"
		aria-label="Storage used on {server.name}"
		aria-valuenow={Math.round(diskRatio * 100)}
		aria-valuemin={0}
		aria-valuemax={100}
	>
		<div
			class="h-full rounded-pill {TONE_CLASSES[diskTone].fill}"
			style:width="{diskRatio * 100}%"
		></div>
	</div>

	<div class="border-border/70 mt-4 flex items-center justify-between gap-3 border-t pt-3">
		<p class="text-content-subtle truncate text-xs">
			Checked {formatRelativeTime(server.lastSeenAt, { style: 'short' })}
		</p>
		<div class="flex shrink-0 items-center gap-2">
			{#if server.connectionKind === 'ssh'}
				<ButtonLink
					href="/dashboard/servers/{server.id}/terminal"
					size="sm"
					variant="ghost"
					aria-label="Open a terminal on {server.name}"
				>
					<SquareTerminal size={13} aria-hidden="true" />
				</ButtonLink>
			{/if}
			<Button size="sm" variant="ghost" onclick={() => start(true)} disabled={starting}>
				<Search size={13} aria-hidden="true" />
				Check
			</Button>
			<Button size="sm" variant="secondary" onclick={() => start(false)} disabled={starting}>
				<Play size={13} aria-hidden="true" />
				{starting ? 'Starting…' : 'Set up'}
			</Button>
		</div>
	</div>

	{#if startError}
		<p class="text-error mt-2 text-xs">{startError}</p>
	{/if}
</article>
