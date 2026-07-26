<script lang="ts">
	import { IconTile, StatusPill } from '$components/ui';
	import { TONE_CLASSES, toneForUsage } from '$components/ui/tone';
	import { SERVER_STATUS } from '$lib/domain/status';
	import { usageRatio, type ManagedServer } from '$types/server';
	import { formatBytes, formatDuration, formatRelativeTime } from '$utils/format';
	import Server from '@lucide/svelte/icons/server';

	interface Props {
		server: ManagedServer;
	}

	let { server }: Props = $props();

	const status = $derived(SERVER_STATUS[server.status]);
	const diskRatio = $derived(usageRatio(server.disk));
	const diskTone = $derived(toneForUsage(diskRatio));
	const isOffline = $derived(server.status === 'offline');
</script>

<article class="border-border bg-surface rounded-card border p-5">
	<div class="flex items-start justify-between gap-3">
		<IconTile icon={Server} tone={status.tone} size="sm" />
		<StatusPill {status} />
	</div>

	<h3 class="text-content mt-4 truncate text-sm font-medium">{server.name}</h3>
	<p class="text-content-subtle mt-1 truncate text-xs">
		{server.region} · {server.operatingSystem}
	</p>

	<dl class="mt-4 space-y-1 text-xs">
		<div class="flex justify-between gap-2">
			<dt class="text-content-subtle">{isOffline ? 'Last seen' : 'Running for'}</dt>
			<dd class="text-content-muted truncate">
				{isOffline ? formatRelativeTime(server.lastSeenAt) : formatDuration(server.uptimeMs)}
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

	<p class="border-border/70 text-content-subtle mt-4 border-t pt-3 text-xs">
		Checked {formatRelativeTime(server.lastSeenAt)}
	</p>
</article>
