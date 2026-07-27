<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import Seo from '$components/Seo.svelte';
	import { Button, ButtonLink, Card, ErrorState, Meter, Skeleton } from '$components/ui';
	import { dashboardRepository } from '$lib/data';
	import { storageClient, type StorageReport } from '$lib/data/storage';
	import { toAppError, type AppError } from '$lib/errors';
	import { usageRatio } from '$types/server';
	import { formatBytes } from '$utils/format';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import Sparkle from '@lucide/svelte/icons/sparkle';

	const serverId = page.params.id ?? '';

	let report = $state<StorageReport | null>(null);
	let serverName = $state('Server');
	let loading = $state(true);
	let error = $state<AppError | null>(null);
	let starting = $state(false);

	// Everything reclaimable is ticked to begin with. Nothing in the list
	// destroys anything the owner put there, and someone who came here because
	// a bar went red wants the obvious answer selected, not a form to fill in.
	let chosen = $state<Set<string>>(new Set());

	async function load() {
		loading = true;
		error = null;

		try {
			const [result, server] = await Promise.all([
				storageClient.report(serverId),
				dashboardRepository.getServer(serverId)
			]);
			report = result;
			serverName = server.name;
			chosen = new Set(result.reclaimable.map((item) => item.id));
		} catch (cause) {
			error = toAppError(cause);
		} finally {
			loading = false;
		}
	}

	async function reclaim() {
		if (starting || chosen.size === 0) return;
		starting = true;

		try {
			const job = await storageClient.reclaim(serverId, [...chosen]);
			await goto(`/dashboard/jobs/${job.id}`);
		} catch (cause) {
			error = toAppError(cause);
			starting = false;
		}
	}

	function toggle(id: string) {
		const next = new Set(chosen);
		if (next.has(id)) next.delete(id);
		else next.add(id);
		chosen = next;
	}

	const totalChosen = $derived(
		(report?.reclaimable ?? [])
			.filter((item) => chosen.has(item.id))
			.reduce((sum, item) => sum + item.bytes, 0)
	);

	// The largest entry sets the scale, so the bars compare against each other
	// rather than against the disk — on a mostly empty disk everything would
	// otherwise be a stub.
	const largest = $derived(Math.max(1, ...(report?.directories ?? []).map((d) => d.bytes)));

	$effect(() => {
		void load();
	});
</script>

<Seo title="Storage" description="What is using this server's disk." noindex />

<DashboardTopbar
	crumbs={[
		{ label: 'Dashboard', href: '/dashboard' },
		{ label: 'Servers', href: '/dashboard/servers' },
		{ label: serverName, href: `/dashboard/servers/${serverId}` },
		{ label: 'Storage' }
	]}
/>

<div class="space-y-6 px-6 py-8">
	<div class="flex flex-wrap items-end justify-between gap-3">
		<div>
			<h1 class="text-content text-lg font-semibold tracking-tight">Storage</h1>
			<p class="text-content-muted mt-0.5 text-sm">What is using the disk, and what you can get back.</p>
		</div>
		<ButtonLink href="/dashboard/servers/{serverId}" variant="ghost" size="sm">
			<ArrowLeft size={15} aria-hidden="true" />
			Back to server
		</ButtonLink>
	</div>

	{#if loading}
		<p class="text-content-subtle text-sm">Looking through the disk — this takes a few seconds.</p>
		<Skeleton shape="card" class="h-40" />
		<Skeleton shape="card" class="h-64" />
	{:else if error}
		<Card size="lg" padded={false} class="overflow-hidden">
			<ErrorState {error} onRetry={load} />
		</Card>
	{:else if report}
		<Card size="lg">
			<Meter
				label="Disk"
				ratio={usageRatio({ usedBytes: report.usedBytes, totalBytes: report.totalBytes })}
				detail="{formatBytes(report.usedBytes)} of {formatBytes(report.totalBytes)}"
			/>
			<p class="text-content-muted mt-3 text-sm">
				{formatBytes(report.freeBytes)} free.
			</p>
		</Card>

		{#if report.reclaimable.length > 0}
			<Card size="lg" class="border-accent-border">
				<div class="flex flex-wrap items-start justify-between gap-4">
					<div>
						<h2 class="text-content flex items-center gap-2 text-sm font-medium">
							<Sparkle size={15} aria-hidden="true" class="text-accent" />
							You can free up to
							{formatBytes(report.reclaimable.reduce((sum, item) => sum + item.bytes, 0))}
						</h2>
						<p class="text-content-muted mt-1 max-w-xl text-sm leading-relaxed">
							None of this is anything you put on the server — it is what builds, updates and
							deployments leave behind. Expect somewhat less back than the total: images share
							layers with each other, so removing two that sit on the same base frees that base
							once.
						</p>
					</div>

					<Button onclick={reclaim} disabled={starting || chosen.size === 0}>
						{starting ? 'Starting…' : `Free ${formatBytes(totalChosen)}`}
					</Button>
				</div>

				<ul class="mt-5 space-y-2">
					{#each report.reclaimable as item (item.id)}
						<li>
							<label
								class="border-border rounded-card-sm hover:border-border-strong flex cursor-pointer items-start gap-3 border p-4 transition-colors duration-150"
							>
								<input
									type="checkbox"
									checked={chosen.has(item.id)}
									onchange={() => toggle(item.id)}
									class="mt-0.5"
								/>
								<span class="min-w-0 flex-1">
									<span class="flex flex-wrap items-baseline justify-between gap-2">
										<span class="text-content text-sm font-medium">{item.label}</span>
										<span class="text-content font-machine text-sm">{formatBytes(item.bytes)}</span>
									</span>
									<span class="text-content-muted mt-1 block text-xs leading-relaxed">
										{item.detail}
									</span>
								</span>
							</label>
						</li>
					{/each}
				</ul>
			</Card>
		{/if}

		<section class="space-y-3">
			<h2 class="text-content text-sm font-medium">Where the space has gone</h2>

			{#if report.directories.length === 0}
				<Card size="lg">
					<p class="text-content-muted text-sm">
						Nothing on this disk is large enough to be worth listing.
					</p>
				</Card>
			{:else}
				<Card size="lg" padded={false}>
					<ul class="divide-border/70 divide-y">
						{#each report.directories as directory (directory.path)}
							<li class="flex items-center gap-4 px-5 py-3">
								<span
									class="text-content font-machine min-w-0 flex-1 truncate text-sm"
									style:padding-left="{(directory.depth - 1) * 16}px"
									title={directory.path}
								>
									{directory.path}
								</span>

								<span class="bg-surface-sunken rounded-pill hidden h-1.5 w-40 overflow-hidden sm:block">
									<span
										class="bg-accent block h-full rounded-pill"
										style:width="{(directory.bytes / largest) * 100}%"
									></span>
								</span>

								<span class="text-content font-machine w-20 text-right text-sm">
									{formatBytes(directory.bytes)}
								</span>
							</li>
						{/each}
					</ul>
				</Card>

				<p class="text-content-subtle text-xs">
					Folders are shown two levels deep, largest first. A folder's size includes everything
					inside it, so <span class="font-machine">/var</span> and
					<span class="font-machine">/var/lib</span> overlap.
				</p>
			{/if}
		</section>
	{/if}
</div>
