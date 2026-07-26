<script lang="ts">
	import { Skeleton } from '$components/ui';

	interface Props {
		columns?: number;
	}

	let { columns = 4 }: Props = $props();
</script>

<div class="grid gap-x-5 gap-y-8 md:grid-cols-2 xl:grid-cols-4" aria-busy="true" aria-label="Loading">
	{#each Array.from({ length: columns }, (_, index) => index) as index (index)}
		<div class="space-y-3">
			<Skeleton class="h-4 w-28" />
			<Skeleton shape="card" class="h-56" />
		</div>
	{/each}
</div>
