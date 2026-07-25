<script lang="ts" generics="T">
	import { ErrorState } from '$components/ui';
	import type { Resource } from '$utils/resource.svelte';
	import type { Snippet } from 'svelte';

	interface Props {
		resource: Resource<T>;
		/** Rendered while the first (or a retried) load is in flight. */
		pending: Snippet;
		children: Snippet<[T]>;
	}

	let { resource, pending, children }: Props = $props();
</script>

{#if resource.error}
	<ErrorState error={resource.error} onRetry={() => resource.reload()} />
{:else if resource.loading}
	{@render pending()}
{:else if resource.data !== null}
	{@render children(resource.data)}
{/if}
