<script lang="ts">
	import { cn } from '$utils/cn';
	import type { LucideProps } from '@lucide/svelte';
	import type { Component } from 'svelte';

	interface Props {
		name: string;
		value: string;
		group: string;
		title: string;
		description: string;
		icon: Component<LucideProps>;
	}

	let { name, value, group = $bindable(), title, description, icon }: Props = $props();

	const Icon = $derived(icon);
	const selected = $derived(group === value);
	const inputId = $props.id();
</script>

<label
	for={inputId}
	class={cn(
		'flex cursor-pointer gap-3 rounded-card border p-4 transition-colors duration-150',
		selected
			? 'border-accent-border bg-accent-soft'
			: 'border-border bg-surface hover:border-border-strong'
	)}
>
	<input
		id={inputId}
		type="radio"
		{name}
		{value}
		bind:group
		class="sr-only"
	/>

	<span class={cn('mt-0.5 shrink-0', selected ? 'text-accent' : 'text-content-subtle')}>
		<Icon size={18} aria-hidden="true" />
	</span>

	<span class="min-w-0">
		<span class="text-content block text-sm font-medium">{title}</span>
		<span class="text-content-muted mt-0.5 block text-xs leading-relaxed">{description}</span>
	</span>
</label>
