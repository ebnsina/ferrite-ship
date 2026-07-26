<script lang="ts">
	import { DeltaBadge, Sparkline } from '$components/ui';
	import { formatMetricValue, trendTone } from '$lib/domain/metrics';
	import type { FleetMetric } from '$types/metric';

	interface Props {
		metric: FleetMetric;
	}

	let { metric }: Props = $props();
</script>

<div class="flex flex-col gap-4 p-5">
	<p class="text-content truncate text-sm font-medium">{metric.label}</p>

	<div class="flex items-end justify-between gap-3">
		<div class="min-w-0">
			<!-- Value and delta must never break apart; long byte values would wrap. -->
			<div class="flex items-center gap-2 whitespace-nowrap">
				<span class="text-content text-2xl font-semibold tracking-tight" data-numeric>
					{formatMetricValue(metric)}
				</span>
				<DeltaBadge ratio={metric.deltaRatio} higherIsBetter={metric.higherIsBetter} />
			</div>
			<p class="text-content-subtle mt-1 truncate text-xs">compared to last week</p>
		</div>

		<div class="w-16 shrink-0 sm:w-20">
			<Sparkline series={metric.series} tone={trendTone(metric)} />
		</div>
	</div>
</div>
