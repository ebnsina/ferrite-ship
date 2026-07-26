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
	// Generous, because the lowest reading in a series is usually a long flat
	// run. Pinned near the floor it draws a horizontal rule a few pixels above
	// the card's own bottom border, and the two together read as one doubled
	// border rather than as a chart.
	const PAD = 5;

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
		const span = max - min;
		const usable = HEIGHT - PAD * 2;

		return series.map((value, index) => ({
			x: (index / (series.length - 1)) * WIDTH,
			// A flat series is drawn down the middle. The old `|| 1` fallback
			// meant to do this but divided by one instead of centring, so every
			// value landed at the floor and a fleet whose size had not changed
			// all week drew a straight line along the bottom of its card.
			y: span === 0 ? HEIGHT / 2 : PAD + (1 - (value - min) / span) * usable
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
