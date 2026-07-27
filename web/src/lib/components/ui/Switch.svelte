<script lang="ts">
	import { cn } from '$utils/cn';

	interface Props {
		label: string;
		/** What turning this on actually causes. Shown under the label. */
		description?: string;
		checked?: boolean;
		disabled?: boolean;
		class?: string;
	}

	let {
		label,
		description,
		checked = $bindable(false),
		disabled = false,
		class: className
	}: Props = $props();

	// A real checkbox underneath rather than a div with role="switch": it is
	// already focusable, already announced, already toggles on space, and
	// already participates in a form. The styling is the only custom part.
	const generatedId = $props.id();
	const describedBy = $derived(description ? `${generatedId}-description` : undefined);
</script>

<div class={cn('flex items-start gap-3', className)}>
	<label class="relative mt-0.5 inline-flex shrink-0 cursor-pointer items-center">
		<input
			id={generatedId}
			type="checkbox"
			role="switch"
			bind:checked
			{disabled}
			aria-describedby={describedBy}
			aria-label={label}
			class="peer sr-only"
		/>
		<span
			aria-hidden="true"
			class={cn(
				'bg-surface-sunken border-border rounded-pill relative h-5 w-9 border transition-colors duration-150',
				'peer-checked:bg-accent-solid peer-checked:border-accent-solid',
				'peer-focus-visible:ring-accent-border peer-focus-visible:ring-2 peer-focus-visible:ring-offset-2',
				'peer-focus-visible:ring-offset-surface',
				disabled && 'opacity-50'
			)}
		>
			<span
				class="bg-surface absolute top-0.5 left-0.5 size-3.5 rounded-pill shadow-sm transition-transform duration-150 peer-checked:translate-x-4"
				style:transform={checked ? 'translateX(1rem)' : 'none'}
			></span>
		</span>
	</label>

	<label for={generatedId} class="min-w-0 cursor-pointer">
		<span class="text-content block text-sm">{label}</span>
		{#if description}
			<span id={describedBy} class="text-content-muted mt-0.5 block text-xs leading-relaxed">
				{description}
			</span>
		{/if}
	</label>
</div>
