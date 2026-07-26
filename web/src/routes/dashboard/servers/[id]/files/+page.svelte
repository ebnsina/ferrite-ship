<script lang="ts">
	import { page } from '$app/state';
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import DeleteFileDialog from '$components/dashboard/DeleteFileDialog.svelte';
	import FileBrowser from '$components/dashboard/FileBrowser.svelte';
	import FileBrowserSkeleton from '$components/dashboard/FileBrowserSkeleton.svelte';
	import FileEditor from '$components/dashboard/FileEditor.svelte';
	import FilePathBreadcrumb from '$components/dashboard/FilePathBreadcrumb.svelte';
	import Seo from '$components/Seo.svelte';
	import { ButtonLink, Card, ErrorState, Skeleton } from '$components/ui';
	import { dashboardRepository } from '$lib/data';
	import {
		filesClient,
		type DirectoryListing,
		type FileContent,
		type FileEntry
	} from '$lib/data/files';
	import { toAppError, type AppError } from '$lib/errors';
	import { createResource } from '$utils/resource.svelte';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';

	const serverId = page.params.id ?? '';

	// Loaded alongside the listing so the trail can name the server rather than
	// saying "Server", which tells nobody which machine they are editing.
	const server = createResource((signal) => dashboardRepository.getServer(serverId, signal));

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
		openFile = null;

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

	$effect(() => {
		void load('/');
	});
</script>

<Seo title="Files" description="Browse and edit files on your server." noindex />

<DashboardTopbar
	crumbs={[
		{ label: 'Dashboard', href: '/dashboard' },
		{ label: 'Servers', href: '/dashboard/servers' },
		{ label: server.data?.name ?? 'Server', href: `/dashboard/servers/${serverId}` },
		{ label: 'Files' }
	]}
/>

<div class="space-y-5 px-6 py-8">
	{#if !openFile}
		<div class="flex flex-wrap items-center justify-between gap-3">
			{#if loading && !listing}
				<Skeleton class="h-6 w-48" />
			{:else}
				<FilePathBreadcrumb {path} onNavigate={load} />
			{/if}

			<ButtonLink href="/dashboard/servers/{serverId}" variant="secondary" size="sm">
				<ArrowLeft size={15} aria-hidden="true" />
				Back to server
			</ButtonLink>
		</div>
	{/if}

	{#if error}
		<Card size="lg" padded={false} class="overflow-hidden">
			<ErrorState {error} onRetry={() => load(path)} />
		</Card>
	{:else if loading}
		<FileBrowserSkeleton />
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

	<DeleteFileDialog
		entry={pendingDelete}
		busy={deleting}
		onConfirm={confirmDelete}
		onCancel={() => (pendingDelete = null)}
	/>
</div>
