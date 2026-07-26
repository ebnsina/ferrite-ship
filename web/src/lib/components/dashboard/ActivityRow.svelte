<script lang="ts">
	import { TONE_CLASSES } from '$components/ui/tone';
	import { ACTIVITY_STATUS } from '$lib/domain/status';
	import type { ActivityEntry } from '$types/activity';
	import { formatDuration, formatRelativeTime } from '$utils/format';
	import Server from '@lucide/svelte/icons/server';
	import User from '@lucide/svelte/icons/user';

	interface Props {
		entry: ActivityEntry;
	}

	let { entry }: Props = $props();

	const status = $derived(ACTIVITY_STATUS[entry.status]);
	const Icon = $derived(status.icon);
</script>

<li class="flex items-start gap-3.5 py-3.5">
	<span class="mt-0.5 shrink-0 {TONE_CLASSES[status.tone].text}">
		<Icon size={17} aria-hidden="true" class={status.animated ? 'animate-spin' : undefined} />
		<span class="sr-only">{status.label}</span>
	</span>

	<div class="min-w-0 flex-1">
		<a
			href="/dashboard/jobs/{entry.id}"
			class="text-content hover:text-accent block truncate text-sm transition-colors duration-150"
		>
			{entry.title}
		</a>
		<p class="text-content-subtle mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
			<span class="flex items-center gap-1.5">
				<Server size={12} aria-hidden="true" />
				{entry.serverName}
			</span>
			<span class="flex items-center gap-1.5">
				<User size={12} aria-hidden="true" />
				{entry.actor}
			</span>
		</p>
	</div>

	<div class="shrink-0 text-right">
		<p class="{TONE_CLASSES[status.tone].text} text-xs font-medium">{status.label}</p>
		<p class="text-content-subtle mt-1 text-xs" data-numeric>
			{formatRelativeTime(entry.startedAt)}{entry.durationMs !== null
				? ` · took ${formatDuration(entry.durationMs, { maxParts: 1 })}`
				: ''}
		</p>
	</div>
</li>
