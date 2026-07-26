<script lang="ts">
	import { Skeleton } from '$components/ui';
</script>

<div
	class="border-border bg-surface divide-border/70 grid divide-y overflow-hidden rounded-card border sm:grid-cols-2 sm:divide-x xl:grid-cols-4"
	aria-busy="true"
	aria-label="Loading"
>
	{#each Array.from({ length: 4 }, (_, index) => index) as index (index)}
		<div class="space-y-4 p-6">
			<Skeleton class="h-4 w-32" />
			<div class="flex items-end justify-between gap-4">
				<div class="space-y-2">
					<Skeleton class="h-7 w-20" />
					<Skeleton class="h-3 w-28" />
				</div>
				<Skeleton class="h-8 w-24" />
			</div>
		</div>
	{/each}
</div>
