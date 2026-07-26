<script lang="ts">
	import { cn } from '$utils/cn';
	import LogoMark from './LogoMark.svelte';

	interface Props {
		href?: string;
		size?: 'sm' | 'md';
		class?: string;
	}

	let { href = '/', size = 'md', class: className }: Props = $props();

	const SIZES = {
		sm: { mark: 26, text: 'text-[0.95rem]' },
		md: { mark: 30, text: 'text-base' }
	} as const;
</script>

<a {href} class={cn('inline-flex items-center gap-2.5', className)}>
	<LogoMark size={SIZES[size].mark} />
	<span class={cn('text-content font-semibold tracking-tight', SIZES[size].text)}>Ferrite Ship</span>
</a>
