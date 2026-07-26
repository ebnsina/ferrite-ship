<script lang="ts">
	import { stats } from '$lib/content/stats';
</script>

<section class="border-border/70 border-y">
	<dl class="divide-border/70 mx-auto grid max-w-6xl grid-cols-2 divide-y sm:divide-y-0 lg:grid-cols-4 lg:divide-x">
		{#each stats as stat (stat.label)}
			<div class="px-5 py-8 text-center">
				<dt class="sr-only">{stat.label}</dt>
				<dd>
					<span class="text-accent block text-3xl font-semibold tracking-tight" data-numeric>
						{stat.value}
					</span>
					<span class="text-content-muted mt-1.5 block text-sm">{stat.label}</span>
				</dd>
			</div>
		{/each}
	</dl>
</section>
