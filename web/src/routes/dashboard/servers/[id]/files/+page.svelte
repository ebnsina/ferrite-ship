<script lang="ts">
	import { page } from '$app/state';
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import FileBrowser from '$components/dashboard/FileBrowser.svelte';
	import FileEditor from '$components/dashboard/FileEditor.svelte';
	import Seo from '$components/Seo.svelte';
	import { ButtonLink, Card, ConfirmDialog, ErrorState, Skeleton } from '$components/ui';
	import {
		filesClient,
		type DirectoryListing,
		type FileContent,
		type FileEntry
	} from '$lib/data/files';
	import { toAppError, type AppError } from '$lib/errors';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';

	const serverId = page.params.id ?? '';

	let path = $state('/');
	let listing = $state<DirectoryListing | null>(null);
	let openFile = $state<FileContent | null>(null);

	let loading = $state(true);
	let error = $state<AppError | null>(null);

	let pendingDelete = $state<FileEntry | null>(null);
	let deleting = $state(false);

	async function load(next: string) {
		loading = true;
		error = null;

		try {
			listing = await filesClient.list(serverId, next);
			path = listing.path;
		} catch (cause) {
			error = toAppError(cause);
		} finally {
			loading = false;
		}
	}

	async function open(entry: FileEntry) {
		loading = true;
		error = null;

		try {
			openFile = await filesClient.read(serverId, entry.path);
		} catch (cause) {
			error = toAppError(cause);
		} finally {
			loading = false;
		}
	}

	async function confirmDelete() {
		if (!pendingDelete) return;
		deleting = true;

		try {
			await filesClient.remove(serverId, pendingDelete.path);
			pendingDelete = null;
			await load(path);
		} catch (cause) {
			error = toAppError(cause);
			pendingDelete = null;
		} finally {
			deleting = false;
		}
	}

	// Each segment of the path is clickable, which is how people navigate back
	// up several levels at once.
	const crumbs = $derived.by(() => {
		const parts = path.split('/').filter(Boolean);
		let built = '';
		return [
			{ label: '/', path: '/' },
			...parts.map((part) => {
				built += `/${part}`;
				return { label: part, path: built };
			})
		];
	});

	$effect(() => {
		void load('/');
	});
</script>

<Seo title="Files" description="Browse and edit files on your server." noindex />

<DashboardTopbar
	crumbs={[
		{ label: 'Dashboard', href: '/dashboard' },
		{ label: 'Servers', href: '/dashboard/servers' },
		{ label: 'Server', href: `/dashboard/servers/${serverId}` },
		{ label: 'Files' }
	]}
/>

<div class="space-y-5 px-6 py-8">
	{#if !openFile}
		<div class="flex flex-wrap items-center justify-between gap-3">
			<nav aria-label="Folder path" class="flex flex-wrap items-center gap-1 text-sm">
				{#each crumbs as crumb, index (crumb.path)}
					{#if index > 0}
						<span class="text-content-subtle" aria-hidden="true">/</span>
					{/if}
					<button
						type="button"
						onclick={() => load(crumb.path)}
						class="text-content-muted hover:text-content font-machine rounded-tile px-1.5 py-0.5 text-xs transition-colors duration-150"
						aria-current={index === crumbs.length - 1 ? 'page' : undefined}
					>
						{crumb.label}
					</button>
				{/each}
			</nav>

			<ButtonLink href="/dashboard/servers/{serverId}" variant="secondary" size="sm">
				<ArrowLeft size={15} aria-hidden="true" />
				Back to server
			</ButtonLink>
		</div>
	{/if}

	{#if error}
		<Card padded={false} class="overflow-hidden">
			<ErrorState {error} onRetry={() => load(path)} />
		</Card>
	{:else if loading}
		<Skeleton shape="card" class="h-96" />
	{:else if openFile}
		<FileEditor {serverId} file={openFile} onClose={() => (openFile = null)} />
	{:else if listing}
		<FileBrowser
			{serverId}
			{listing}
			onOpenDir={load}
			onOpenFile={open}
			onDelete={(entry) => (pendingDelete = entry)}
		/>
	{/if}

	<ConfirmDialog
		open={pendingDelete !== null}
		title="Delete {pendingDelete?.name ?? ''}?"
		description="This removes it from the server for good. There is no undo, and nothing is backed up first."
		confirmLabel="Delete it"
		danger
		busy={deleting}
		onConfirm={confirmDelete}
		onCancel={() => (pendingDelete = null)}
	/>
</div>
