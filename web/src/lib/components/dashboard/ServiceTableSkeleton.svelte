<script lang="ts">
	import { Card, Skeleton } from '$components/ui';

	interface Props {
		rows?: number;
	}

	let { rows = 12 }: Props = $props();

	// Service names vary a lot in length; a uniform placeholder reads as a
	// striped block rather than a list.
	const NAME_WIDTHS = ['w-32', 'w-24', 'w-44', 'w-28', 'w-36', 'w-20'];
	const DESC_WIDTHS = ['w-56', 'w-40', 'w-64', 'w-48'];
</script>

<Card padded={false} class="overflow-hidden">
	<div class="border-border/70 flex items-center gap-4 border-b px-5 py-2.5">
		<Skeleton class="h-3 w-16" />
		<Skeleton class="h-3 w-12" />
		<Skeleton class="ml-auto h-3 w-20" />
	</div>

	<div class="divide-border/70 divide-y" aria-busy="true" aria-label="Loading services">
		{#each Array.from({ length: rows }, (_, index) => index) as index (index)}
			<div class="flex items-center gap-4 px-5 py-3">
				<div class="space-y-1.5">
					<Skeleton class="h-3 {NAME_WIDTHS[index % NAME_WIDTHS.length]}" />
					<Skeleton class="h-2.5 {DESC_WIDTHS[index % DESC_WIDTHS.length]}" />
				</div>
				<Skeleton shape="pill" class="ml-auto h-5 w-20" />
				<Skeleton class="hidden h-3 w-28 lg:block" />
			</div>
		{/each}
	</div>
</Card>
