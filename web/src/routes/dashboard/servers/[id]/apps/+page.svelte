<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import AppSheet from '$components/dashboard/AppSheet.svelte';
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import Seo from '$components/Seo.svelte';
	import {
		Button,
		ButtonLink,
		Card,
		ConfirmDialog,
		EmptyState,
		ErrorState,
		Menu,
		Skeleton,
		StatusPill,
		type MenuItem
	} from '$components/ui';
	import { dashboardRepository } from '$lib/data';
	import { appsClient, type App } from '$lib/data/apps';
	import { appStatus } from '$lib/domain/tools';
	import { toAppError, type AppError } from '$lib/errors';
	import { formatRelativeTime } from '$utils/format';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import Boxes from '@lucide/svelte/icons/boxes';
	import ExternalLink from '@lucide/svelte/icons/external-link';
	import Pencil from '@lucide/svelte/icons/pencil';
	import Rocket from '@lucide/svelte/icons/rocket';
	import Trash2 from '@lucide/svelte/icons/trash-2';

	const serverId = page.params.id ?? '';
	const server = dashboardRepository;

	let apps = $state<App[]>([]);
	let serverName = $state('Server');
	let loading = $state(true);
	let error = $state<AppError | null>(null);
	let actionError = $state<string | null>(null);

	let editing = $state<App | null>(null);
	let sheetOpen = $state(false);
	let removing = $state<App | null>(null);
	let removingBusy = $state(false);

	async function load() {
		loading = true;
		error = null;

		try {
			const [list, detail] = await Promise.all([
				appsClient.list(serverId),
				server.getServer(serverId)
			]);
			apps = list;
			serverName = detail.name;
		} catch (cause) {
			error = toAppError(cause);
		} finally {
			loading = false;
		}
	}

	async function deploy(app: App) {
		actionError = null;
		try {
			const job = await appsClient.deploy(app.id);
			await goto(`/dashboard/jobs/${job.id}`);
		} catch (cause) {
			actionError = toAppError(cause).message;
		}
	}

	async function remove() {
		if (!removing) return;
		removingBusy = true;

		try {
			const job = await appsClient.remove(removing.id);
			await goto(`/dashboard/jobs/${job.id}`);
		} catch (cause) {
			actionError = toAppError(cause).message;
			removingBusy = false;
			removing = null;
		}
	}

	function actionsFor(app: App): MenuItem[] {
		return [
			{
				label: app.status === 'new' ? 'Deploy it' : 'Deploy again',
				icon: Rocket,
				onSelect: () => void deploy(app)
			},
			{
				label: 'Change settings',
				icon: Pencil,
				onSelect: () => {
					editing = app;
					sheetOpen = true;
				}
			},
			{
				label: 'Remove from this server',
				icon: Trash2,
				danger: true,
				separated: true,
				onSelect: () => (removing = app)
			}
		];
	}

	$effect(() => {
		void load();
	});
</script>

<Seo title="Applications" description="What you have deployed to this server." noindex />

<DashboardTopbar
	crumbs={[
		{ label: 'Dashboard', href: '/dashboard' },
		{ label: 'Servers', href: '/dashboard/servers' },
		{ label: serverName, href: `/dashboard/servers/${serverId}` },
		{ label: 'Applications' }
	]}
/>

<div class="space-y-5 px-6 py-8">
	<div class="flex flex-wrap items-end justify-between gap-3">
		<div>
			<h1 class="text-content text-lg font-semibold tracking-tight">Applications</h1>
			<p class="text-content-muted mt-0.5 text-sm">
				Your own code, built from a repository and running here.
			</p>
		</div>

		<div class="flex items-center gap-2">
			<ButtonLink href="/dashboard/servers/{serverId}" variant="ghost" size="sm">
				<ArrowLeft size={15} aria-hidden="true" />
				Back to server
			</ButtonLink>
			<Button
				size="sm"
				onclick={() => {
					editing = null;
					sheetOpen = true;
				}}
			>
				Add an application
			</Button>
		</div>
	</div>

	{#if actionError}
		<Card size="sm" class="border-error/40 bg-error-soft">
			<p class="text-error text-sm">{actionError}</p>
		</Card>
	{/if}

	{#if loading}
		<Skeleton shape="card" class="h-64" />
	{:else if error}
		<Card size="lg" padded={false} class="overflow-hidden">
			<ErrorState {error} onRetry={load} />
		</Card>
	{:else if apps.length === 0}
		<Card size="lg" padded={false} class="overflow-hidden">
			<EmptyState
				icon={Boxes}
				title="Nothing deployed here yet"
				description="Point this at a repository and we will build it and keep it running, next to the databases it needs."
			>
				{#snippet action()}
					<Button
						size="sm"
						onclick={() => {
							editing = null;
							sheetOpen = true;
						}}
					>
						Add your first application
					</Button>
				{/snippet}
			</EmptyState>
		</Card>
	{:else}
		<div class="grid gap-4 md:grid-cols-2">
			{#each apps as app (app.id)}
				{@const status = appStatus(app.status)}
				<Card class="flex h-full flex-col">
					<div class="flex items-start justify-between gap-3">
						<div class="min-w-0">
							<h2 class="text-content text-sm font-medium">{app.name}</h2>
							<p class="text-content-subtle font-machine mt-1 truncate text-xs" title={app.repository}>
								{app.repository}
								<span class="text-content-muted">· {app.branch}</span>
							</p>
						</div>
						<StatusPill {status} />
					</div>

					<div class="mt-4 flex-1 space-y-1.5 text-xs">
						{#if app.domain}
							<a
								href="https://{app.domain}"
								target="_blank"
								rel="noreferrer noopener"
								class="text-accent inline-flex items-center gap-1.5 hover:underline"
							>
								{app.domain}
								<ExternalLink size={12} aria-hidden="true" />
							</a>
						{:else}
							<p class="text-content-subtle">
								No domain yet, so only this server can reach it.
							</p>
						{/if}

						<p class="text-content-subtle">
							{#if app.deployedAt}
								Last deployed {formatRelativeTime(app.deployedAt, { style: 'short' })}
							{:else}
								Never deployed
							{/if}
							· listens on {app.port}
						</p>
					</div>

					<div class="border-border/70 mt-4 flex items-center justify-between gap-2 border-t pt-4">
						{#if app.lastJobId}
							<ButtonLink href="/dashboard/jobs/{app.lastJobId}" variant="ghost" size="sm">
								See the last run
							</ButtonLink>
						{:else}
							<span></span>
						{/if}
						<Menu label="Actions" items={actionsFor(app)} />
					</div>
				</Card>
			{/each}
		</div>
	{/if}
</div>

<AppSheet
	bind:open={sheetOpen}
	{serverId}
	app={editing}
	onClose={() => (sheetOpen = false)}
	onSaved={(saved) => {
		sheetOpen = false;
		// Straight into deploying a brand new app: adding one and then having
		// to find the menu to start it is a step that serves nobody.
		if (!editing) void deploy(saved);
		else void load();
	}}
/>

{#if removing}
	<ConfirmDialog
		open={true}
		title="Remove {removing.name}?"
		description="This stops it, deletes its build and takes its domain off the server."
		confirmLabel="Remove it"
		danger
		busy={removingBusy}
		onConfirm={remove}
		onCancel={() => (removing = null)}
	>
		<p class="text-content-muted text-xs leading-relaxed">
			Your repository is untouched — this only removes what is running here. Deploying it again
			rebuilds from the same branch.
		</p>
	</ConfirmDialog>
{/if}
