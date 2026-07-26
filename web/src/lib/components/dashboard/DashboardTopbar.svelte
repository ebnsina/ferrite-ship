<script lang="ts">
	import { ButtonLink, SearchInput, ThemeToggle } from '$components/ui';
	import { dashboardTheme } from '$lib/theme/theme.svelte';
	import type { Crumb } from '$types/breadcrumb';
	import Bell from '@lucide/svelte/icons/bell';
	import Breadcrumb from './Breadcrumb.svelte';

	interface Props {
		crumbs: Crumb[];
		/** Number of unread notifications; the dot is hidden when zero. */
		unread?: number;
	}

	let { crumbs, unread = 0 }: Props = $props();
</script>

<header class="border-border/70 flex h-16 items-center gap-4 border-b px-6">
	<Breadcrumb items={crumbs} />

	<div class="ml-auto flex items-center gap-2">
		<div class="hidden sm:block">
			<SearchInput placeholder="Search" />
		</div>

		<button
			type="button"
			class="text-content-muted hover:bg-surface-raised hover:text-content relative inline-flex size-9 items-center justify-center rounded-tile transition-colors duration-150"
			aria-label={unread > 0 ? `Notifications, ${unread} unread` : 'Notifications'}
		>
			<Bell size={17} aria-hidden="true" />
			{#if unread > 0}
				<span
					class="bg-error border-canvas absolute top-1.5 right-1.5 size-2 rounded-pill border-2"
					aria-hidden="true"
				></span>
			{/if}
		</button>

		<ThemeToggle theme={dashboardTheme.value} onToggle={() => dashboardTheme.toggle()} />
	</div>
</header>
