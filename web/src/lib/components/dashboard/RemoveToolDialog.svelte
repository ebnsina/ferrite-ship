<script lang="ts">
	import { ConfirmDialog } from '$components/ui';
	import type { Tool } from '$lib/data/tools';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';

	interface Props {
		open: boolean;
		tool: Tool;
		serverName: string;
		busy?: boolean;
		onConfirm: (purge: boolean) => void;
		onCancel: () => void;
	}

	let {
		open = $bindable(),
		tool,
		serverName,
		busy = false,
		onConfirm,
		onCancel
	}: Props = $props();

	// Always starts off, and is reset every time the dialog opens. A checkbox
	// that remembers "yes, delete the data" from last time is how someone
	// deletes a database they meant to keep.
	let purge = $state(false);

	$effect(() => {
		if (open) purge = false;
	});

	// Offered only where there is something to delete. The API decides, so a
	// tool that stores nothing is never asked a frightening question with no
	// answer.
	const keepsData = $derived(tool.keepsData);
</script>

<ConfirmDialog
	bind:open
	title="Remove {tool.name} from {serverName}?"
	description="This stops it and deletes its settings. Anything connecting to it will stop working."
	confirmLabel={purge ? 'Remove it and delete the data' : 'Remove it'}
	danger
	{busy}
	onConfirm={() => onConfirm(purge)}
	{onCancel}
>
	<div class="space-y-4">
		<p class="text-content-muted text-xs leading-relaxed">{tool.dataNote}</p>

		{#if keepsData}
			<label
				class="border-border rounded-field flex items-start gap-3 border p-3
					{purge ? 'border-error/40 bg-error-soft' : ''}"
			>
				<input type="checkbox" bind:checked={purge} class="mt-0.5" />
				<span class="text-xs leading-relaxed">
					<span class="text-content font-medium">Delete the stored data too</span>
					<span class="text-content-muted block">
						Everything {tool.name} has saved on this server is deleted.
					</span>
				</span>
			</label>

			{#if purge}
				<p class="text-error flex items-start gap-2 text-xs leading-relaxed">
					<TriangleAlert size={14} aria-hidden="true" class="mt-0.5 shrink-0" />
					<span>
						There is no undo and no backup. If you are not certain, leave this unticked — the data
						will still be here if you install {tool.name} again.
					</span>
				</p>
			{/if}
		{/if}
	</div>
</ConfirmDialog>
