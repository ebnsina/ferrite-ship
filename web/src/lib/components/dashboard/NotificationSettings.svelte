<script lang="ts">
	import { Button, Card, Skeleton, Switch, TextField } from '$components/ui';
	import {
		notificationsClient,
		type NotificationSettings as Settings
	} from '$lib/data/notifications';
	import { toAppError } from '$lib/errors';
	import { formatRelativeTime } from '$utils/format';
	import CircleCheck from '@lucide/svelte/icons/circle-check';

	let settings = $state<Settings | null>(null);
	let loading = $state(true);
	let saving = $state(false);
	let testing = $state(false);
	let error = $state<string | null>(null);
	let sentTo = $state<string | null>(null);

	let email = $state('');
	let onBackupFailed = $state(true);
	let onServerDown = $state(true);
	let onDiskLow = $state(true);
	let diskPercent = $state(85);

	function adopt(loaded: Settings) {
		settings = loaded;
		email = loaded.email;
		onBackupFailed = loaded.onBackupFailed;
		onServerDown = loaded.onServerDown;
		onDiskLow = loaded.onDiskLow;
		diskPercent = loaded.diskPercent || 85;
	}

	async function load() {
		loading = true;
		error = null;

		try {
			adopt(await notificationsClient.get());
		} catch (cause) {
			error = toAppError(cause).message;
		} finally {
			loading = false;
		}
	}

	async function save() {
		saving = true;
		error = null;
		sentTo = null;

		try {
			adopt(
				await notificationsClient.save({
					email,
					onBackupFailed,
					onServerDown,
					onDiskLow,
					diskPercent
				})
			);
		} catch (cause) {
			error = toAppError(cause).message;
		} finally {
			saving = false;
		}
	}

	// Save first, then send. Otherwise the test goes to whatever address was
	// stored last time, and reports success for a message the person will
	// never see.
	async function test() {
		testing = true;
		error = null;
		sentTo = null;

		try {
			await save();
			const result = await notificationsClient.test();
			sentTo = result.sentTo;
		} catch (cause) {
			error = toAppError(cause).message;
		} finally {
			testing = false;
		}
	}

	const changed = $derived(
		settings !== null &&
			(email !== settings.email ||
				onBackupFailed !== settings.onBackupFailed ||
				onServerDown !== settings.onServerDown ||
				onDiskLow !== settings.onDiskLow ||
				diskPercent !== settings.diskPercent)
	);

	$effect(() => {
		void load();
	});
</script>

{#if loading}
	<Skeleton shape="card" class="h-64" />
{:else if settings}
	<Card size="lg">
		<div class="max-w-xl space-y-5">
			<TextField
				label="Send alerts to"
				type="email"
				bind:value={email}
				placeholder="you@example.com"
				hint="Leave this empty to be told nothing."
			/>

			{#if !settings.canSend}
				<!-- Said plainly rather than by disabling the switches. The
				     preferences are still worth setting; what is missing is a mail
				     server, and that is not something this page can fix. -->
				<div class="border-warn/40 bg-warn-soft rounded-card-sm border p-4">
					<p class="text-warn text-sm">This installation cannot send email yet.</p>
					<p class="text-content-muted mt-1 text-xs leading-relaxed">
						Your choices are saved and problems are still shown on the dashboard, but nothing will
						arrive in your inbox until a mail server is configured.
					</p>
				</div>
			{/if}

			<div class="space-y-4">
				<Switch
					bind:checked={onBackupFailed}
					label="A scheduled backup did not finish"
					description="The one that runs on its own, at an hour when nobody is looking."
				/>
				<Switch
					bind:checked={onServerDown}
					label="A server stopped responding"
					description="Checked every five minutes. You get one message when it goes, and one when it comes back."
				/>
				<Switch
					bind:checked={onDiskLow}
					label="A disk is nearly full"
					description="Before it fills, while there is still time to do something about it."
				/>
			</div>

			{#if onDiskLow}
				<label class="text-content-muted block text-xs font-medium">
					Tell me when a disk passes
					<select
						bind:value={diskPercent}
						class="border-border bg-surface-sunken text-content rounded-field mt-1.5 block h-9 border px-3 text-sm outline-none"
					>
						{#each [70, 80, 85, 90, 95] as percent (percent)}
							<option value={percent}>{percent}% full</option>
						{/each}
					</select>
				</label>
			{/if}

			{#if error}
				<p class="text-error text-sm">{error}</p>
			{/if}

			{#if sentTo}
				<p class="text-ok flex items-center gap-2 text-sm">
					<CircleCheck size={15} aria-hidden="true" />
					Sent to {sentTo}. If it does not arrive, check the spam folder.
				</p>
			{/if}

			<div class="flex items-center gap-2">
				<Button onclick={save} disabled={saving || testing || !changed}>
					{saving && !testing ? 'Saving…' : 'Save'}
				</Button>
				<Button
					variant="secondary"
					onclick={test}
					disabled={saving || testing || email.trim() === ''}
				>
					{testing ? 'Sending…' : 'Send a test'}
				</Button>
			</div>

			{#if settings.updatedAt}
				<p class="text-content-subtle text-xs">
					Last changed {formatRelativeTime(settings.updatedAt, { style: 'short' })}.
				</p>
			{/if}
		</div>
	</Card>
{/if}
