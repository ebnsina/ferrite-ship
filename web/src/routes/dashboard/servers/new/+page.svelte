<script lang="ts">
	import { goto } from '$app/navigation';
	import Seo from '$components/Seo.svelte';
	import { Button, ChoiceCard, ErrorState, Sheet, TextArea, TextField } from '$components/ui';
	import { dashboardCommands } from '$lib/data/commands';
	import { toAppError, type AppError } from '$lib/errors';
	import CircleDashed from '@lucide/svelte/icons/circle-dashed';
	import Play from '@lucide/svelte/icons/play';
	import Search from '@lucide/svelte/icons/search';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';

	/** What happens once the server is connected. */
	type NextStep = 'setup' | 'check' | 'none';

	let name = $state('');
	let host = $state('');
	let port = $state('22');
	let user = $state('root');
	let region = $state('');
	let authMethod = $state<'privateKey' | 'password'>('privateKey');
	let privateKey = $state('');
	let password = $state('');
	let publicKey = $state('');
	let nextStep = $state<NextStep>('setup');

	let submitting = $state(false);
	let error = $state<AppError | null>(null);

	// The sheet is the page: the route still exists so the link can be shared
	// and the browser's back button behaves, but closing it returns to the list
	// underneath rather than leaving an empty screen.
	function close() {
		void goto('/dashboard/servers');
	}

	async function connect() {
		if (submitting) return;

		error = null;
		submitting = true;

		try {
			const server = await dashboardCommands.connectServer({
				name: name.trim(),
				connectionKind: 'ssh',
				region: region.trim() || undefined,
				publicKey: publicKey.trim() || undefined,
				host: host.trim(),
				port: Number(port) || 22,
				user: user.trim(),
				...(authMethod === 'password' ? { password } : { privateKey })
			});

			if (nextStep === 'none') {
				await goto('/dashboard/servers');
				return;
			}

			const job = await dashboardCommands.startBaseline(server.id, {
				dryRun: nextStep === 'check'
			});
			await goto(`/dashboard/jobs/${job.id}`);
		} catch (cause) {
			error = toAppError(cause);
			submitting = false;
		}
	}
</script>

<Seo
	title="Connect a server"
	description="Point Ferrite Ship at a machine and set it up safely."
	noindex
/>

<Sheet
	open={true}
	size="lg"
	title="Connect a server"
	description="Point us at a machine and we will check it over, then set it up safely."
	busy={submitting}
	onClose={close}
>
	<form id="connect-server" class="space-y-6" onsubmit={(event) => {
		event.preventDefault();
		void connect();
	}}>
		{#if error}
			<div class="border-border rounded-card-sm overflow-hidden border">
				<ErrorState {error} onRetry={() => (error = null)} />
			</div>
		{/if}

		<div class="grid gap-5 sm:grid-cols-2">
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
		</div>

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

		<TextArea
			label="Your public key"
			bind:value={publicKey}
			rows={3}
			autocomplete="off"
			spellcheck={false}
			placeholder="ssh-ed25519 AAAA... you@laptop"
			hint="Optional, but without it the account we create cannot be logged into — and we will leave password logins on rather than lock you out."
		/>

		<!--
			The three things that used to be three submit buttons. As a choice
			they can carry a sentence each, which the buttons could not, and the
			footer is left with one action whose label says what will happen.
		-->
		<fieldset class="space-y-3">
			<legend class="text-content mb-2 text-sm font-medium">Then what?</legend>

			<ChoiceCard
				name="next"
				value="setup"
				bind:group={nextStep}
				icon={Play}
				title="Set it up"
				description="Updates, a login account, a firewall and automatic security updates."
			/>
			<ChoiceCard
				name="next"
				value="check"
				bind:group={nextStep}
				icon={Search}
				title="Only check what would change"
				description="Runs every check and reports back. Nothing on the machine is altered."
			/>
			<ChoiceCard
				name="next"
				value="none"
				bind:group={nextStep}
				icon={CircleDashed}
				title="Nothing for now"
				description="Just add it to your list. You can set it up whenever you like."
			/>
		</fieldset>

		{#if nextStep === 'setup'}
			<div class="border-warn/40 bg-warn-soft flex gap-3 rounded-card-sm border p-4">
				<TriangleAlert size={17} aria-hidden="true" class="text-warn mt-0.5 shrink-0" />
				<p class="text-content-muted text-xs leading-relaxed">
					Setting up changes how the server accepts logins and turns on a firewall. Try it on a
					machine you can afford to lose first. We will not turn off password logins unless a key is
					already in place, so you cannot be locked out.
				</p>
			</div>
		{/if}
	</form>

	{#snippet footer()}
		<Button variant="ghost" onclick={close} disabled={submitting}>Cancel</Button>
		<Button type="submit" form="connect-server" disabled={submitting}>
			{#if submitting}
				Connecting…
			{:else if nextStep === 'setup'}
				Connect and set up
			{:else if nextStep === 'check'}
				Connect and check
			{:else}
				Connect
			{/if}
		</Button>
	{/snippet}
</Sheet>
