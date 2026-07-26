<script lang="ts">
	import { Button, ButtonLink, Card, IconTile, StatusPill } from '$components/ui';
	import type { Tool } from '$lib/data/tools';
	import { isBusy, reachability, toolIcon, toolStatus } from '$lib/domain/tools';
	import Download from '@lucide/svelte/icons/download';
	import Globe from '@lucide/svelte/icons/globe';
	import Lock from '@lucide/svelte/icons/lock';
	import RotateCw from '@lucide/svelte/icons/rotate-cw';
	import SquareTerminal from '@lucide/svelte/icons/square-terminal';
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

	const Icon = $derived(toolIcon(tool.icon));
	const status = $derived(toolStatus(tool.status));
	const busy = $derived(isBusy(tool));
	const reach = $derived(reachability(tool));
	const installed = $derived(tool.status === 'ready');
</script>

<!--
	A tile rather than a wide row: the catalogue is browsed before it is used,
	and a grid of recognisable things is easier to scan than a stack of
	full-width bars. The whole card is not a link, because most of it is only
	interesting once the tool is installed.
-->
<Card class="flex h-full flex-col">
	<div class="flex items-start gap-3">
		<!-- The tool's own colour, not a status colour. The pill already says
		     "Running", with an icon and a word; tinting the logo green too would
		     say that twice in the one channel that cannot be read aloud. A brand
		     colour says something else — which tool this is — and is readable
		     before the name is. -->
		<IconTile icon={Icon} color={tool.accent} />

		<div class="min-w-0 flex-1">
			<div class="flex flex-wrap items-center gap-x-2 gap-y-1">
				<h3 class="text-content text-sm font-medium">{tool.name}</h3>
				<span class="text-content-subtle text-xs">{tool.version}</span>
			</div>
			<p class="text-content-subtle mt-1 flex items-center gap-1.5 text-xs">
				{#if tool.ports.some((port) => port.public)}
					<Globe size={13} aria-hidden="true" />
				{:else}
					<Lock size={13} aria-hidden="true" />
				{/if}
				{reach.label}
			</p>
		</div>

		{#if status}
			<StatusPill {status} compact />
		{/if}
	</div>

	<p class="text-content-muted mt-3 flex-1 text-sm leading-relaxed">{tool.summary}</p>

	{#if tool.status === 'failed'}
		<p class="text-error mt-3 text-xs leading-relaxed">
			Setting this up did not finish.
			{#if tool.lastJobId}
				<a href="/dashboard/jobs/{tool.lastJobId}" class="underline underline-offset-2">
					See what happened
				</a>
			{/if}
		</p>
	{:else if unavailable}
		<p class="text-content-subtle mt-3 text-xs leading-relaxed">{unavailable}</p>
	{/if}

	<div class="border-border/70 mt-4 flex flex-wrap items-center gap-2 border-t pt-4">
		{#if installed}
			<ButtonLink href="/dashboard/servers/{serverId}/tools/{tool.id}" size="sm">
				{#if tool.hasConsole}
					<SquareTerminal size={15} aria-hidden="true" />
					Open
				{:else}
					Connect
				{/if}
			</ButtonLink>
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
</Card>
