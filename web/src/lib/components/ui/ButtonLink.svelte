<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLAnchorAttributes } from 'svelte/elements';
	import { buttonClasses, type ButtonSize, type ButtonVariant } from './button-styles';

	interface Props extends Omit<HTMLAnchorAttributes, 'class'> {
		href: string;
		variant?: ButtonVariant;
		size?: ButtonSize;
		class?: string;
		children: Snippet;
	}

	let {
		href,
		variant = 'primary',
		size = 'md',
		class: className,
		children,
		...rest
	}: Props = $props();
</script>

<a {href} class={buttonClasses(variant, size, className)} {...rest}>
	{@render children()}
</a>
