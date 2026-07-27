<script lang="ts">
	import { ButtonLink } from '$components/ui';
	import { PROBLEMS_COPY } from '$lib/content/problems';
	import type { ProblemFailure } from '$lib/data/problems';
	import { formatRelativeTime } from '$utils/format';
	import CircleX from '@lucide/svelte/icons/circle-x';

	interface Props {
		failure: ProblemFailure;
	}

	let { failure }: Props = $props();

	// Nobody was watching this one, which is the whole reason the page exists.
	const unattended = $derived(failure.actor === 'Scheduled');
</script>

<div class="flex flex-wrap items-start gap-3 p-4">
	<CircleX size={17} aria-hidden="true" class="text-error mt-0.5 shrink-0" />

	<div class="min-w-0 flex-1 space-y-1">
		<p class="text-content text-sm font-medium">
			{failure.title}
			<span class="text-content-muted font-normal">on {failure.serverName}</span>
		</p>

		<!-- The error itself, in full. Somebody on this page came looking for
		     exactly this sentence, so it is not truncated or tucked behind a
		     disclosure. font-machine because it is the server's words, not ours. -->
		{#if failure.error}
			<p class="text-content-muted font-machine text-xs leading-relaxed break-words">
				{failure.error}
			</p>
		{/if}

		<p class="text-content-subtle text-xs">
			{formatRelativeTime(failure.finishedAt)}
			·
			{#if unattended}
				<span class="text-content-muted font-medium">Scheduled</span>
			{:else}
				{failure.actor}
			{/if}
		</p>
	</div>

	<div class="flex shrink-0 flex-wrap gap-2">
		<ButtonLink href="/dashboard/jobs/{failure.jobId}" variant="ghost" size="sm">
			{PROBLEMS_COPY.openJob}
		</ButtonLink>
		<ButtonLink href="/dashboard/servers/{failure.serverId}" variant="ghost" size="sm">
			{PROBLEMS_COPY.openServer}
		</ButtonLink>
	</div>
</div>
