<script lang="ts">
	import { EmptyState } from '$components/ui';
	import type { LucideProps } from '@lucide/svelte';
	import type { Component } from 'svelte';

	interface Props {
		title: string;
		description: string;
		icon: Component<LucideProps>;
	}

	let { title, description, icon }: Props = $props();
</script>

<div class="border-border bg-surface rounded-card border">
	<EmptyState {icon} {title} {description} />
</div>
