<script lang="ts">
	import { ConfirmDialog } from '$components/ui';
	import type { FileEntry } from '$lib/data/files';
	import { formatBytes } from '$utils/format';
	import FolderIcon from '@lucide/svelte/icons/folder';
	import FileIcon from '@lucide/svelte/icons/file';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';

	interface Props {
		entry: FileEntry | null;
		busy: boolean;
		onConfirm: () => void;
		onCancel: () => void;
	}

	let { entry, busy, onConfirm, onCancel }: Props = $props();

	// Deleting a file and deleting a folder fail in different ways, so they get
	// different warnings rather than one vague sentence covering both.
	const description = $derived(
		entry?.isDir
			? 'Only empty folders can be removed. If anything is still inside, nothing will be deleted and we will tell you.'
			: 'This removes the file from the server straight away. It does not go to a bin, nothing is backed up first, and it cannot be undone.'
	);
</script>

<ConfirmDialog
	open={entry !== null}
	title={entry?.isDir ? `Delete the folder ${entry.name}?` : `Delete ${entry?.name ?? ''}?`}
	{description}
	confirmLabel={entry?.isDir ? 'Delete folder' : 'Delete file'}
	danger
	{busy}
	{onConfirm}
	{onCancel}
>
	{#if entry}
		<div class="space-y-3">
			<div class="border-border bg-surface-sunken rounded-tile border p-3">
				<div class="text-content-muted flex items-center gap-2 text-xs">
					{#if entry.isDir}
						<FolderIcon size={13} aria-hidden="true" class="text-accent shrink-0" />
					{:else}
						<FileIcon size={13} aria-hidden="true" class="shrink-0" />
					{/if}
					<span class="font-machine text-content truncate">{entry.path}</span>
				</div>

				{#if !entry.isDir}
					<p class="text-content-subtle mt-2 text-xs" data-numeric>
						{formatBytes(entry.size)} · permissions {entry.mode}
					</p>
				{/if}
			</div>

			{#if entry.isSymlink}
				<p class="text-content-muted flex gap-2 text-xs leading-relaxed">
					<TriangleAlert size={13} aria-hidden="true" class="text-warn mt-0.5 shrink-0" />
					This is a shortcut to somewhere else. Removing it deletes the shortcut, not whatever it
					points at.
				</p>
			{/if}

			<p class="text-content-muted flex gap-2 text-xs leading-relaxed">
				<TriangleAlert size={13} aria-hidden="true" class="text-warn mt-0.5 shrink-0" />
				If something on this server is using it, that thing may stop working.
			</p>
		</div>
	{/if}
</ConfirmDialog>
