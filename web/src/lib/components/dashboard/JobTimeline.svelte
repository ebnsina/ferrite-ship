<script lang="ts">
	import { Card, Skeleton } from '$components/ui';
	import type { JobStream } from '$lib/jobs/job-stream.svelte';
	import JobStep from './JobStep.svelte';

	interface Props {
		stream: JobStream;
	}

	let { stream }: Props = $props();

	let container = $state<HTMLDivElement | null>(null);

	// Follow the tail while the job runs, so the newest step stays in view.
	$effect(() => {
		const count = stream.steps.length;
		if (stream.state !== 'streaming' || count === 0 || !container) return;
		container.scrollIntoView({ block: 'end', behavior: 'smooth' });
	});
</script>

<Card size="lg" padded={false} class="overflow-hidden">
	{#if stream.steps.length === 0}
		<div class="space-y-3 p-5">
			<Skeleton class="h-5 w-2/3" />
			<Skeleton class="h-5 w-1/2" />
			<Skeleton class="h-5 w-3/5" />
		</div>
	{:else}
		{@const live = stream.state !== 'done' && stream.state !== 'error'}
		<div bind:this={container}>
			{#each stream.steps as step (step.id)}
				<JobStep
					{step}
					{live}
					defaultOpen={step.outcome === 'failed' || (step.outcome === null && live)}
				/>
			{/each}
		</div>
	{/if}
</Card>
