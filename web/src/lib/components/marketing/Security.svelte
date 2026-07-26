<script lang="ts">
	import { securityPoints } from '$lib/content/security';
	import SectionHeading from './SectionHeading.svelte';
</script>

<section id="security" class="bg-surface/40 border-border/70 scroll-mt-20 border-y">
	<div class="mx-auto max-w-6xl px-5 py-20">
		<SectionHeading
			eyebrow="Safety"
			title="You are handing us the keys, so here is how we hold them"
			description="Everything below is how the product is built, not a promise we are asking you to take on trust."
		/>

		<div class="mt-14 grid gap-5 sm:grid-cols-2">
			{#each securityPoints as point (point.id)}
				{@const Icon = point.icon}
				<article class="border-border bg-surface flex gap-4 rounded-card border p-6">
					<span
						class="bg-accent-soft text-accent flex size-10 shrink-0 items-center justify-center rounded-tile"
					>
						<Icon size={18} aria-hidden="true" />
					</span>
					<div>
						<h3 class="text-content text-base font-medium">{point.title}</h3>
						<p class="text-content-muted mt-1.5 text-sm leading-relaxed">{point.description}</p>
					</div>
				</article>
			{/each}
		</div>
	</div>
</section>
