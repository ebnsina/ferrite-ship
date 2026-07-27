<script lang="ts">
	import { Button } from '$components/ui';
	import { schedulesClient, type BackupSchedule, type Cadence } from '$lib/data/backups';
	import { toAppError } from '$lib/errors';
	import { formatRelativeTime } from '$utils/format';
	import CalendarClock from '@lucide/svelte/icons/calendar-clock';

	interface Props {
		serverId: string;
		toolId: string;
		toolName: string;
		/** Called after a change, so the backup list can refresh alongside. */
		onChanged?: () => void;
	}

	let { serverId, toolId, toolName, onChanged }: Props = $props();

	let schedule = $state<BackupSchedule | null>(null);
	let editing = $state(false);
	let busy = $state(false);
	let error = $state<string | null>(null);

	let cadence = $state<Cadence>('daily');
	let hour = $state(3);
	let weekday = $state(0);
	let keep = $state(7);

	const DAYS = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];

	async function load() {
		try {
			schedule = await schedulesClient.get(serverId, toolId);
			if (schedule) {
				cadence = schedule.cadence;
				hour = schedule.hour;
				weekday = schedule.weekday;
				keep = schedule.keep;
			}
		} catch {
			// A schedule that will not load is not worth an error over the
			// backups themselves, which still work by hand.
			schedule = null;
		}
	}

	async function save() {
		busy = true;
		error = null;

		try {
			schedule = await schedulesClient.save(serverId, toolId, { cadence, hour, weekday, keep });
			editing = false;
			onChanged?.();
		} catch (cause) {
			error = toAppError(cause).message;
		} finally {
			busy = false;
		}
	}

	async function turnOff() {
		busy = true;
		error = null;

		try {
			await schedulesClient.turnOff(serverId, toolId);
			schedule = null;
			editing = false;
			onChanged?.();
		} catch (cause) {
			error = toAppError(cause).message;
		} finally {
			busy = false;
		}
	}

	// Times are UTC throughout, and the label says so. A schedule that silently
	// meant something else in another timezone would be very hard to explain
	// when a backup arrived at the wrong hour.
	const summary = $derived.by(() => {
		if (!schedule) return 'Only when you ask.';
		const at = `${String(schedule.hour).padStart(2, '0')}:00 UTC`;
		return schedule.cadence === 'weekly'
			? `Every ${DAYS[schedule.weekday]} at ${at}, keeping the newest ${schedule.keep}.`
			: `Every day at ${at}, keeping the newest ${schedule.keep}.`;
	});

	$effect(() => {
		void toolId;
		void load();
	});
</script>

<div class="border-border rounded-card-sm border p-4">
	<div class="flex flex-wrap items-start justify-between gap-3">
		<div class="flex gap-3">
			<CalendarClock size={17} aria-hidden="true" class="text-content-muted mt-0.5 shrink-0" />
			<div>
				<p class="text-content text-sm font-medium">Automatic backups</p>
				<p class="text-content-muted mt-0.5 text-sm">{summary}</p>
				{#if schedule}
					<p class="text-content-subtle mt-1 text-xs">
						Next {formatRelativeTime(schedule.nextRunAt, { style: 'long' })}{#if schedule.lastRunAt}
							· last ran {formatRelativeTime(schedule.lastRunAt, { style: 'short' })}{/if}
					</p>
				{/if}
			</div>
		</div>

		{#if !editing}
			<div class="flex items-center gap-2">
				{#if schedule}
					<Button variant="ghost" size="sm" onclick={turnOff} disabled={busy}>Turn off</Button>
				{/if}
				<Button variant="secondary" size="sm" onclick={() => (editing = true)}>
					{schedule ? 'Change' : 'Set a schedule'}
				</Button>
			</div>
		{/if}
	</div>

	{#if editing}
		<form
			class="mt-4 space-y-4"
			onsubmit={(event) => {
				event.preventDefault();
				void save();
			}}
		>
			<div class="flex flex-wrap items-end gap-4">
				<label class="text-content-muted text-xs font-medium">
					How often
					<select
						bind:value={cadence}
						class="border-border bg-surface-sunken text-content rounded-field mt-1.5 block h-9 border px-3 text-sm outline-none"
					>
						<option value="daily">Every day</option>
						<option value="weekly">Every week</option>
					</select>
				</label>

				{#if cadence === 'weekly'}
					<label class="text-content-muted text-xs font-medium">
						On
						<select
							bind:value={weekday}
							class="border-border bg-surface-sunken text-content rounded-field mt-1.5 block h-9 border px-3 text-sm outline-none"
						>
							{#each DAYS as day, index (day)}
								<option value={index}>{day}</option>
							{/each}
						</select>
					</label>
				{/if}

				<label class="text-content-muted text-xs font-medium">
					At (UTC)
					<select
						bind:value={hour}
						class="border-border bg-surface-sunken text-content rounded-field mt-1.5 block h-9 border px-3 text-sm outline-none"
					>
						{#each { length: 24 } as _, value (value)}
							<option {value}>{String(value).padStart(2, '0')}:00</option>
						{/each}
					</select>
				</label>

				<label class="text-content-muted text-xs font-medium">
					Keep the newest
					<select
						bind:value={keep}
						class="border-border bg-surface-sunken text-content rounded-field mt-1.5 block h-9 border px-3 text-sm outline-none"
					>
						{#each [3, 7, 14, 30] as count (count)}
							<option value={count}>{count}</option>
						{/each}
					</select>
				</label>
			</div>

			<p class="text-content-subtle text-xs leading-relaxed">
				Older backups are deleted from your storage after each successful run, once the new one has
				arrived. {toolName} is backed up while it keeps running.
			</p>

			{#if error}
				<p class="text-error text-xs">{error}</p>
			{/if}

			<div class="flex items-center gap-2">
				<Button type="submit" size="sm" disabled={busy}>{busy ? 'Saving…' : 'Save'}</Button>
				<Button variant="ghost" size="sm" onclick={() => (editing = false)} disabled={busy}>
					Cancel
				</Button>
			</div>
		</form>
	{/if}
</div>
