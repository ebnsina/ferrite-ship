<script lang="ts">
	import ToolConnection from '$components/dashboard/ToolConnection.svelte';
	import { Button, ButtonLink, Card, IconTile, StatusPill } from '$components/ui';
	import type { Tool } from '$lib/data/tools';
	import { isBusy, reachability, toolIcon, toolStatus } from '$lib/domain/tools';
	import ChevronDown from '@lucide/svelte/icons/chevron-down';
	import Download from '@lucide/svelte/icons/download';
	import Globe from '@lucide/svelte/icons/globe';
	import Lock from '@lucide/svelte/icons/lock';
	import RotateCw from '@lucide/svelte/icons/rotate-cw';
	import Trash2 from '@lucide/svelte/icons/trash-2';

	interface Props {
		tool: Tool;
		serverId: string;
		/** Set when this server cannot host the tool, with the reason. */
		unavailable?: string;
		onInstall: (tool: Tool) => void;
		onRemove: (tool: Tool) => void;
	}

	let { tool, serverId, unavailable, onInstall, onRemove }: Props = $props();

	let showDetails = $state(false);

	const Icon = $derived(toolIcon(tool.icon));
	const status = $derived(toolStatus(tool.status));
	const busy = $derived(isBusy(tool));
	const reach = $derived(reachability(tool));
	const installed = $derived(tool.status === 'ready');

	// Details are only meaningful once it is actually running — there is no
	// password to show for something that has not been installed.
	$effect(() => {
		if (!installed) showDetails = false;
	});
</script>

<Card>
	<div class="flex items-start gap-4">
		<IconTile icon={Icon} tone={installed ? 'ok' : 'neutral'} />

		<div class="min-w-0 flex-1">
			<div class="flex flex-wrap items-center gap-x-2 gap-y-1">
				<h3 class="text-content text-sm font-medium">{tool.name}</h3>
				<span class="text-content-subtle text-xs">{tool.version}</span>
				{#if status}
					<StatusPill {status} />
				{/if}
			</div>

			<p class="text-content-muted mt-1.5 text-sm leading-relaxed">{tool.summary}</p>

			<p class="text-content-subtle mt-2 flex items-center gap-1.5 text-xs">
				{#if tool.ports.some((port) => port.public)}
					<Globe size={13} aria-hidden="true" />
				{:else}
					<Lock size={13} aria-hidden="true" />
				{/if}
				{reach.label}
			</p>

			{#if tool.status === 'failed'}
				<p class="text-error mt-2 text-xs leading-relaxed">
					Setting this up did not finish.
					{#if tool.lastJobId}
						<ButtonLink href="/dashboard/jobs/{tool.lastJobId}" variant="ghost" size="sm">
							See what happened
						</ButtonLink>
					{/if}
				</p>
			{:else if unavailable}
				<p class="text-content-subtle mt-2 text-xs leading-relaxed">{unavailable}</p>
			{/if}
		</div>

		<div class="flex shrink-0 flex-wrap items-center justify-end gap-2">
			{#if installed}
				<Button variant="ghost" size="sm" onclick={() => (showDetails = !showDetails)}>
					<ChevronDown
						size={15}
						aria-hidden="true"
						class="transition-transform duration-150 {showDetails ? 'rotate-180' : ''}"
					/>
					{showDetails ? 'Hide details' : 'Connect'}
				</Button>
				<Button
					variant="ghost"
					size="sm"
					onclick={() => onInstall(tool)}
					disabled={busy}
					title="Checks everything is as it should be and repairs anything that has drifted."
				>
					<RotateCw size={15} aria-hidden="true" />
					Repair
				</Button>
				<Button variant="ghost" size="sm" onclick={() => onRemove(tool)} disabled={busy}>
					<Trash2 size={15} aria-hidden="true" />
					Remove
				</Button>
			{:else}
				<Button
					variant="secondary"
					size="sm"
					onclick={() => onInstall(tool)}
					disabled={busy || Boolean(unavailable)}
				>
					<Download size={15} aria-hidden="true" />
					{#if busy}
						Working…
					{:else if tool.status === 'failed'}
						Try again
					{:else}
						Install
					{/if}
				</Button>
			{/if}
		</div>
	</div>

	{#if showDetails && installed}
		<div class="mt-5">
			<ToolConnection {serverId} toolId={tool.id} />
		</div>
	{/if}
</Card>
