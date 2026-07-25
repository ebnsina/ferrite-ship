<script lang="ts">
	import { cn } from '$utils/cn';

	interface Props {
		size?: number;
		class?: string;
	}

	let { size = 28, class: className }: Props = $props();
</script>

<!-- Stylised hull + rivet: "ferrite" (iron) and "ship". -->
<svg
	width={size}
	height={size}
	viewBox="0 0 32 32"
	fill="none"
	class={cn('shrink-0', className)}
	aria-hidden="true"
>
	<rect width="32" height="32" rx="9" fill="var(--ui-accent)" />
	<path
		d="M7 19.5h18l-2.4 4.2a2 2 0 0 1-1.74 1.01H11.14a2 2 0 0 1-1.74-1.01L7 19.5Z"
		fill="var(--ui-accent-content)"
	/>
	<path
		d="M10.5 16.8V9.4a1.6 1.6 0 0 1 1.6-1.6h5.1a1.6 1.6 0 0 1 1.13.47l3.2 3.2a1.6 1.6 0 0 1 .47 1.13v4.2"
		stroke="var(--ui-accent-content)"
		stroke-width="2.1"
		stroke-linecap="round"
		stroke-linejoin="round"
	/>
</svg>
