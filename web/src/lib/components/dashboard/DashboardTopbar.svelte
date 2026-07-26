<script lang="ts">
	import { SearchInput, ThemeToggle } from '$components/ui';
	import { dashboardTheme } from '$lib/theme/theme.svelte';
	import type { Crumb } from '$types/breadcrumb';
	import Breadcrumb from './Breadcrumb.svelte';

	/*
	 * The notifications bell is gone, along with the `unread={2}` that four
	 * pages passed it. There is no notifications feature, so the red dot was
	 * announcing two things that did not exist and the button opened nothing.
	 * It comes back when there is something real to announce.
	 */
	interface Props {
		crumbs: Crumb[];
	}

	let { crumbs }: Props = $props();
</script>

<header class="border-border/70 flex h-16 items-center gap-4 border-b px-6">
	<Breadcrumb items={crumbs} />

	<div class="ml-auto flex items-center gap-2">
		<div class="hidden sm:block">
			<SearchInput placeholder="Search" />
		</div>

		<ThemeToggle theme={dashboardTheme.value} onToggle={() => dashboardTheme.toggle()} />
	</div>
</header>
