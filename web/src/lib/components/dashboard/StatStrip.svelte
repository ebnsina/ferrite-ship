<script lang="ts">
	import type { FleetMetric } from '$types/metric';
	import StatStripItem from './StatStripItem.svelte';

	interface Props {
		metrics: readonly FleetMetric[];
	}

	let { metrics }: Props = $props();
</script>

<!--
	One bordered surface split by dividers rather than separate cards: the
	metrics are read together, so they should look like one object.
-->
<div
	class="border-border bg-surface divide-border/70 grid divide-y overflow-hidden rounded-panel border sm:grid-cols-2 sm:divide-x xl:grid-cols-4"
>
	{#each metrics as metric (metric.id)}
		<StatStripItem {metric} />
	{/each}
</div>
