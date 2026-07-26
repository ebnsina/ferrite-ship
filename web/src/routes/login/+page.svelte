<script lang="ts">
	import { goto } from '$app/navigation';
	import LogoMark from '$components/brand/LogoMark.svelte';
	import Seo from '$components/Seo.svelte';
	import { Button, Card, Skeleton, TextField, ThemeScope } from '$components/ui';
	import { authClient } from '$lib/data/auth';
	import { toAppError } from '$lib/errors';
	import CircleAlert from '@lucide/svelte/icons/circle-alert';

	let mode = $state<'loading' | 'setup' | 'login'>('loading');
	let email = $state('');
	let password = $state('');
	let submitting = $state(false);
	let error = $state<string | null>(null);

	async function check() {
		try {
			const status = await authClient.status();
			if (status.authenticated) {
				await goto('/dashboard');
				return;
			}
			mode = status.needsSetup ? 'setup' : 'login';
		} catch (cause) {
			error = toAppError(cause).message;
			mode = 'login';
		}
	}

	async function submit() {
		if (submitting) return;
		submitting = true;
		error = null;

		try {
			if (mode === 'setup') {
				await authClient.setup(email.trim(), password);
			} else {
				await authClient.login(email.trim(), password);
			}
			await goto('/dashboard');
		} catch (cause) {
			error = toAppError(cause).message;
			submitting = false;
		}
	}

	$effect(() => {
		void check();
	});
</script>

<Seo
	title={mode === 'setup' ? 'Create your account' : 'Sign in'}
	description="Sign in to Ferrite Ship."
	noindex
/>

<ThemeScope theme="light" class="flex min-h-dvh items-center justify-center px-5 py-16">
	<div class="w-full max-w-sm">
		<div class="flex flex-col items-center text-center">
			<LogoMark size={40} />
			<h1 class="text-content mt-5 text-xl font-semibold tracking-tight">
				{#if mode === 'setup'}
					Create your account
				{:else}
					Welcome back
				{/if}
			</h1>
			<p class="text-content-muted mt-1.5 text-sm">
				{#if mode === 'setup'}
					This is the first time this has run, so pick the account you will sign in with.
				{:else}
					Sign in to manage your servers.
				{/if}
			</p>
		</div>

		<Card class="mt-8">
			{#if mode === 'loading'}
				<div class="space-y-4">
					<Skeleton class="h-10" />
					<Skeleton class="h-10" />
					<Skeleton class="h-10" />
				</div>
			{:else}
				<form
					class="space-y-4"
					onsubmit={(event) => {
						event.preventDefault();
						void submit();
					}}
				>
					<TextField
						label="Email"
						type="email"
						bind:value={email}
						required
						autocomplete="username"
						placeholder="you@example.com"
					/>

					<TextField
						label="Password"
						type="password"
						bind:value={password}
						required
						autocomplete={mode === 'setup' ? 'new-password' : 'current-password'}
						hint={mode === 'setup'
							? 'At least 10 characters. Length matters more than symbols.'
							: undefined}
					/>

					{#if error}
						<p class="text-error flex gap-2 text-sm">
							<CircleAlert size={15} aria-hidden="true" class="mt-0.5 shrink-0" />
							{error}
						</p>
					{/if}

					<Button type="submit" class="w-full" disabled={submitting}>
						{#if submitting}
							Just a moment…
						{:else if mode === 'setup'}
							Create account
						{:else}
							Sign in
						{/if}
					</Button>
				</form>
			{/if}
		</Card>

		{#if mode === 'setup'}
			<p class="text-content-subtle mt-5 text-center text-xs leading-relaxed">
				Only one account can be created this way. Once it exists, this page becomes a sign-in form.
			</p>
		{/if}
	</div>
</ThemeScope>
