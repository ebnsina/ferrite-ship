<script lang="ts">
	import { ButtonLink } from '$components/ui';
	import { ALERT_COPY } from '$lib/content/alerts';
	import { notificationsClient, type Alert } from '$lib/data/notifications';
	import { formatRelativeTime } from '$utils/format';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';

	interface Props {
		/** Names by server id, so a row can say "web-1" rather than "srv_a2c7". */
		names?: Record<string, string>;
	}

	let { names = {} }: Props = $props();

	let alerts = $state<Alert[]>([]);

	// Failing quietly is right here. This sits above the page's real content,
	// and an error box about the thing that reports errors would push what
	// somebody came for off the screen.
	async function load() {
		try {
			alerts = await notificationsClient.alerts();
		} catch {
			alerts = [];
		}
	}

	$effect(() => {
		void load();
	});
</script>

{#if alerts.length > 0}
	<section class="border-warn/40 bg-warn-soft rounded-card divide-warn/20 divide-y border">
		{#each alerts as alert (alert.id)}
			{@const copy = ALERT_COPY[alert.kind]}
			{@const server = names[alert.serverId] ?? 'A server'}
			<div class="flex flex-wrap items-start gap-3 p-4">
				<TriangleAlert size={17} aria-hidden="true" class="text-warn mt-0.5 shrink-0" />

				<div class="min-w-0 flex-1">
					<p class="text-content text-sm font-medium">
						{copy ? copy.title(server, alert.subject) : `Something needs attention on ${server}`}
					</p>
					{#if alert.detail}
						<p class="text-content-muted mt-0.5 text-sm">{alert.detail}</p>
					{/if}
					<p class="text-content-subtle mt-1 text-xs">
						Since {formatRelativeTime(alert.openedAt, { style: 'long' })}
					</p>
				</div>

				<ButtonLink
					href={copy
						? copy.href(alert.serverId, alert.subject)
						: `/dashboard/servers/${alert.serverId}`}
					variant="ghost"
					size="sm"
				>
					{copy?.action ?? 'Take a look'}
				</ButtonLink>
			</div>
		{/each}
	</section>
{/if}
