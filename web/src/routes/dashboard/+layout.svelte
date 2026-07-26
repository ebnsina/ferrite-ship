<script lang="ts">
	import DashboardSidebar from '$components/dashboard/DashboardSidebar.svelte';
	import { ThemeScope } from '$components/ui';
	import { dashboardTheme } from '$lib/theme/theme.svelte';
	import type { Snippet } from 'svelte';

	interface Props {
		children: Snippet;
	}

	let { children }: Props = $props();
</script>

<!--
	The dashboard is scoped to its own theme (light by default) so the marketing
	site can stay dark. Tokens are declared against [data-theme], which matches
	any element, so this re-declares them for the subtree.
-->
<ThemeScope theme={dashboardTheme.value} class="flex min-h-dvh">
	<DashboardSidebar />
	<div class="min-w-0 flex-1">
		{@render children()}
	</div>
</ThemeScope>
