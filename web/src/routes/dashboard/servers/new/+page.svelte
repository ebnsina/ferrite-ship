<script lang="ts">
	import { goto } from '$app/navigation';
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import SectionHeader from '$components/dashboard/SectionHeader.svelte';
	import {
		Button,
		ButtonLink,
		Card,
		ChoiceCard,
		ErrorState,
		TextArea,
		TextField
	} from '$components/ui';
	import { dashboardCommands, type ConnectionKind } from '$lib/data/commands';
	import { toAppError, type AppError } from '$lib/errors';
	import FlaskConical from '@lucide/svelte/icons/flask-conical';
	import Server from '@lucide/svelte/icons/server';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';

	let kind = $state<ConnectionKind>('demo');
	let name = $state('');
	let host = $state('');
	let port = $state('22');
	let user = $state('root');
	let region = $state('');
	let authMethod = $state<'privateKey' | 'password'>('privateKey');
	let privateKey = $state('');
	let password = $state('');

	let submitting = $state(false);
	let error = $state<AppError | null>(null);

	const isSSH = $derived(kind === 'ssh');

	async function connect(runSetup: boolean) {
		if (submitting) return;

		error = null;
		submitting = true;

		try {
			const server = await dashboardCommands.connectServer({
				name: name.trim(),
				connectionKind: kind,
				region: region.trim() || undefined,
				...(isSSH
					? {
							host: host.trim(),
							port: Number(port) || 22,
							user: user.trim(),
							...(authMethod === 'password'
								? { password }
								: { privateKey })
						}
					: {})
			});

			if (!runSetup) {
				await goto('/dashboard/servers');
				return;
			}

			const job = await dashboardCommands.startBaseline(server.id);
			await goto(`/dashboard/jobs/${job.id}`);
		} catch (cause) {
			error = toAppError(cause);
			submitting = false;
		}
	}
</script>

<svelte:head>
	<title>Connect a server · ferrite-ship</title>
</svelte:head>

<DashboardTopbar
	crumbs={[
		{ label: 'Dashboard', href: '/dashboard' },
		{ label: 'Servers', href: '/dashboard/servers' },
		{ label: 'Connect' }
	]}
/>

<div class="mx-auto max-w-2xl space-y-6 px-6 py-8">
	<SectionHeader
		title="Connect a server"
		description="Point us at a machine and we will check it over, then set it up safely."
	/>

	{#if error}
		<Card padded={false}>
			<ErrorState {error} onRetry={() => (error = null)} />
		</Card>
	{/if}

	<form
		class="space-y-6"
		onsubmit={(event) => {
			event.preventDefault();
			void connect(true);
		}}
	>
		<fieldset class="space-y-3">
			<legend class="text-content mb-3 text-sm font-medium">What are we connecting?</legend>

			<ChoiceCard
				name="kind"
				value="demo"
				bind:group={kind}
				icon={FlaskConical}
				title="A pretend server"
				description="Nothing real is touched. Best for seeing how this works before you point it at a machine you own."
			/>

			<ChoiceCard
				name="kind"
				value="ssh"
				bind:group={kind}
				icon={Server}
				title="A real server"
				description="Ubuntu 22.04 or 24.04, reachable over SSH, with an account that can install software."
			/>
		</fieldset>

		<Card class="space-y-5">
			<TextField
				label="Name"
				bind:value={name}
				required
				placeholder="edge-1"
				hint="Just for you — how it will appear in your list."
			/>

			<TextField
				label="Location"
				bind:value={region}
				placeholder="Frankfurt"
				hint="Optional. Where the machine lives."
			/>

			{#if isSSH}
				<div class="grid gap-5 sm:grid-cols-[1fr_8rem]">
					<TextField label="Address" bind:value={host} required placeholder="203.0.113.10" />
					<TextField label="Port" bind:value={port} inputmode="numeric" placeholder="22" />
				</div>

				<TextField
					label="Username"
					bind:value={user}
					required
					placeholder="root"
					hint="Must be able to install software without being asked for a password."
				/>

				<fieldset class="space-y-3">
					<legend class="text-content mb-2 text-sm font-medium">How do we sign in?</legend>

					<div class="flex gap-4">
						<label class="text-content-muted flex items-center gap-2 text-sm">
							<input type="radio" name="auth" value="privateKey" bind:group={authMethod} />
							With a key
						</label>
						<label class="text-content-muted flex items-center gap-2 text-sm">
							<input type="radio" name="auth" value="password" bind:group={authMethod} />
							With a password
						</label>
					</div>

					{#if authMethod === 'privateKey'}
						<TextArea
							label="Private key"
							bind:value={privateKey}
							rows={6}
							required
							autocomplete="off"
							spellcheck={false}
							placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
							hint="Encrypted before it is stored. We never write it to a log."
						/>
					{:else}
						<TextField
							label="Password"
							type="password"
							bind:value={password}
							required
							autocomplete="off"
							hint="Encrypted before it is stored. We never write it to a log."
						/>
					{/if}
				</fieldset>

				<div class="border-warn/40 bg-warn-soft flex gap-3 rounded-card border p-4">
					<TriangleAlert size={17} aria-hidden="true" class="text-warn mt-0.5 shrink-0" />
					<p class="text-content-muted text-xs leading-relaxed">
						Setting up changes how the server accepts logins and turns on a firewall. Try it on a
						machine you can afford to lose first. We will not turn off password logins unless a key
						is already in place, so you cannot be locked out.
					</p>
				</div>
			{/if}
		</Card>

		<div class="flex flex-wrap items-center gap-3">
			<Button type="submit" disabled={submitting}>
				{submitting ? 'Connecting…' : 'Connect and set up'}
			</Button>
			<Button
				type="button"
				variant="secondary"
				disabled={submitting}
				onclick={() => void connect(false)}
			>
				Just connect
			</Button>
			<ButtonLink href="/dashboard/servers" variant="ghost">Cancel</ButtonLink>
		</div>
	</form>
</div>
