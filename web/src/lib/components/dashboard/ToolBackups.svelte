<script lang="ts">
	import { goto } from '$app/navigation';
	import { Button, ButtonLink, ConfirmDialog } from '$components/ui';
	import { backupsClient, type Backup } from '$lib/data/backups';
	import type { Tool } from '$lib/data/tools';
	import { toAppError } from '$lib/errors';
	import { formatBytes, formatRelativeTime } from '$utils/format';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import CircleX from '@lucide/svelte/icons/circle-x';
	import LoaderCircle from '@lucide/svelte/icons/loader-circle';
	import RotateCcw from '@lucide/svelte/icons/rotate-ccw';

	interface Props {
		serverId: string;
		tool: Tool;
	}

	let { serverId, tool }: Props = $props();

	let backups = $state<Backup[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let starting = $state(false);
	let restoring = $state<Backup | null>(null);

	// Whether the account has storage set up at all. Until it does, offering a
	// "Back up now" button that always fails is worse than saying so.
	let configured = $state<boolean | null>(null);

	async function load() {
		loading = true;
		try {
			const [list, destination] = await Promise.all([
				backupsClient.list(serverId, tool.id),
				backupsClient.destination()
			]);
			backups = list;
			configured = destination.configured;
		} catch (cause) {
			error = toAppError(cause).message;
		} finally {
			loading = false;
		}
	}

	async function backUp() {
		if (starting) return;
		starting = true;
		error = null;

		try {
			const job = await backupsClient.start(serverId, tool.id);
			await goto(`/dashboard/jobs/${job.id}`);
		} catch (cause) {
			error = toAppError(cause).message;
			starting = false;
		}
	}

	async function restore() {
		if (!restoring) return;
		const target = restoring;
		error = null;

		try {
			const job = await backupsClient.restore(target.id);
			await goto(`/dashboard/jobs/${job.id}`);
		} catch (cause) {
			error = toAppError(cause).message;
			restoring = null;
		}
	}

	const STATE = {
		ready: { icon: CircleCheck, class: 'text-ok', label: 'Kept' },
		running: { icon: LoaderCircle, class: 'text-info animate-spin', label: 'Copying' },
		failed: { icon: CircleX, class: 'text-error', label: 'Did not finish' }
	} as const;

	$effect(() => {
		void tool.id;
		void load();
	});
</script>

<div class="space-y-4">
	<div class="flex flex-wrap items-end justify-between gap-3">
		<div>
			<h2 class="text-content text-sm font-medium">Backups</h2>
			<p class="text-content-muted mt-0.5 text-sm">
				Copies kept away from this server, so they outlive the machine.
			</p>
		</div>

		{#if configured}
			<Button onclick={backUp} disabled={starting}>
				{starting ? 'Starting…' : 'Back up now'}
			</Button>
		{/if}
	</div>

	{#if error}
		<div class="border-error/40 bg-error-soft rounded-card-sm border p-4">
			<p class="text-error text-sm">{error}</p>
		</div>
	{/if}

	{#if loading}
		<p class="text-content-subtle text-sm">Looking…</p>
	{:else if configured === false}
		<!-- The button is absent rather than disabled: there is somewhere to go
		     and something to do, and a dead button says neither. -->
		<div class="border-border rounded-card-sm border border-dashed p-5">
			<p class="text-content text-sm">Backups have nowhere to go yet.</p>
			<p class="text-content-muted mt-1 max-w-xl text-sm leading-relaxed">
				Tell us where to keep them and this fills up. Somewhere other than this server — a copy on
				the same disk survives a mistake, but not the disk.
			</p>
			<div class="mt-4">
				<ButtonLink href="/dashboard/settings" size="sm">Choose somewhere</ButtonLink>
			</div>
		</div>
	{:else if backups.length === 0}
		<div class="border-border rounded-card-sm border border-dashed p-5">
			<p class="text-content-muted text-sm">
				No backups of {tool.name} yet. Take one whenever you like.
			</p>
		</div>
	{:else}
		<div class="border-border rounded-card-sm divide-border/70 divide-y overflow-hidden border">
			{#each backups as backup (backup.id)}
				{@const state = STATE[backup.status]}
				{@const Icon = state.icon}
				<div class="flex flex-wrap items-center gap-3 px-4 py-3">
					<Icon size={15} aria-hidden="true" class="shrink-0 {state.class}" />

					<div class="min-w-0 flex-1">
						<p class="text-content text-sm">
							{formatRelativeTime(backup.createdAt, { style: 'long' })}
						</p>
						<p class="text-content-subtle mt-0.5 text-xs">
							{state.label}{#if backup.status === 'ready'}
								· {formatBytes(backup.sizeBytes)}{/if}
						</p>
					</div>

					{#if backup.jobId}
						<ButtonLink href="/dashboard/jobs/{backup.jobId}" variant="ghost" size="sm">
							See the run
						</ButtonLink>
					{/if}

					{#if backup.status === 'ready'}
						<Button variant="ghost" size="sm" onclick={() => (restoring = backup)}>
							<RotateCcw size={15} aria-hidden="true" />
							Restore
						</Button>
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</div>

{#if restoring}
	<ConfirmDialog
		open={true}
		title="Restore {tool.name} from {formatRelativeTime(restoring.createdAt, { style: 'long' })}?"
		description="This replaces everything in {tool.name} with the contents of that backup."
		confirmLabel="Replace it"
		danger
		onConfirm={restore}
		onCancel={() => (restoring = null)}
	>
		<p class="text-content-muted text-xs leading-relaxed">
			Anything written since that backup was taken is lost, and there is no undo. If you are not
			certain, take a backup now first — then you can come back to this moment.
		</p>
	</ConfirmDialog>
{/if}
