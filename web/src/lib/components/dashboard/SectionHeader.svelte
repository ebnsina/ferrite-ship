<script lang="ts">
	interface Props {
		title: string;
		description?: string;
		linkHref?: string;
		linkLabel?: string;
	}

	let { title, description, linkHref, linkLabel = 'View all' }: Props = $props();
</script>

<div class="flex flex-wrap items-end justify-between gap-3">
	<div>
		<h2 class="text-content text-lg font-semibold tracking-tight">{title}</h2>
		{#if description}
			<p class="text-content-muted mt-1 text-sm">{description}</p>
		{/if}
	</div>

	{#if linkHref}
		<a
			href={linkHref}
			class="text-content hover:text-accent text-sm font-medium underline underline-offset-4 transition-colors duration-150"
		>
			{linkLabel}
		</a>
	{/if}
</div>
