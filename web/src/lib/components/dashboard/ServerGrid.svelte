<script lang="ts">
	import { ButtonLink, EmptyState } from '$components/ui';
	import type { ManagedServer } from '$types/server';
	import ServerIcon from '@lucide/svelte/icons/server';
	import ServerCard from './ServerCard.svelte';

	interface Props {
		servers: readonly ManagedServer[];
	}

	let { servers }: Props = $props();
</script>

{#if servers.length === 0}
	<EmptyState
		icon={ServerIcon}
		title="No servers yet"
		description="Run the setup line on a new server and it will show up here within a few seconds."
	>
		{#snippet action()}
			<ButtonLink href="/dashboard/servers" size="sm">Connect your first server</ButtonLink>
		{/snippet}
	</EmptyState>
{:else}
	<div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
		{#each servers as server (server.id)}
			<ServerCard {server} />
		{/each}
	</div>
{/if}
