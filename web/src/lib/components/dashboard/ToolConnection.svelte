<script lang="ts">
	import CopyField from '$components/dashboard/CopyField.svelte';
	import { Card, ErrorState, Skeleton } from '$components/ui';
	import { toolsClient, type ToolConnection } from '$lib/data/tools';
	import { toAppError, type AppError } from '$lib/errors';
	import Globe from '@lucide/svelte/icons/globe';
	import Lock from '@lucide/svelte/icons/lock';

	interface Props {
		serverId: string;
		toolId: string;
	}

	let { serverId, toolId }: Props = $props();

	let connection = $state<ToolConnection | null>(null);
	let loading = $state(true);
	let error = $state<AppError | null>(null);

	async function load() {
		loading = true;
		error = null;

		try {
			connection = await toolsClient.connection(serverId, toolId);
		} catch (cause) {
			error = toAppError(cause);
		} finally {
			loading = false;
		}
	}

	// Re-reads whenever the tool changes, so opening a second tool's details
	// does not show the first one's password.
	$effect(() => {
		void toolId;
		void load();
	});
</script>

<Card size="sm" class="bg-surface-sunken/40">
	{#if loading}
		<div class="space-y-3">
			<Skeleton class="h-3 w-32" />
			<Skeleton class="h-9 w-full" />
			<Skeleton class="h-9 w-full" />
		</div>
	{:else if error}
		<ErrorState {error} onRetry={load} />
	{:else if connection}
		<div class="flex items-start gap-2">
			{#if connection.web && connection.public}
				<Globe size={15} aria-hidden="true" class="text-warn mt-0.5 shrink-0" />
				<p class="text-content-muted text-xs leading-relaxed">
					<span class="text-content font-medium">Open this in a browser.</span>
					It has its own web address with a certificate, so anyone can reach the sign-in
					page — keep the password to yourself.
				</p>
			{:else if connection.public}
				<Globe size={15} aria-hidden="true" class="text-warn mt-0.5 shrink-0" />
				<p class="text-content-muted text-xs leading-relaxed">
					<span class="text-content font-medium">Anyone can reach this.</span>
					It is open to the internet on purpose, so keep the password to yourself.
				</p>
			{:else}
				<Lock size={15} aria-hidden="true" class="text-ok mt-0.5 shrink-0" />
				<p class="text-content-muted text-xs leading-relaxed">
					<span class="text-content font-medium">Only this server can reach it.</span>
					Nothing on the internet can connect. To use it from your own computer, run the tunnel
					command below and leave it open.
				</p>
			{/if}
		</div>

		<div class="mt-5 space-y-4">
			{#if connection.tunnel}
				<CopyField
					label="1. Open the tunnel, and leave it running"
					value={connection.tunnel}
					hint="Run this in a terminal on your own computer. Nothing is shown while it works."
				/>
			{/if}

			{#if connection.web}
				<!-- Not hidden: a web address has no password in it, and masking
				     something harmless teaches people to reveal fields without
				     thinking about which ones actually matter. -->
				<CopyField
					label={connection.tunnel ? '2. Then open this' : 'Open this'}
					value={connection.url}
					hint="Paste it into a browser and sign in with the details below."
				/>
			{:else}
				<!-- Hidden by default like the password field below, because it
				     contains that same password. Masking one and printing the other
				     in full would be no protection at all. -->
				<CopyField
					label={connection.tunnel ? '2. Then connect with this' : 'Connect with this'}
					value={connection.url}
					secret
					hint="Most tools accept this whole line as the connection address. It has the password in it, so treat it like one."
				/>
			{/if}

			<div class="grid gap-4 sm:grid-cols-2">
				<CopyField label="Username" value={connection.username} />
				<CopyField label="Password" value={connection.password} secret />
			</div>
		</div>
	{/if}
</Card>
