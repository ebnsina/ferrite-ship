<script lang="ts">
	import { cn } from '$utils/cn';
	import LogoMark from './LogoMark.svelte';

	interface Props {
		href?: string;
		size?: 'sm' | 'md';
		class?: string;
	}

	let { href = '/', size = 'md', class: className }: Props = $props();

	const MARK_SIZE = { sm: 28, md: 32 } as const;
</script>

<!-- Mark only: the name is carried by the accessible label, not by set type. -->
<a {href} class={cn('inline-flex items-center', className)} aria-label="Ferrite Ship home">
	<LogoMark size={MARK_SIZE[size]} />
</a>
