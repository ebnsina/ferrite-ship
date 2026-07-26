<script lang="ts">
	import { Card, Skeleton } from '$components/ui';

	interface Props {
		rows?: number;
	}

	let { rows = 10 }: Props = $props();

	// Varying the widths stops the placeholder reading as a striped block and
	// makes it resemble the file names it stands in for.
	const NAME_WIDTHS = ['w-40', 'w-28', 'w-56', 'w-36', 'w-24', 'w-48', 'w-32', 'w-44'];
</script>

<Card padded={false} class="overflow-hidden">
	<div class="border-border/70 flex items-center gap-4 border-b px-5 py-2.5">
		<Skeleton class="h-3 w-16" />
		<div class="ml-auto hidden gap-8 sm:flex">
			<Skeleton class="h-3 w-10" />
			<Skeleton class="h-3 w-14" />
			<Skeleton class="h-3 w-20" />
		</div>
	</div>

	<div class="divide-border/70 divide-y" aria-busy="true" aria-label="Loading files">
		{#each Array.from({ length: rows }, (_, index) => index) as index (index)}
			<div class="flex items-center gap-4 px-5 py-3">
				<Skeleton class="size-4 shrink-0" />
				<Skeleton class="h-3 {NAME_WIDTHS[index % NAME_WIDTHS.length]}" />
				<div class="ml-auto hidden items-center gap-8 sm:flex">
					<Skeleton class="h-3 w-10" />
					<Skeleton class="h-3 w-14" />
					<Skeleton class="h-3 w-20" />
				</div>
			</div>
		{/each}
	</div>
</Card>
