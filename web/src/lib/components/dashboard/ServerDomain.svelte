<script lang="ts">
	import { Button, Card, TextField } from '$components/ui';
	import { DOMAIN_COPY } from '$lib/content/domains';
	import { domainsClient } from '$lib/data/domains';
	import { toAppError } from '$lib/errors';
	import type { ManagedServer } from '$types/server';
	import Check from '@lucide/svelte/icons/check';
	import Globe from '@lucide/svelte/icons/globe';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';

	interface Props {
		server: ManagedServer;
		/** Lets the page hold one copy of the server rather than two. */
		onSaved: (server: ManagedServer) => void;
	}

	let { server, onSaved }: Props = $props();

	let domain = $state('');
	let email = $state('');
	let saving = $state(false);
	let error = $state<string | null>(null);

	// Seeded from the server rather than captured once, so that saving — which
	// hands a new server back up to the page — leaves the fields showing what
	// was actually stored, and so that opening a different server does not show
	// the previous one's domain. Only the saved values are read here, so typing
	// does not re-run it.
	$effect(() => {
		domain = server.domain;
		email = server.acmeEmail;
	});

	// The saved value, not the typed one: the confirmation below has to describe
	// what is actually stored, or it congratulates you on an unsaved edit.
	const routed = $derived(server.domain !== '');
	const changed = $derived(domain !== server.domain || email !== server.acmeEmail);

	async function save(next: { domain: string; email: string }) {
		saving = true;
		error = null;

		try {
			// The fields follow from the page's copy of the server, via the
			// effect above, so there is nothing to assign here.
			onSaved(await domainsClient.save(server.id, next));
		} catch (cause) {
			error = toAppError(cause).message;
		} finally {
			saving = false;
		}
	}
</script>

<Card size="md">
	<div class="space-y-5">
		<div class="flex items-start gap-3">
			<span class="bg-surface-muted text-accent rounded-tile p-2">
				<Globe class="size-5" aria-hidden="true" />
			</span>
			<div class="space-y-1">
				<h3 class="text-content text-sm font-medium">{DOMAIN_COPY.title}</h3>
				<p class="text-content-muted max-w-2xl text-sm leading-relaxed">{DOMAIN_COPY.intro}</p>
			</div>
		</div>

		<div class="border-border bg-surface-muted rounded-card-sm border p-4">
			<h4 class="text-content text-sm font-medium">{DOMAIN_COPY.dnsTitle}</h4>
			<ol class="text-content-muted mt-2 list-decimal space-y-1 pl-5 text-sm leading-relaxed">
				{#each DOMAIN_COPY.dnsSteps(server.ipAddress) as step (step)}
					<li>{step}</li>
				{/each}
			</ol>
		</div>

		<div class="grid gap-4 sm:grid-cols-2">
			<TextField
				label={DOMAIN_COPY.domainLabel}
				hint={DOMAIN_COPY.domainHint}
				bind:value={domain}
				placeholder="example.com"
				autocomplete="off"
				spellcheck={false}
			/>
			<TextField
				label={DOMAIN_COPY.emailLabel}
				hint={DOMAIN_COPY.emailHint}
				bind:value={email}
				type="email"
				placeholder="you@example.com"
				autocomplete="off"
			/>
		</div>

		{#if error}
			<p class="text-status-error flex items-start gap-2 text-sm">
				<TriangleAlert class="mt-0.5 size-4 shrink-0" aria-hidden="true" />
				<span>{error}</span>
			</p>
		{:else if routed}
			<p class="text-content-muted flex items-start gap-2 text-sm">
				<Check class="text-status-ok mt-0.5 size-4 shrink-0" aria-hidden="true" />
				<span>{DOMAIN_COPY.live(server.domain)}</span>
			</p>
		{/if}

		<p class="text-content-subtle text-sm leading-relaxed">{DOMAIN_COPY.note}</p>

		<div class="flex flex-wrap gap-3">
			<Button
				onclick={() => save({ domain, email })}
				disabled={saving || !changed}
			>
				{saving ? DOMAIN_COPY.saving : DOMAIN_COPY.save}
			</Button>

			{#if routed}
				<Button
					variant="ghost"
					disabled={saving}
					onclick={() => save({ domain: '', email: '' })}
				>
					{DOMAIN_COPY.clear}
				</Button>
			{/if}
		</div>
	</div>
</Card>
