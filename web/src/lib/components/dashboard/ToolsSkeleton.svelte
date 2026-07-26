<script lang="ts">
	import { Card, Skeleton } from '$components/ui';

	interface Props {
		/** Matches the catalogue's size, so the page does not jump when it loads. */
		count?: number;
	}

	let { count = 4 }: Props = $props();
</script>

<div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
	{#each { length: count } as _, index (index)}
		<Card>
			<div class="flex items-start gap-3">
				<Skeleton class="size-10 shrink-0" />
				<div class="flex-1 space-y-2">
					<Skeleton class="h-4 w-28" />
					<Skeleton class="h-3 w-32" />
				</div>
			</div>
			<div class="mt-3 space-y-2">
				<Skeleton class="h-3 w-full" />
				<Skeleton class="h-3 w-3/4" />
			</div>
			<Skeleton class="mt-5 h-8 w-24" />
		</Card>
	{/each}
</div>
