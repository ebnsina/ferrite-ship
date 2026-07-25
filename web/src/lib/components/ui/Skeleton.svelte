<script lang="ts">
	import { cn } from '$utils/cn';

	interface Props {
		class?: string;
		/** Match the radius of whatever this is standing in for. */
		shape?: 'tile' | 'card' | 'pill';
	}

	let { class: className, shape = 'tile' }: Props = $props();

	const RADIUS = {
		tile: 'rounded-tile',
		card: 'rounded-card',
		pill: 'rounded-pill'
	} as const;
</script>

<div
	class={cn('bg-surface-raised animate-pulse', RADIUS[shape], className)}
	aria-hidden="true"
></div>
