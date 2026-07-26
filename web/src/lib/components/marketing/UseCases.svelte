<script lang="ts">
	import { useCases } from '$lib/content/use-cases';
	import Check from '@lucide/svelte/icons/check';
	import SectionHeading from './SectionHeading.svelte';
</script>

<section id="use-cases" class="mx-auto max-w-6xl scroll-mt-20 px-5 py-20">
	<SectionHeading
		eyebrow="Who it is for"
		title="Built for people who would rather be building"
		description="Whatever you are running, the boring parts are the same — and they are the parts we take care of."
	/>

	<div class="mt-14 grid gap-5 md:grid-cols-2">
		{#each useCases as useCase (useCase.id)}
			{@const Icon = useCase.icon}
			<article class="border-border bg-surface rounded-card border p-7">
				<div class="flex items-center gap-3">
					<span class="bg-accent-soft text-accent flex size-10 items-center justify-center rounded-tile">
						<Icon size={18} aria-hidden="true" />
					</span>
					<h3 class="text-content text-base font-medium">{useCase.title}</h3>
				</div>

				<p class="text-content-muted mt-4 text-sm leading-relaxed">{useCase.description}</p>

				<ul class="mt-5 space-y-2">
					{#each useCase.points as point (point)}
						<li class="text-content-muted flex items-center gap-2.5 text-sm">
							<Check size={15} aria-hidden="true" class="text-accent shrink-0" />
							{point}
						</li>
					{/each}
				</ul>
			</article>
		{/each}
	</div>
</section>
