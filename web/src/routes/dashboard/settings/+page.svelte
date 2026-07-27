<script lang="ts">
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import DestinationSheet from '$components/dashboard/DestinationSheet.svelte';
	import GitHubConnection from '$components/dashboard/GitHubConnection.svelte';
	import NotificationSettings from '$components/dashboard/NotificationSettings.svelte';
	import SectionHeader from '$components/dashboard/SectionHeader.svelte';
	import Seo from '$components/Seo.svelte';
	import { Button, Card, ConfirmDialog, ErrorState, Skeleton } from '$components/ui';
	import { backupsClient, type BackupDestination } from '$lib/data/backups';
	import { toAppError, type AppError } from '$lib/errors';
	import { formatRelativeTime } from '$utils/format';
	import HardDriveDownload from '@lucide/svelte/icons/hard-drive-download';

	let destination = $state<BackupDestination | null>(null);
	let loading = $state(true);
	let error = $state<AppError | null>(null);

	let editing = $state(false);
	let confirmForget = $state(false);
	let forgetting = $state(false);

	async function load() {
		loading = true;
		error = null;

		try {
			destination = await backupsClient.destination();
		} catch (cause) {
			error = toAppError(cause);
		} finally {
			loading = false;
		}
	}

	async function forget() {
		forgetting = true;
		try {
			await backupsClient.forgetDestination();
			confirmForget = false;
			await load();
		} catch (cause) {
			error = toAppError(cause);
		} finally {
			forgetting = false;
		}
	}

	$effect(() => {
		void load();
	});
</script>

<Seo title="Settings" description="How you are told about problems, and where backups are kept." noindex />

<DashboardTopbar crumbs={[{ label: 'Dashboard', href: '/dashboard' }, { label: 'Settings' }]} />

<div class="space-y-6 px-6 py-8">
	<SectionHeader title="Settings" description="How Ferrite Ship is set up for you." />

	<section class="space-y-3">
		<h2 class="text-content text-sm font-medium">When we should tell you something</h2>
		<p class="text-content-muted max-w-2xl text-sm leading-relaxed">
			Most of what happens here happens because you pressed a button and watched it run. These
			are the things that do not — and finding out about them a week later is no use.
		</p>
		<NotificationSettings />
	</section>

	<section class="space-y-3">
		<h2 class="text-content text-sm font-medium">Where your code comes from</h2>
		<GitHubConnection />
	</section>

	<section class="space-y-3">
		<h2 class="text-content text-sm font-medium">Where backups are kept</h2>

		{#if loading}
			<Skeleton shape="card" class="h-40" />
		{:else if error}
			<Card size="lg" padded={false} class="overflow-hidden">
				<ErrorState {error} onRetry={load} />
			</Card>
		{:else}
			<Card size="lg">
				<div class="flex flex-wrap items-start justify-between gap-4">
					<div class="flex gap-4">
						<HardDriveDownload size={20} aria-hidden="true" class="text-content-muted mt-0.5" />
						<div class="max-w-xl">
							{#if destination?.configured}
								<p class="text-content text-sm">
									Backups go to <span class="font-machine">{destination.bucket}</span>
									{#if destination.prefix}
										<span class="text-content-muted">/ {destination.prefix}</span>
									{/if}
								</p>
								<p class="text-content-subtle font-machine mt-1 text-xs">
									{destination.endpoint}
								</p>
								{#if destination.updatedAt}
									<p class="text-content-subtle mt-2 text-xs">
										Set {formatRelativeTime(destination.updatedAt, { style: 'short' })}
									</p>
								{/if}
							{:else}
								<p class="text-content text-sm">Backups have nowhere to go yet.</p>
								<p class="text-content-muted mt-1 text-sm leading-relaxed">
									Point this at storage somewhere other than your servers. A copy that lives on
									the same disk as the database survives a mistake, but not the disk.
								</p>
							{/if}
						</div>
					</div>

					<div class="flex items-center gap-2">
						{#if destination?.configured}
							<Button variant="ghost" size="sm" onclick={() => (confirmForget = true)}>
								Forget it
							</Button>
						{/if}
						<Button size="sm" onclick={() => (editing = true)}>
							{destination?.configured ? 'Change' : 'Set it up'}
						</Button>
					</div>
				</div>
			</Card>
		{/if}
	</section>
</div>

<DestinationSheet
	bind:open={editing}
	current={destination}
	onClose={() => (editing = false)}
	onSaved={() => {
		editing = false;
		void load();
	}}
/>

<ConfirmDialog
	bind:open={confirmForget}
	title="Forget this storage?"
	description="We stop sending backups here and delete the keys we hold."
	confirmLabel="Forget it"
	danger
	busy={forgetting}
	onConfirm={forget}
	onCancel={() => (confirmForget = false)}
>
	<p class="text-content-muted text-xs leading-relaxed">
		The backups already in the bucket are left exactly where they are — this only forgets how to
		reach them. You would need to enter the keys again to restore from them.
	</p>
</ConfirmDialog>
