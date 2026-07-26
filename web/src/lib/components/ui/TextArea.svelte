<script lang="ts">
	import { cn } from '$utils/cn';
	import type { HTMLTextareaAttributes } from 'svelte/elements';

	interface Props extends Omit<HTMLTextareaAttributes, 'class' | 'value'> {
		label: string;
		value?: string;
		hint?: string;
		class?: string;
	}

	let { label, value = $bindable(''), hint, class: className, id, ...rest }: Props = $props();

	// $props.id() must initialise a variable directly, hence the two steps.
	const generatedId = $props.id();
	const fieldId = $derived(id ?? generatedId);
</script>

<div class={cn('space-y-1.5', className)}>
	<label for={fieldId} class="text-content block text-sm font-medium">{label}</label>

	<textarea
		id={fieldId}
		bind:value
		aria-describedby={hint ? `${fieldId}-hint` : undefined}
		class="border-border bg-surface-sunken text-content placeholder:text-content-subtle font-machine focus-visible:border-accent-border w-full rounded-field border px-3.5 py-2.5 text-xs outline-none transition-colors duration-150"
		{...rest}
	></textarea>

	{#if hint}
		<p id="{fieldId}-hint" class="text-content-subtle text-xs">{hint}</p>
	{/if}
</div>
