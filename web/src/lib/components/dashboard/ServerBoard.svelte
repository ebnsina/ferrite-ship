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
		description="Run the setup line on a new server and it will show up here within a few seconds."
	>
		{#snippet action()}
			<ButtonLink href="/dashboard/servers/new" size="sm">Connect your first server</ButtonLink>
		{/snippet}
	</EmptyState>
{:else}
	<div class="grid gap-x-5 gap-y-8 md:grid-cols-2 xl:grid-cols-4">
		{#each groups as group (group.status)}
			<section class="space-y-3">
				<div class="flex items-center gap-2">
					<span
						class="size-2 rounded-pill {TONE_CLASSES[group.presentation.tone].fill}"
						aria-hidden="true"
					></span>
					<h3 class="text-content text-sm font-medium">{group.presentation.label}</h3>
					<CountBadge count={group.servers.length} />
				</div>

				<div class="space-y-3">
					{#each group.servers as server (server.id)}
						<ServerCard {server} />
					{/each}
				</div>
			</section>
		{/each}
	</div>
{/if}
