<script lang="ts">
	import { testimonials } from '$lib/content/testimonials';
	import Quote from '@lucide/svelte/icons/quote';
	import SectionHeading from './SectionHeading.svelte';
</script>

<!--
	The quotes in $lib/content/testimonials.ts are PLACEHOLDERS, not real
	feedback. Replace them with genuine attributed quotes, or remove this
	section, before the page is published.
-->
<section id="testimonials" class="mx-auto max-w-6xl scroll-mt-20 px-5 py-20">
	<SectionHeading eyebrow="In their words" title="What people say" />

	<div class="mt-14 grid gap-5 md:grid-cols-3">
		{#each testimonials as testimonial (testimonial.id)}
			<figure class="border-border bg-surface flex flex-col rounded-card border p-7">
				<Quote size={20} aria-hidden="true" class="text-accent" />

				<blockquote class="text-content mt-5 flex-1 text-sm leading-relaxed">
					{testimonial.quote}
				</blockquote>

				<figcaption class="border-border/70 mt-6 flex items-center gap-3 border-t pt-5">
					<span
						class="bg-surface-sunken text-content-muted flex size-9 items-center justify-center rounded-pill text-xs font-medium"
						aria-hidden="true"
					>
						{testimonial.initials}
					</span>
					<span class="min-w-0">
						<span class="text-content block truncate text-sm font-medium">{testimonial.name}</span>
						<span class="text-content-subtle block truncate text-xs">{testimonial.role}</span>
					</span>
				</figcaption>
			</figure>
		{/each}
	</div>
</section>
