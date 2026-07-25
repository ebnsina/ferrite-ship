<script lang="ts">
	import { ButtonLink } from '$components/ui';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
</script>

<section class="mx-auto max-w-6xl px-5 pb-24">
	<div
		class="border-accent-border bg-accent-soft relative overflow-hidden rounded-panel border px-8 py-16 text-center"
	>
		<h2 class="text-content text-3xl font-semibold tracking-tight sm:text-4xl">
			Stop setting up servers by hand
		</h2>
		<p class="text-content-muted mx-auto mt-4 max-w-xl text-base">
			Connect one and have it ready to use before your coffee goes cold.
		</p>
		<div class="mt-8 flex justify-center">
			<ButtonLink href="/dashboard" size="lg">
				Open dashboard
				<ArrowRight size={17} aria-hidden="true" />
			</ButtonLink>
		</div>
	</div>
</section>
