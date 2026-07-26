<script lang="ts">
	import { Meter, StatusPill } from '$components/ui';
	import { SERVER_STATUS } from '$lib/domain/status';
	import { usageRatio, type ManagedServer } from '$types/server';
	import { formatBytes, formatRelativeTime } from '$utils/format';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';

	interface Props {
		servers: readonly ManagedServer[];
	}

	let { servers }: Props = $props();
</script>

<!--
	A dense list rather than a grid of cards.

	Cards suit a handful of servers and stop working at fifty: each one is a
	block you read rather than a row you scan, and comparing "which of these is
	nearly full" means looking in a different place on every tile. A table puts
	the same figure in the same column, and one row per server means the page
	holds an order of magnitude more without scrolling for a minute.
-->
<div class="border-border rounded-panel overflow-hidden border">
	<!-- Scrolls inside its own box so a narrow window never widens the page. -->
	<div class="overflow-x-auto">
		<table class="w-full text-left text-sm">
			<thead class="border-border bg-surface-sunken text-content-muted border-b">
				<tr>
					<th class="px-5 py-3 font-medium">Server</th>
					<th class="px-5 py-3 font-medium">State</th>
					<th class="hidden px-5 py-3 font-medium lg:table-cell">Processor</th>
					<th class="hidden px-5 py-3 font-medium lg:table-cell">Memory</th>
					<th class="px-5 py-3 font-medium">Storage</th>
					<th class="hidden px-5 py-3 font-medium xl:table-cell">Last seen</th>
					<th class="px-5 py-3"><span class="sr-only">Open</span></th>
				</tr>
			</thead>

			<tbody class="divide-border/70 divide-y">
				{#each servers as server (server.id)}
					{@const status = SERVER_STATUS[server.status]}
					<tr class="hover:bg-surface-sunken/60 transition-colors duration-150">
						<td class="px-5 py-3">
							<a
								href="/dashboard/servers/{server.id}"
								class="text-content hover:text-accent block font-medium transition-colors duration-150"
							>
								{server.name}
							</a>
							<span class="text-content-subtle font-machine mt-0.5 block text-xs">
								{server.ipAddress}
							</span>
						</td>

						<td class="px-5 py-3"><StatusPill {status} /></td>

						<td class="hidden w-40 px-5 py-3 lg:table-cell">
							<Meter ratio={server.cpuUsage} compact />
						</td>

						<td class="hidden w-40 px-5 py-3 lg:table-cell">
							<Meter ratio={usageRatio(server.memory)} compact />
							<span class="text-content-subtle mt-1 block text-xs">
								{formatBytes(server.memory.usedBytes)} of {formatBytes(server.memory.totalBytes)}
							</span>
						</td>

						<td class="w-40 px-5 py-3">
							<Meter ratio={usageRatio(server.disk)} compact />
							<span class="text-content-subtle mt-1 block text-xs">
								{formatBytes(server.disk.usedBytes)} of {formatBytes(server.disk.totalBytes)}
							</span>
						</td>

						<td class="text-content-muted hidden px-5 py-3 text-xs whitespace-nowrap xl:table-cell">
							{formatRelativeTime(server.lastSeenAt, { style: 'short' })}
						</td>

						<td class="px-5 py-3 text-right">
							<a
								href="/dashboard/servers/{server.id}"
								class="text-content-subtle hover:text-content inline-flex transition-colors duration-150"
								aria-label="Open {server.name}"
							>
								<ChevronRight size={16} aria-hidden="true" />
							</a>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
</div>
