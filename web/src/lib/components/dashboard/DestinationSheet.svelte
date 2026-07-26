<script lang="ts">
	import { Button, Sheet, TextField } from '$components/ui';
	import { backupsClient, type BackupDestination } from '$lib/data/backups';
	import { toAppError } from '$lib/errors';
	import ShieldCheck from '@lucide/svelte/icons/shield-check';

	interface Props {
		open: boolean;
		/** The destination as it stands, to prefill everything except the keys. */
		current: BackupDestination | null;
		onClose: () => void;
		onSaved: () => void;
	}

	let { open = $bindable(), current, onClose, onSaved }: Props = $props();

	let endpoint = $state('');
	let region = $state('');
	let bucket = $state('');
	let prefix = $state('');
	let accessKey = $state('');
	let secretKey = $state('');

	let saving = $state(false);
	let error = $state<string | null>(null);

	// Prefill from what is stored each time it opens — except the keys, which
	// the API does not return and so cannot be shown. Re-entering them is the
	// price of never sending them back to a browser.
	$effect(() => {
		if (!open) return;
		endpoint = current?.endpoint ?? '';
		region = current?.region ?? '';
		bucket = current?.bucket ?? '';
		prefix = current?.prefix ?? '';
		accessKey = '';
		secretKey = '';
		error = null;
	});

	async function save() {
		if (saving) return;
		saving = true;
		error = null;

		try {
			await backupsClient.saveDestination({
				endpoint: endpoint.trim(),
				region: region.trim(),
				bucket: bucket.trim(),
				prefix: prefix.trim(),
				accessKey,
				secretKey
			});
			onSaved();
		} catch (cause) {
			error = toAppError(cause).message;
		} finally {
			saving = false;
		}
	}
</script>

<Sheet
	bind:open
	size="lg"
	title={current?.configured ? 'Change where backups go' : 'Choose where backups go'}
	description="Any storage that speaks S3 works — Amazon, Cloudflare R2, Backblaze B2, DigitalOcean Spaces, or your own MinIO."
	busy={saving}
	{onClose}
>
	<form
		id="backup-destination"
		class="space-y-6"
		onsubmit={(event) => {
			event.preventDefault();
			void save();
		}}
	>
		{#if error}
			<div class="border-error/40 bg-error-soft rounded-card-sm border p-4">
				<p class="text-error text-sm">{error}</p>
			</div>
		{/if}

		<TextField
			label="Address of your storage"
			bind:value={endpoint}
			required
			placeholder="https://s3.eu-central-1.amazonaws.com"
			hint="The endpoint your provider gave you, including https://."
		/>

		<div class="grid gap-5 sm:grid-cols-2">
			<TextField
				label="Bucket"
				bind:value={bucket}
				required
				placeholder="my-backups"
				hint="Create it with your provider first — we do not make one for you."
			/>
			<TextField
				label="Region"
				bind:value={region}
				placeholder="eu-central-1"
				hint="Optional. Some providers ignore it."
			/>
		</div>

		<TextField
			label="Folder inside the bucket"
			bind:value={prefix}
			placeholder="ferrite"
			hint="Optional. Keeps these backups apart from anything else in the bucket."
		/>

		<div class="border-border rounded-card-sm space-y-5 border p-5">
			<div class="flex gap-3">
				<ShieldCheck size={17} aria-hidden="true" class="text-ok mt-0.5 shrink-0" />
				<p class="text-content-muted text-xs leading-relaxed">
					These are encrypted before they are stored and are never shown again — not here, not
					anywhere. Changing anything else on this page means entering them once more, which is the
					trade for never sending them back to a browser.
					<span class="text-content">
						Use keys that can write to this one bucket and nothing else.
					</span>
				</p>
			</div>

			<TextField
				label="Access key"
				bind:value={accessKey}
				required
				autocomplete="off"
				placeholder="AKIA…"
			/>
			<TextField
				label="Secret key"
				type="password"
				bind:value={secretKey}
				required
				autocomplete="off"
			/>
		</div>
	</form>

	{#snippet footer()}
		<Button variant="ghost" onclick={onClose} disabled={saving}>Cancel</Button>
		<Button type="submit" form="backup-destination" disabled={saving}>
			{saving ? 'Saving…' : 'Save'}
		</Button>
	{/snippet}
</Sheet>
