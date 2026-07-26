<script lang="ts">
	import { Card, Skeleton } from '$components/ui';

	interface Props {
		/** Matches the catalogue's size, so the page does not jump when it loads. */
		count?: number;
	}

	let { count = 4 }: Props = $props();
</script>

<div class="space-y-4">
	{#each { length: count } as _, index (index)}
		<Card>
			<div class="flex items-start gap-4">
				<Skeleton class="size-10 shrink-0" />
				<div class="flex-1 space-y-2">
					<Skeleton class="h-4 w-32" />
					<Skeleton class="h-3 w-full max-w-md" />
					<Skeleton class="h-3 w-40" />
				</div>
				<Skeleton class="h-8 w-24 shrink-0" />
			</div>
		</Card>
	{/each}
</div>
