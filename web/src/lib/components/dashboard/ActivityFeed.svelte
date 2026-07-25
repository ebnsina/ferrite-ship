<script lang="ts">
	import { Card, EmptyState } from '$components/ui';
	import type { ActivityEntry } from '$types/activity';
	import History from '@lucide/svelte/icons/history';
	import ActivityRow from './ActivityRow.svelte';

	interface Props {
		entries: readonly ActivityEntry[];
	}

	let { entries }: Props = $props();
</script>

<Card padded={false}>
	{#if entries.length === 0}
		<EmptyState
			icon={History}
			title="Nothing has happened yet"
			description="Anything you start, and anything that runs on a schedule, will be listed here with how it went."
		/>
	{:else}
		<ul class="divide-border/70 divide-y px-6 py-1">
			{#each entries as entry (entry.id)}
				<ActivityRow {entry} />
			{/each}
		</ul>
	{/if}
</Card>
