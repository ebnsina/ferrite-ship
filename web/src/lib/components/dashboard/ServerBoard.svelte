<script lang="ts">
	import { ButtonLink, CountBadge, EmptyState } from '$components/ui';
	import { TONE_CLASSES } from '$components/ui/tone';
	import { groupServersByStatus } from '$lib/domain/fleet';
	import type { ManagedServer } from '$types/server';
	import ServerIcon from '@lucide/svelte/icons/server';
	import ServerCard from './ServerCard.svelte';

	interface Props {
		servers: readonly ManagedServer[];
	}

	let { servers }: Props = $props();

	const groups = $derived(groupServersByStatus(servers));
</script>

{#if servers.length === 0}
	<EmptyState
		icon={ServerIcon}
		title="No servers yet"
		description="Connect a machine and it will show up here within a few seconds."
	>
		{#snippet action()}
			<ButtonLink href="/dashboard/servers/new" size="sm">Connect your first server</ButtonLink>
		{/snippet}
	</EmptyState>
{:else}
	<!--
		Each status is a full-width section with its own grid, not a column.
		Grouping by column only reads well when statuses are evenly spread —
		with every server healthy it left one narrow stack and empty space.
	-->
	<div class="space-y-8">
		{#each groups as group (group.status)}
			<section class="space-y-3">
				<h3 class="flex items-center gap-2">
					<span
						class="size-2 rounded-pill {TONE_CLASSES[group.presentation.tone].fill}"
						aria-hidden="true"
					></span>
					<span class="text-content text-sm font-medium">{group.presentation.label}</span>
					<CountBadge count={group.servers.length} />
				</h3>

				<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
					{#each group.servers as server (server.id)}
						<ServerCard {server} />
					{/each}
				</div>
			</section>
		{/each}
	</div>
{/if}
