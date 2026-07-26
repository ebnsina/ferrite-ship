<script lang="ts">
	import { Button, Sheet, TextArea, TextField } from '$components/ui';
	import { appsClient, type App } from '$lib/data/apps';
	import { toAppError } from '$lib/errors';

	interface Props {
		open: boolean;
		serverId: string;
		/** The app being edited, or null to add a new one. */
		app: App | null;
		onClose: () => void;
		onSaved: (app: App) => void;
	}

	let { open = $bindable(), serverId, app, onClose, onSaved }: Props = $props();

	let name = $state('');
	let repository = $state('');
	let branch = $state('');
	let domain = $state('');
	let port = $state('3000');
	let envText = $state('');
	let deployKey = $state('');

	let saving = $state(false);
	let error = $state<string | null>(null);

	$effect(() => {
		if (!open) return;
		name = app?.name ?? '';
		repository = app?.repository ?? '';
		branch = app?.branch ?? 'main';
		domain = app?.domain ?? '';
		port = String(app?.port ?? 3000);
		// Existing variables are not shown: they are sealed and never returned,
		// for the same reason storage keys are not.
		envText = '';
		deployKey = '';
		error = null;
	});

	/**
	 * Parses KEY=value lines, the way every .env file people already have is
	 * written — so configuring an app is pasting what you already had rather
	 * than filling in a row of boxes one at a time.
	 */
	function parseEnv(text: string): Record<string, string> {
		const env: Record<string, string> = {};

		for (const raw of text.split('\n')) {
			const line = raw.trim();
			if (!line || line.startsWith('#')) continue;

			const at = line.indexOf('=');
			if (at < 1) continue;

			const key = line.slice(0, at).trim();
			let value = line.slice(at + 1).trim();

			// Strip one layer of matching quotes, which is what people paste
			// and what the shell would have removed anyway.
			if (value.length > 1 && value[0] === value.at(-1) && (value[0] === '"' || value[0] === "'")) {
				value = value.slice(1, -1);
			}
			env[key] = value;
		}

		return env;
	}

	async function save() {
		if (saving) return;
		saving = true;
		error = null;

		const input = {
			name: name.trim(),
			repository: repository.trim(),
			branch: branch.trim() || 'main',
			domain: domain.trim(),
			port: Number(port) || 3000,
			env: parseEnv(envText),
			deployKey
		};

		try {
			const saved = app
				? await appsClient.update(app.id, input)
				: await appsClient.create(serverId, input);
			onSaved(saved);
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
	title={app ? `Change ${app.name}` : 'Add an application'}
	description="Point us at a repository and we will build it and run it on this server."
	busy={saving}
	{onClose}
>
	<form
		id="app-form"
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

		<div class="grid gap-5 sm:grid-cols-2">
			<TextField
				label="Name"
				bind:value={name}
				required
				placeholder="checkout-api"
				hint="Just for you — how it appears in your list."
			/>
			<TextField
				label="Branch"
				bind:value={branch}
				placeholder="main"
				hint="Which branch to deploy."
			/>
		</div>

		<TextField
			label="Repository"
			bind:value={repository}
			required
			placeholder="https://github.com/you/your-project"
			hint="If it has a Dockerfile we use it; if not we work out how to build it. Private repositories need a key below."
		/>

		<TextArea
			label="Deploy key"
			bind:value={deployKey}
			rows={4}
			autocomplete="off"
			spellcheck={false}
			placeholder={'-----BEGIN OPENSSH PRIVATE KEY-----'}
			hint={app?.hasDeployKey
				? 'A key is already stored. Leave this empty to keep it, or paste a new one to replace it.'
				: 'Only for a private repository. Add a read-only deploy key on the repository and paste its private half here — it is encrypted before it is stored.'}
		/>

		<div class="grid gap-5 sm:grid-cols-[1fr_8rem]">
			<TextField
				label="Domain"
				bind:value={domain}
				placeholder="app.yourdomain.com"
				hint="Optional. Point its DNS at this server first, then we get a certificate automatically."
			/>
			<TextField
				label="Port"
				bind:value={port}
				inputmode="numeric"
				placeholder="3000"
				hint="What it listens on."
			/>
		</div>

		<TextArea
			label="Environment variables"
			bind:value={envText}
			rows={6}
			autocomplete="off"
			spellcheck={false}
			placeholder={'DATABASE_URL=postgresql://…\nSTRIPE_KEY=sk_live_…'}
			hint={app
				? 'One per line. These replace whatever is set now — they are encrypted, so we cannot show you the current ones.'
				: 'One per line, exactly as in a .env file. Encrypted before they are stored, and masked in the build log.'}
		/>
	</form>

	{#snippet footer()}
		<Button variant="ghost" onclick={onClose} disabled={saving}>Cancel</Button>
		<Button type="submit" form="app-form" disabled={saving}>
			{saving ? 'Saving…' : app ? 'Save changes' : 'Add it'}
		</Button>
	{/snippet}
</Sheet>
