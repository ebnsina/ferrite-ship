<script lang="ts">
	import { Card } from '$components/ui';
	import type { DirectoryListing, FileEntry } from '$lib/data/files';
	import { filesClient } from '$lib/data/files';
	import { formatBytes, formatRelativeTime } from '$utils/format';
	import CornerLeftUp from '@lucide/svelte/icons/corner-left-up';
	import Download from '@lucide/svelte/icons/download';
	import FileIcon from '@lucide/svelte/icons/file';
	import Folder from '@lucide/svelte/icons/folder';
	import Link2 from '@lucide/svelte/icons/link-2';
	import Trash2 from '@lucide/svelte/icons/trash-2';

	interface Props {
		serverId: string;
		listing: DirectoryListing;
		onOpenDir: (path: string) => void;
		onOpenFile: (entry: FileEntry) => void;
		onDelete: (entry: FileEntry) => void;
	}

	let { serverId, listing, onOpenDir, onOpenFile, onDelete }: Props = $props();

	const atRoot = $derived(listing.path === '/');
</script>

<Card padded={false} class="overflow-hidden">
	<table class="w-full text-sm">
		<caption class="sr-only">Files in {listing.path}</caption>
		<thead class="border-border/70 text-content-subtle border-b text-xs">
			<tr>
				<th scope="col" class="px-5 py-2.5 text-left font-medium">Name</th>
				<th scope="col" class="hidden px-3 py-2.5 text-right font-medium sm:table-cell">Size</th>
				<th scope="col" class="hidden px-3 py-2.5 text-right font-medium md:table-cell">Changed</th>
				<th scope="col" class="hidden px-3 py-2.5 text-right font-medium lg:table-cell">
					Permissions
				</th>
				<th scope="col" class="px-5 py-2.5 text-right font-medium">
					<span class="sr-only">Actions</span>
				</th>
			</tr>
		</thead>

		<tbody class="divide-border/70 divide-y">
			{#if !atRoot}
				<tr class="hover:bg-surface-sunken transition-colors duration-150">
					<td colspan="5" class="px-5 py-2.5">
						<button
							type="button"
							onclick={() => onOpenDir(listing.parent)}
							class="text-content-muted hover:text-content flex items-center gap-2.5"
						>
							<CornerLeftUp size={15} aria-hidden="true" />
							Up one level
						</button>
					</td>
				</tr>
			{/if}

			{#each listing.entries as entry (entry.path)}
				<tr class="hover:bg-surface-sunken group transition-colors duration-150">
					<td class="max-w-0 px-5 py-2.5">
						<button
							type="button"
							onclick={() => (entry.isDir ? onOpenDir(entry.path) : onOpenFile(entry))}
							class="flex w-full items-center gap-2.5 text-left"
						>
							{#if entry.isDir}
								<Folder size={15} aria-hidden="true" class="text-accent shrink-0" />
							{:else if entry.isSymlink}
								<Link2 size={15} aria-hidden="true" class="text-content-subtle shrink-0" />
							{:else}
								<FileIcon size={15} aria-hidden="true" class="text-content-subtle shrink-0" />
							{/if}
							<span class="text-content font-machine truncate text-xs">{entry.name}</span>
						</button>
					</td>

					<td class="text-content-muted hidden px-3 py-2.5 text-right text-xs sm:table-cell" data-numeric>
						{entry.isDir ? '—' : formatBytes(entry.size)}
					</td>

					<td class="text-content-subtle hidden px-3 py-2.5 text-right text-xs md:table-cell">
						{formatRelativeTime(entry.modifiedAt, { style: 'short' })}
					</td>

					<td class="text-content-subtle font-machine hidden px-3 py-2.5 text-right text-xs lg:table-cell">
						{entry.mode}
					</td>

					<td class="px-5 py-2.5 text-right">
						<!-- Row actions stay hidden until hover or focus, so the list reads
						     as a list rather than a wall of buttons. -->
						<div class="flex items-center justify-end gap-1 opacity-0 transition-opacity duration-150 group-hover:opacity-100 focus-within:opacity-100">
							{#if !entry.isDir}
								<a
									href={filesClient.downloadUrl(serverId, entry.path)}
									class="text-content-muted hover:bg-surface-raised hover:text-content inline-flex size-7 items-center justify-center rounded-tile transition-colors duration-150"
									aria-label="Download {entry.name}"
								>
									<Download size={14} aria-hidden="true" />
								</a>
							{/if}
							<button
								type="button"
								onclick={() => onDelete(entry)}
								class="text-content-muted hover:bg-error-soft hover:text-error inline-flex size-7 items-center justify-center rounded-tile transition-colors duration-150"
								aria-label="Delete {entry.name}"
							>
								<Trash2 size={14} aria-hidden="true" />
							</button>
						</div>
					</td>
				</tr>
			{/each}

			{#if listing.entries.length === 0}
				<tr>
					<td colspan="5" class="text-content-subtle px-5 py-10 text-center text-sm">
						This folder is empty.
					</td>
				</tr>
			{/if}
		</tbody>
	</table>
</Card>
