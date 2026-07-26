<script lang="ts">
	import { cn } from '$utils/cn';
	import type { Tone } from './tone';

	interface Props {
		/** Readings oldest first. Fewer than two points renders nothing. */
		series: readonly number[];
		tone?: Tone;
		class?: string;
	}

	let { series, tone = 'ok', class: className }: Props = $props();

	// Gradient ids must be unique per instance or fills bleed between charts.
	const gradientId = $props.id();

	const WIDTH = 100;
	const HEIGHT = 32;
	const PAD = 2;

	const STROKE: Record<Tone, string> = {
		ok: 'var(--ui-ok)',
		warn: 'var(--ui-warn)',
		error: 'var(--ui-error)',
		info: 'var(--ui-info)',
		pending: 'var(--ui-content-subtle)'
	};

	const points = $derived.by(() => {
		if (series.length < 2) return [];

		const min = Math.min(...series);
		const max = Math.max(...series);
		// A flat series would divide by zero; draw it down the middle instead.
		const span = max - min || 1;
		const usable = HEIGHT - PAD * 2;

		return series.map((value, index) => ({
			x: (index / (series.length - 1)) * WIDTH,
			y: PAD + (1 - (value - min) / span) * usable
		}));
	});

	const linePath = $derived(
		points.map((point, index) => `${index === 0 ? 'M' : 'L'}${point.x} ${point.y}`).join(' ')
	);

	const areaPath = $derived(
		points.length === 0
			? ''
			: `${linePath} L${WIDTH} ${HEIGHT} L0 ${HEIGHT} Z`
	);
</script>

{#if points.length > 0}
	<svg
		viewBox="0 0 {WIDTH} {HEIGHT}"
		preserveAspectRatio="none"
		class={cn('h-8 w-full', className)}
		role="presentation"
		aria-hidden="true"
	>
		<defs>
			<linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
				<stop offset="0%" stop-color={STROKE[tone]} stop-opacity="0.28" />
				<stop offset="100%" stop-color={STROKE[tone]} stop-opacity="0" />
			</linearGradient>
		</defs>

		<path d={areaPath} fill="url(#{gradientId})" />
		<path
			d={linePath}
			fill="none"
			stroke={STROKE[tone]}
			stroke-width="1.75"
			stroke-linecap="round"
			stroke-linejoin="round"
			vector-effect="non-scaling-stroke"
		/>
	</svg>
{/if}
