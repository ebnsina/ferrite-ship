<script lang="ts">
	import DashboardTopbar from '$components/dashboard/DashboardTopbar.svelte';
	import SectionHeader from '$components/dashboard/SectionHeader.svelte';
	import Seo from '$components/Seo.svelte';
	import { Card, SearchInput } from '$components/ui';
	import { helpSections } from '$lib/content/help';

	let query = $state('');

	// Filters across both the question and the answer: people search using a
	// word from the problem they have ("locked out", "tunnel"), which is rarely
	// the word in the heading.
	const sections = $derived.by(() => {
		const needle = query.trim().toLowerCase();
		if (!needle) return helpSections;

		return helpSections
			.map((section) => ({
				...section,
				topics: section.topics.filter(
					(topic) =>
						topic.question.toLowerCase().includes(needle) ||
						topic.answer.toLowerCase().includes(needle)
				)
			}))
			.filter((section) => section.topics.length > 0);
	});
</script>

<Seo title="Help" description="How Ferrite Ship works, and what it does not do yet." noindex />

<DashboardTopbar crumbs={[{ label: 'Dashboard', href: '/dashboard' }, { label: 'Help' }]} />

<div class="space-y-6 px-6 py-8">
	<SectionHeader
		title="Help"
		description="How things work here — including the parts that are not built yet."
	/>

	<SearchInput bind:value={query} placeholder="Search for a word from your problem" />

	{#if sections.length === 0}
		<Card size="lg">
			<p class="text-content text-sm">Nothing here matches that.</p>
			<p class="text-content-muted mt-1 text-sm">
				Try a plainer word — "backup", "domain", "locked out".
			</p>
		</Card>
	{:else}
		<div class="space-y-8">
			{#each sections as section (section.heading)}
				<section class="space-y-3">
					<h2 class="text-content text-sm font-medium">{section.heading}</h2>

					<div class="border-border rounded-panel divide-border/70 divide-y overflow-hidden border">
						{#each section.topics as topic (topic.question)}
							<details>
								<summary
									class="text-content hover:bg-surface-sunken cursor-pointer list-none px-5 py-3.5 text-sm transition-colors duration-150"
								>
									{topic.question}
								</summary>
								<p class="text-content-muted px-5 pb-4 text-sm leading-relaxed">
									{topic.answer}
								</p>
							</details>
						{/each}
					</div>
				</section>
			{/each}
		</div>
	{/if}
</div>

<style>
	/* Safari draws its own disclosure triangle unless told otherwise, which
	   sits oddly in a list that has none. */
	summary::-webkit-details-marker {
		display: none;
	}
</style>
