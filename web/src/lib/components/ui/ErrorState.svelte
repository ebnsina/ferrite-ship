<script lang="ts">
	import type { AppError } from '$lib/errors';
	import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
	import Button from './Button.svelte';

	interface Props {
		error: AppError;
		/** Omit to hide the retry affordance for non-retryable failures. */
		onRetry?: () => void;
	}

	let { error, onRetry }: Props = $props();

	const canRetry = $derived(Boolean(onRetry) && error.retryable);
</script>

<div class="flex flex-col items-center px-6 py-16 text-center" role="alert">
	<div class="bg-error-soft text-error rounded-tile p-4">
		<TriangleAlert size={22} aria-hidden="true" />
	</div>

	<h3 class="text-content mt-5 text-base font-medium">{error.message}</h3>

	{#if error.action}
		<p class="text-content-muted mt-1.5 max-w-sm text-sm">{error.action}</p>
	{/if}

	{#if error.requestId}
		<p class="text-content-subtle font-machine mt-4 text-xs">
			Reference: {error.requestId}
		</p>
	{/if}

	{#if canRetry}
		<div class="mt-6">
			<Button variant="secondary" size="sm" onclick={onRetry}>Try again</Button>
		</div>
	{/if}
</div>
