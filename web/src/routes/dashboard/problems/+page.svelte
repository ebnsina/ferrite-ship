<script lang="ts">
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import FailureRow from '$components/dashboard/FailureRow.svelte';
	import ResourceView from '$components/dashboard/ResourceView.svelte';
	import SectionHeader from '$components/dashboard/SectionHeader.svelte';
	import Seo from '$components/Seo.svelte';
	import { ButtonLink, Card, Skeleton } from '$components/ui';
	import { ALERT_COPY } from '$lib/content/alerts';
	import { PROBLEMS_COPY } from '$lib/content/problems';
	import { problemsClient } from '$lib/data/problems';
	import { formatRelativeTime } from '$utils/format';
	import { createResource } from '$utils/resource.svelte';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';

	// One request for both halves. Loading them separately would let the page
	// render half an answer, and a page whose job is to report problems must
	// never show fewer than there are.
	const problems = createResource((signal) => problemsClient.list(signal));
</script>

<Seo title="What needs attention" description="Anything currently wrong, and anything that failed." noindex />

<DashboardTopbar
	crumbs={[{ label: 'Dashboard', href: '/dashboard' }, { label: 'What needs attention' }]}
/>

<div class="space-y-6 px-6 py-8">
	<SectionHeader title={PROBLEMS_COPY.title} description={PROBLEMS_COPY.description} />

	<ResourceView resource={problems}>
		{#snippet pending()}
			<Skeleton shape="card" class="h-64" />
		{/snippet}

		{#snippet children(found)}
			{@const quiet = found.alerts.length === 0 && found.failures.length === 0}

			{#if quiet}
				<!-- Said plainly, with an icon that reads as an answer. A blank
				     space here would be taken for a page that failed to load by
				     exactly the person least able to tell the difference. -->
				<Card size="lg">
					<div class="flex flex-col items-center gap-3 py-10 text-center">
						<CircleCheck size={28} aria-hidden="true" class="text-ok" />
						<h2 class="text-content text-base font-medium">{PROBLEMS_COPY.allClearTitle}</h2>
						<p class="text-content-muted max-w-md text-sm leading-relaxed">
							{PROBLEMS_COPY.allClearBody}
						</p>
					</div>
				</Card>
			{:else}
				<section class="space-y-3">
					<div>
						<h2 class="text-content text-sm font-medium">{PROBLEMS_COPY.alertsTitle}</h2>
						<p class="text-content-muted text-sm">{PROBLEMS_COPY.alertsHint}</p>
					</div>

					{#if found.alerts.length === 0}
						<p class="text-content-subtle text-sm">{PROBLEMS_COPY.noAlerts}</p>
					{:else}
						<div class="border-warn/40 bg-warn-soft rounded-card divide-warn/20 divide-y border">
							{#each found.alerts as alert (alert.id)}
								{@const copy = ALERT_COPY[alert.kind]}
								<div class="flex flex-wrap items-start gap-3 p-4">
									<TriangleAlert size={17} aria-hidden="true" class="text-warn mt-0.5 shrink-0" />
									<div class="min-w-0 flex-1 space-y-1">
										<p class="text-content text-sm font-medium">
											{copy.title(alert.serverName, alert.subject)}
										</p>
										{#if alert.detail}
											<p class="text-content-muted text-xs leading-relaxed break-words">
												{alert.detail}
											</p>
										{/if}
										<p class="text-content-subtle text-xs">
											Since {formatRelativeTime(alert.openedAt)}
										</p>
									</div>
									<ButtonLink
										href={copy.href(alert.serverId, alert.subject)}
										variant="ghost"
										size="sm"
										class="shrink-0"
									>
										{copy.action}
									</ButtonLink>
								</div>
							{/each}
						</div>
					{/if}
				</section>

				<section class="space-y-3">
					<div>
						<h2 class="text-content text-sm font-medium">{PROBLEMS_COPY.failuresTitle}</h2>
						<p class="text-content-muted text-sm">{PROBLEMS_COPY.failuresHint}</p>
					</div>

					{#if found.failures.length === 0}
						<p class="text-content-subtle text-sm">{PROBLEMS_COPY.noFailures}</p>
					{:else}
						<div class="border-border bg-surface rounded-card divide-border divide-y border">
							{#each found.failures as failure (failure.jobId)}
								<FailureRow {failure} />
							{/each}
						</div>
						<p class="text-content-subtle text-sm leading-relaxed">
							{PROBLEMS_COPY.scheduledNote}
						</p>
					{/if}
				</section>
			{/if}
		{/snippet}
	</ResourceView>
</div>
