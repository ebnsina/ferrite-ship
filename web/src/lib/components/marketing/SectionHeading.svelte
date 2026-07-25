<script lang="ts">
	interface Props {
		eyebrow: string;
		title: string;
		description?: string;
	}

	let { eyebrow, title, description }: Props = $props();
</script>

<div class="mx-auto max-w-2xl text-center">
	<p class="text-accent text-xs font-medium tracking-[0.12em] uppercase">{eyebrow}</p>
	<h2 class="text-content mt-3 text-3xl font-semibold tracking-tight sm:text-4xl">{title}</h2>
	{#if description}
		<p class="text-content-muted mt-4 text-base">{description}</p>
	{/if}
</div>
