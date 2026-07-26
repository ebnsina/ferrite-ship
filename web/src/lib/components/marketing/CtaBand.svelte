<script lang="ts">
	import { ButtonLink } from '$components/ui';
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import CircleCheck from '@lucide/svelte/icons/circle-check';

	const assurances = ['Free to start', 'No card needed', 'Remove it any time'];
</script>

<section class="mx-auto max-w-6xl px-5 py-24">
	<div class="border-border bg-surface relative overflow-hidden rounded-panel border">
		<!-- Accent glow rather than a flat fill: reads as a considered surface. -->
		<div
			class="pointer-events-none absolute inset-x-0 -top-32 h-72 opacity-25 blur-3xl"
			style="background: radial-gradient(50% 60% at 50% 50%, var(--ui-accent), transparent 70%)"
			aria-hidden="true"
		></div>

		<div class="relative px-8 py-16 text-center sm:px-12 sm:py-20">
			<h2 class="text-content mx-auto max-w-2xl text-3xl font-semibold tracking-tight sm:text-4xl">
				Your next server could be ready in two minutes
			</h2>

			<p class="text-content-muted mx-auto mt-4 max-w-xl text-base">
				Connect a machine you already have and watch it get set up properly, without reading a
				single guide.
			</p>

			<div class="mt-9 flex flex-wrap items-center justify-center gap-3">
				<ButtonLink href="/dashboard" size="lg">
					Get started
					<ArrowRight size={17} aria-hidden="true" />
				</ButtonLink>
				<ButtonLink href="#pricing" variant="secondary" size="lg">See pricing</ButtonLink>
			</div>

			<ul class="mt-8 flex flex-wrap items-center justify-center gap-x-6 gap-y-2">
				{#each assurances as assurance (assurance)}
					<li class="text-content-subtle flex items-center gap-1.5 text-xs">
						<CircleCheck size={14} aria-hidden="true" class="text-accent" />
						{assurance}
					</li>
				{/each}
			</ul>
		</div>
	</div>
</section>
