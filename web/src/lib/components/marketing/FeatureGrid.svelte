<script lang="ts">
	import { features } from '$lib/content/features';
	import FeatureCard from './FeatureCard.svelte';
	import SectionHeading from './SectionHeading.svelte';
</script>

<section id="features" class="mx-auto max-w-6xl scroll-mt-20 px-5 py-20">
	<SectionHeading
		eyebrow="What you get"
		title="More than somewhere to put your apps"
		description="Most tools stop once your app is running. We keep going — the files, the updates, the storage, the whole machine stays yours to manage."
	/>

	<div class="mt-14 grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
		{#each features as feature (feature.id)}
			<FeatureCard {feature} />
		{/each}
	</div>
</section>
