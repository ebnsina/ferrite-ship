<script lang="ts">
	import { ThemeToggle } from '$components/ui';
	import { dashboardTheme } from '$lib/theme/theme.svelte';
	import type { Crumb } from '$types/breadcrumb';
	import Breadcrumb from './Breadcrumb.svelte';

	/*
	 * The notifications bell is gone, along with the `unread={2}` that four
	 * pages passed it. There is no notifications feature, so the red dot was
	 * announcing two things that did not exist and the button opened nothing.
	 *
	 * The search box has gone the same way, and for the same reason: it had no
	 * handler at all. A box you can type into that never answers is worse than
	 * no box, because it reads as a search that found nothing. The servers list
	 * has a real one that filters as you type.
	 *
	 * Both come back when there is something behind them.
	 */
	interface Props {
		crumbs: Crumb[];
	}

	let { crumbs }: Props = $props();
</script>

<header class="border-border/70 flex h-16 items-center gap-4 border-b px-6">
	<Breadcrumb items={crumbs} />

	<div class="ml-auto flex items-center gap-2">
		<ThemeToggle theme={dashboardTheme.value} onToggle={() => dashboardTheme.toggle()} />
	</div>
</header>
