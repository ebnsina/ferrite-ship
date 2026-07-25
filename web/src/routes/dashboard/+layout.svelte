<script lang="ts">
	import DashboardSidebar from '$components/dashboard/DashboardSidebar.svelte';
	import type { Snippet } from 'svelte';

	interface Props {
		children: Snippet;
	}

	let { children }: Props = $props();
</script>

<div class="flex min-h-dvh">
	<DashboardSidebar />
	<div class="min-w-0 flex-1">
		{@render children()}
	</div>
</div>
