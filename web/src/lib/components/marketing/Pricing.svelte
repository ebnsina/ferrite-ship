<script lang="ts">
	import { ButtonLink } from '$components/ui';
	import { pricingCurrency, pricingPlans } from '$lib/content/pricing';
	import { cn } from '$utils/cn';
	import { formatNumber } from '$utils/format';
	import Check from '@lucide/svelte/icons/check';
	import SectionHeading from './SectionHeading.svelte';

	function priceLabel(monthlyPrice: number | null): string {
		if (monthlyPrice === null) return "Let's talk";
		if (monthlyPrice === 0) return 'Free';
		return formatNumber(monthlyPrice, {
			style: 'currency',
			currency: pricingCurrency,
			maximumFractionDigits: 0
		});
	}
</script>

<section id="pricing" class="mx-auto max-w-6xl scroll-mt-20 px-5 py-20">
	<SectionHeading
		eyebrow="Pricing"
		title="Pay for the servers you actually have"
		description="You bring your own server, so you keep paying your host directly. This is only for the panel that looks after it."
	/>

	<div class="mt-14 grid gap-5 lg:grid-cols-3">
		{#each pricingPlans as plan (plan.id)}
			<article
				class={cn(
					'flex flex-col rounded-card border p-7',
					plan.featured
						? 'border-accent-border bg-accent-soft'
						: 'border-border bg-surface'
				)}
			>
				<div class="flex items-center justify-between gap-3">
					<h3 class="text-content text-base font-medium">{plan.name}</h3>
					{#if plan.featured}
						<span class="bg-accent-solid text-accent-content rounded-pill px-2.5 py-1 text-xs font-medium">
							Most popular
						</span>
					{/if}
				</div>

				<p class="text-content-muted mt-1.5 text-sm">{plan.tagline}</p>

				<p class="mt-6 flex items-baseline gap-1.5">
					<span class="text-content text-4xl font-semibold tracking-tight" data-numeric>
						{priceLabel(plan.monthlyPrice)}
					</span>
					{#if plan.monthlyPrice !== null && plan.monthlyPrice > 0}
						<span class="text-content-subtle text-sm">a month</span>
					{/if}
				</p>

				<ul class="mt-7 flex-1 space-y-2.5">
					{#each plan.features as feature (feature)}
						<li class="text-content-muted flex items-start gap-2.5 text-sm">
							<Check size={15} aria-hidden="true" class="text-accent mt-0.5 shrink-0" />
							{feature}
						</li>
					{/each}
				</ul>

				<div class="mt-8">
					<ButtonLink
						href="/dashboard"
						variant={plan.featured ? 'primary' : 'secondary'}
						class="w-full"
					>
						{plan.ctaLabel}
					</ButtonLink>
				</div>
			</article>
		{/each}
	</div>

	<p class="text-content-subtle mt-8 text-center text-sm">
		Every paid plan starts with a 14-day trial. No card needed to begin.
	</p>
</section>
