<script lang="ts">
	import { cn } from '$utils/cn';
	import type { HTMLInputAttributes } from 'svelte/elements';

	interface Props extends Omit<HTMLInputAttributes, 'class' | 'value'> {
		label: string;
		value?: string;
		hint?: string;
		error?: string;
		class?: string;
	}

	let {
		label,
		value = $bindable(''),
		hint,
		error,
		class: className,
		id,
		...rest
	}: Props = $props();

	// $props.id() must initialise a variable directly, hence the two steps.
	const generatedId = $props.id();
	const fieldId = $derived(id ?? generatedId);
	const describedBy = $derived(error ? `${fieldId}-error` : hint ? `${fieldId}-hint` : undefined);
</script>

<div class={cn('space-y-1.5', className)}>
	<label for={fieldId} class="text-content block text-sm font-medium">{label}</label>

	<input
		id={fieldId}
		bind:value
		aria-invalid={error ? 'true' : undefined}
		aria-describedby={describedBy}
		class={cn(
			'border-border bg-surface-sunken text-content placeholder:text-content-subtle h-10 w-full rounded-field border px-3.5 text-sm outline-none transition-colors duration-150',
			'focus-visible:border-accent-border',
			error && 'border-error'
		)}
		{...rest}
	/>

	{#if error}
		<p id="{fieldId}-error" class="text-error text-xs">{error}</p>
	{:else if hint}
		<p id="{fieldId}-hint" class="text-content-subtle text-xs">{hint}</p>
	{/if}
</div>
