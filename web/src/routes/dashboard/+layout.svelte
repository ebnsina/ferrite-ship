<script lang="ts">
	import { goto } from '$app/navigation';
	import DashboardSidebar from '$components/dashboard/DashboardSidebar.svelte';
	import { Skeleton, ThemeScope } from '$components/ui';
	import { authClient } from '$lib/data/auth';
	import { dashboardTheme } from '$lib/theme/theme.svelte';
	import type { Snippet } from 'svelte';

	interface Props {
		children: Snippet;
	}

	let { children }: Props = $props();

	// Checked once for the whole section rather than per page. Until the answer
	// arrives nothing is rendered, so a signed-out visitor never glimpses the
	// dashboard shell.
	let allowed = $state(false);

	$effect(() => {
		void (async () => {
			try {
				const status = await authClient.status();
				if (status.authenticated) {
					allowed = true;
					return;
				}
				await goto('/login');
			} catch {
				await goto('/login');
			}
		})();
	});
</script>

<!--
	The dashboard is scoped to its own theme (light by default) so the marketing
	site can stay dark. Tokens are declared against [data-theme], which matches
	any element, so this re-declares them for the subtree.
-->
<ThemeScope theme={dashboardTheme.value} class="flex min-h-dvh">
	{#if allowed}
		<DashboardSidebar />
		<div class="min-w-0 flex-1">
			{@render children()}
		</div>
	{:else}
		<div class="flex-1 space-y-4 p-6">
			<Skeleton class="h-8 w-48" />
			<Skeleton shape="card" class="h-64" />
		</div>
	{/if}
</ThemeScope>
