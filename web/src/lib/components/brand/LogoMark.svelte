<script lang="ts">
	import { cn } from '$utils/cn';

	interface Props {
		size?: number;
		class?: string;
	}

	let { size = 30, class: className }: Props = $props();
</script>

<!--
	The ship glyph is Lucide's "ship", drawn directly rather than imported so it
	can sit on the brand tile at a fixed optical size. Kept in step with
	static/favicon.svg and static/og.png.
-->
<svg
	width={size}
	height={size}
	viewBox="0 0 32 32"
	fill="none"
	class={cn('shrink-0', className)}
	aria-hidden="true"
>
	<rect width="32" height="32" rx="9" fill="var(--ui-accent)" />
	<g
		transform="translate(4 4) scale(1)"
		stroke="var(--ui-accent-content)"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
	>
		<path d="M12 10.189V14" />
		<path d="M12 2v3" />
		<path d="M19 13V7a2 2 0 0 0-2-2H7a2 2 0 0 0-2 2v6" />
		<path d="M19.38 20A11.6 11.6 0 0 0 21 14l-8.188-3.639a2 2 0 0 0-1.624 0L3 14a11.6 11.6 0 0 0 2.81 7.76" />
		<path d="M2 21c.6.5 1.2 1 2.5 1 2.5 0 2.5-2 5-2 1.3 0 1.9.5 2.5 1s1.2 1 2.5 1c2.5 0 2.5-2 5-2 1.3 0 1.9.5 2.5 1" />
	</g>
</svg>
