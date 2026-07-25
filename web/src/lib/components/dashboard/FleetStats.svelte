<script lang="ts">
	import { summarizeFleet } from '$lib/domain/fleet';
	import type { ManagedServer } from '$types/server';
	import { formatBytes, formatNumber, formatPercent } from '$utils/format';
	import Cpu from '@lucide/svelte/icons/cpu';
	import MemoryStick from '@lucide/svelte/icons/memory-stick';
	import Server from '@lucide/svelte/icons/server';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
	import StatCard from './StatCard.svelte';

	interface Props {
		servers: readonly ManagedServer[];
	}

	let { servers }: Props = $props();

	const summary = $derived(summarizeFleet(servers));
</script>

<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
	<StatCard
		label="Servers connected"
		value={formatNumber(summary.total)}
		hint="{formatNumber(summary.online)} running fine right now"
		icon={Server}
		tone="info"
	/>
	<StatCard
		label="Need a look"
		value={formatNumber(summary.needsAttention)}
		hint={summary.needsAttention === 0
			? 'Everything is healthy'
			: 'Working hard or not responding'}
		icon={TriangleAlert}
		tone={summary.needsAttention === 0 ? 'ok' : 'warn'}
	/>
	<StatCard
		label="How busy they are"
		value={formatPercent(summary.averageCpu)}
		hint="Average across servers we can reach"
		icon={Cpu}
		tone="pending"
	/>
	<StatCard
		label="Memory in use"
		value={formatBytes(summary.memoryUsedBytes)}
		hint="out of {formatBytes(summary.memoryTotalBytes)} you have"
		icon={MemoryStick}
		tone="pending"
	/>
</div>
