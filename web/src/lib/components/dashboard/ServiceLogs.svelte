<script lang="ts">
	import { Button, Card, Skeleton } from '$components/ui';
	import { servicesClient } from '$lib/data/services';
	import { toAppError } from '$lib/errors';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import RotateCw from '@lucide/svelte/icons/rotate-cw';

	interface Props {
		serverId: string;
		unit: string;
		onClose: () => void;
	}

	let { serverId, unit, onClose }: Props = $props();

	let text = $state('');
	let loading = $state(true);
	let error = $state<string | null>(null);

	async function load() {
		loading = true;
		error = null;

		try {
			const result = await servicesClient.logs(serverId, unit);
			text = result.text.trimEnd();
		} catch (cause) {
			error = toAppError(cause).message;
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		// Re-reads whenever a different unit is opened.
		void unit;
		void load();
	});
</script>

<Card size="lg" padded={false} class="overflow-hidden">
	<div class="border-border/70 flex flex-wrap items-center gap-3 border-b px-5 py-3">
		<Button variant="ghost" size="sm" onclick={onClose}>
			<ArrowLeft size={15} aria-hidden="true" />
			Back
		</Button>

		<div class="min-w-0 flex-1">
			<p class="text-content font-machine truncate text-xs">{unit}</p>
			<p class="text-content-subtle mt-0.5 text-xs">Most recent messages, oldest first</p>
		</div>

		<Button variant="secondary" size="sm" onclick={load} disabled={loading}>
			<RotateCw size={14} aria-hidden="true" />
			Refresh
		</Button>
	</div>

	{#if error}
		<p class="bg-error-soft text-error px-5 py-3 text-sm">{error}</p>
	{:else if loading}
		<div class="space-y-2 p-5">
			{#each Array.from({ length: 12 }, (_, index) => index) as index (index)}
				<Skeleton class="h-3" />
			{/each}
		</div>
	{:else if text === ''}
		<p class="text-content-subtle px-5 py-10 text-center text-sm">
			This service has not written anything to the log.
		</p>
	{:else}
		<pre
			class="font-machine bg-surface-sunken text-content-muted max-h-[60vh] overflow-auto px-5 py-4 text-xs leading-relaxed">{text}</pre>
	{/if}
</Card>
