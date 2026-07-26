<script lang="ts">
	import { Skeleton } from '$components/ui';

	interface Props {
		/** Roughly how many cards to stand in for. */
		count?: number;
	}

	let { count = 4 }: Props = $props();
</script>

<div class="space-y-3" aria-busy="true" aria-label="Loading">
	<Skeleton class="h-4 w-32" />
	<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
		{#each Array.from({ length: count }, (_, index) => index) as index (index)}
			<Skeleton shape="card" class="h-60" />
		{/each}
	</div>
</div>
