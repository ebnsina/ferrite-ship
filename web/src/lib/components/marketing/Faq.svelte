<script lang="ts">
	import { faqEntries } from '$lib/content/faq';
	import Plus from '@lucide/svelte/icons/plus';
	import SectionHeading from './SectionHeading.svelte';
</script>

<section id="faq" class="bg-surface/40 border-border/70 scroll-mt-20 border-y">
	<div class="mx-auto max-w-3xl px-5 py-20">
		<SectionHeading eyebrow="Questions" title="Things people ask before signing up" />

		<!-- Native <details> keeps this keyboard- and screen-reader-friendly with no JS. -->
		<div class="border-border bg-surface divide-border/70 mt-14 divide-y overflow-hidden rounded-card border">
			{#each faqEntries as entry (entry.question)}
				<details class="group">
					<summary
						class="text-content hover:bg-surface-sunken flex cursor-pointer items-center justify-between gap-4 px-6 py-5 text-sm font-medium transition-colors duration-150 [&::-webkit-details-marker]:hidden"
					>
						{entry.question}
						<Plus
							size={17}
							aria-hidden="true"
							class="text-content-subtle shrink-0 transition-transform duration-200 group-open:rotate-45"
						/>
					</summary>
					<p class="text-content-muted px-6 pb-5 text-sm leading-relaxed">{entry.answer}</p>
				</details>
			{/each}
		</div>
	</div>
</section>
