<script lang="ts">
	import { serviceHighlights } from '$lib/content/services';
	import SectionHeading from './SectionHeading.svelte';
</script>

<section id="tools" class="bg-surface/40 border-border/70 scroll-mt-20 border-y">
	<div class="mx-auto max-w-6xl px-5 py-20">
		<SectionHeading
			eyebrow="One-click tools"
			title="The things you would otherwise install by hand"
			description="Pick one and it starts running — passwords already made for you, health checks already watching, backups already booked in."
		/>

		<ul class="mt-14 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
			{#each serviceHighlights as service (service.name)}
				{@const Icon = service.icon}
				<li
					class="border-border bg-surface hover:border-accent-border group flex items-center gap-3.5 rounded-tile border px-4 py-3.5 transition-colors duration-150"
				>
					<span
						class="bg-surface-sunken text-content-muted group-hover:bg-accent-soft group-hover:text-accent flex size-9 shrink-0 items-center justify-center rounded-tile transition-colors duration-150"
					>
						<Icon size={16} aria-hidden="true" />
					</span>
					<span class="min-w-0">
						<span class="text-content block truncate text-sm font-medium">{service.name}</span>
						<span class="text-content-subtle block truncate text-xs">{service.purpose}</span>
					</span>
				</li>
			{/each}
		</ul>

		<p class="text-content-muted mt-8 text-center text-sm">
			And plenty more — over a hundred, with new ones added regularly.
		</p>
	</div>
</section>
