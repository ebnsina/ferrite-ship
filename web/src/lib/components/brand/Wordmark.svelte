<script lang="ts">
	import { cn } from '$utils/cn';
	import LogoMark from './LogoMark.svelte';

	interface Props {
		href?: string;
		class?: string;
	}

	let { href = '/', class: className }: Props = $props();
</script>

<a {href} class={cn('inline-flex items-center gap-2.5', className)}>
	<LogoMark size={26} />
	<span class="text-content text-[0.95rem] font-semibold tracking-tight">
		ferrite<span class="text-content-subtle">·</span>ship
	</span>
</a>
