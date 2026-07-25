<script lang="ts">
	import { Card, Skeleton } from '$components/ui';

	interface Props {
		count?: number;
		class?: string;
	}

	let { count = 3, class: className = 'grid gap-4 md:grid-cols-2 xl:grid-cols-3' }: Props = $props();
</script>

<div class={className} aria-busy="true" aria-label="Loading">
	{#each Array.from({ length: count }, (_, index) => index) as index (index)}
		<Card>
			<div class="flex items-start justify-between gap-3">
				<div class="w-full space-y-2">
					<Skeleton class="h-4 w-1/3" />
					<Skeleton class="h-3 w-1/2" />
				</div>
				<Skeleton class="h-6 w-20" />
			</div>
			<div class="mt-8 space-y-3.5">
				<Skeleton class="h-3 w-full" />
				<Skeleton class="h-3 w-full" />
				<Skeleton class="h-3 w-2/3" />
			</div>
		</Card>
	{/each}
</div>
