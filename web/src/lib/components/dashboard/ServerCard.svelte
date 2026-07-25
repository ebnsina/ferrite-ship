<script lang="ts">
	import { Card, Meter, StatusPill } from '$components/ui';
	import { SERVER_STATUS } from '$lib/domain/status';
	import { usageRatio, type ManagedServer } from '$types/server';
	import { formatBytes, formatDuration, formatRelativeTime } from '$utils/format';
	import Clock from '@lucide/svelte/icons/clock';
	import Cpu from '@lucide/svelte/icons/cpu';
	import HardDrive from '@lucide/svelte/icons/hard-drive';
	import MapPin from '@lucide/svelte/icons/map-pin';
	import MemoryStick from '@lucide/svelte/icons/memory-stick';

	interface Props {
		server: ManagedServer;
	}

	let { server }: Props = $props();

	const memoryRatio = $derived(usageRatio(server.memory));
	const diskRatio = $derived(usageRatio(server.disk));
	const isOffline = $derived(server.status === 'offline');
</script>

<Card class="flex h-full flex-col">
	<div class="flex items-start justify-between gap-3">
		<div class="min-w-0">
			<h3 class="text-content truncate text-sm font-medium">{server.name}</h3>
			<p class="text-content-subtle font-machine mt-1 truncate text-xs">{server.ipAddress}</p>
		</div>
		<StatusPill status={SERVER_STATUS[server.status]} />
	</div>

	<dl class="text-content-muted mt-5 space-y-2 text-xs">
		<div class="flex items-center gap-2">
			<dt class="flex items-center gap-1.5"><MapPin size={13} aria-hidden="true" /> Location</dt>
			<dd class="text-content ml-auto truncate">{server.region}</dd>
		</div>
		<div class="flex items-center gap-2">
			<dt class="flex items-center gap-1.5">
				<Clock size={13} aria-hidden="true" />
				{isOffline ? 'Last heard from' : 'Running for'}
			</dt>
			<dd class="text-content ml-auto truncate">
				{isOffline ? formatRelativeTime(server.lastSeenAt) : formatDuration(server.uptimeMs)}
			</dd>
		</div>
	</dl>

	<div class="mt-6 space-y-3.5">
		<Meter label="Processor" icon={Cpu} ratio={server.cpuUsage} />
		<Meter
			label="Memory"
			icon={MemoryStick}
			ratio={memoryRatio}
			detail="{formatBytes(server.memory.usedBytes)} of {formatBytes(server.memory.totalBytes)}"
		/>
		<Meter
			label="Storage"
			icon={HardDrive}
			ratio={diskRatio}
			detail="{formatBytes(server.disk.usedBytes)} of {formatBytes(server.disk.totalBytes)}"
		/>
	</div>

	{#if server.services.length > 0}
		<div class="mt-6">
			<p class="text-content-subtle text-xs">Running here</p>
			<ul class="mt-2 flex flex-wrap gap-1.5">
				{#each server.services as service (service)}
					<li
						class="border-border text-content-muted rounded-pill border px-2.5 py-1 text-[0.6875rem]"
					>
						{service}
					</li>
				{/each}
			</ul>
		</div>
	{/if}
</Card>
