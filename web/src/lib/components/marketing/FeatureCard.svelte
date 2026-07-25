<script lang="ts">
	import { Card } from '$components/ui';
	import type { Feature } from '$lib/content/features';

	interface Props {
		feature: Feature;
	}

	let { feature }: Props = $props();

	const Icon = $derived(feature.icon);
</script>

<Card class="h-full">
	<div class="bg-accent-soft text-accent inline-flex rounded-tile p-3">
		<Icon size={19} aria-hidden="true" />
	</div>
	<h3 class="text-content mt-5 text-base font-medium">{feature.title}</h3>
	<p class="text-content-muted mt-2 text-sm leading-relaxed">{feature.description}</p>
</Card>
