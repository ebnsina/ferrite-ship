<script lang="ts">
	import { Button, Card } from '$components/ui';
	import { githubClient, type GitHubStatus } from '$lib/data/github';
	import { toAppError } from '$lib/errors';
	import { formatRelativeTime } from '$utils/format';
	import Check from '@lucide/svelte/icons/check';
	// lucide dropped brand marks, so this is the generic one. Not a GitHub
	// logo, which we should not be drawing by hand anyway.
	import GitBranch from '@lucide/svelte/icons/git-branch';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';

	let status = $state<GitHubStatus | null>(null);
	let loading = $state(true);
	let working = $state(false);
	let error = $state<string | null>(null);

	async function load() {
		loading = true;
		try {
			status = await githubClient.status();
			error = null;
		} catch (cause) {
			error = toAppError(cause).message;
		} finally {
			loading = false;
		}
	}

	// The browser leaves this page for GitHub, so the link is followed here
	// rather than fetched — a redirect inside a fetch would be followed by the
	// fetch and land nowhere the person can see.
	async function connect() {
		working = true;
		error = null;

		try {
			const { url } = await githubClient.connect();
			window.location.href = url;
		} catch (cause) {
			error = toAppError(cause).message;
			working = false;
		}
	}

	async function disconnect(id: number) {
		working = true;
		error = null;

		try {
			await githubClient.disconnect(id);
			await load();
		} catch (cause) {
			error = toAppError(cause).message;
		} finally {
			working = false;
		}
	}

	$effect(() => {
		void load();
	});
</script>

<Card size="md">
	<div class="space-y-5">
		<div class="flex items-start gap-3">
			<span class="bg-surface-muted text-content rounded-tile p-2">
				<GitBranch class="size-5" aria-hidden="true" />
			</span>
			<div class="space-y-1">
				<h3 class="text-content text-sm font-medium">Deploying from GitHub</h3>
				<p class="text-content-muted max-w-2xl text-sm leading-relaxed">
					Connect GitHub and you can pick a repository from a list instead of pasting an
					address and a key. You choose which repositories we may see, and you can take that
					back from GitHub at any time.
				</p>
			</div>
		</div>

		{#if loading}
			<p class="text-content-subtle text-sm">Checking…</p>
		{:else if !status?.configured}
			<!-- Not a failure, and not something the person can fix from here.
			     Saying "connect" and then refusing would be worse than not
			     offering it. -->
			<p class="text-content-muted text-sm leading-relaxed">
				This installation is not set up to talk to GitHub, so deploying uses a git address and
				a deploy key. Whoever runs this can register a GitHub app to change that.
			</p>
		{:else if status.installations.length === 0}
			<Button onclick={connect} disabled={working}>
				<GitBranch size={15} aria-hidden="true" />
				{working ? 'Opening GitHub…' : 'Connect GitHub'}
			</Button>
		{:else}
			<div class="border-border divide-border rounded-card-sm divide-y border">
				{#each status.installations as installation (installation.id)}
					<div class="flex flex-wrap items-center gap-3 p-3">
						<Check class="text-status-ok size-4 shrink-0" aria-hidden="true" />
						<div class="min-w-0 flex-1">
							<p class="text-content text-sm font-medium">
								{installation.account || 'Connected'}
							</p>
							<p class="text-content-subtle text-xs">
								{installation.selection === 'all'
									? 'Every repository'
									: 'Selected repositories only'}
								· connected {formatRelativeTime(installation.connectedAt)}
							</p>
						</div>
						<Button
							variant="ghost"
							size="sm"
							disabled={working}
							onclick={() => disconnect(installation.id)}
						>
							Forget
						</Button>
					</div>
				{/each}
			</div>

			<div class="flex flex-wrap items-center gap-3">
				<Button variant="secondary" size="sm" onclick={connect} disabled={working}>
					Add another, or change repositories
				</Button>
				<p class="text-content-subtle text-xs leading-relaxed">
					Forgetting it here does not uninstall it from GitHub — that stays yours to do.
				</p>
			</div>
		{/if}

		{#if error}
			<p class="text-status-error flex items-start gap-2 text-sm">
				<TriangleAlert class="mt-0.5 size-4 shrink-0" aria-hidden="true" />
				<span>{error}</span>
			</p>
		{/if}
	</div>
</Card>
