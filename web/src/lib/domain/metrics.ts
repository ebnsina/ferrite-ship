import type { Tone } from '$components/ui/tone';
import type { FleetMetric } from '$types/metric';
import { formatBytes, formatNumber, formatPercent } from '$utils/format';

/** Renders a metric's raw value according to its declared format. */
export function formatMetricValue(metric: FleetMetric): string {
	switch (metric.format) {
		case 'percent':
			return formatPercent(metric.value, { fractionDigits: 1 });
		case 'bytes':
			return formatBytes(metric.value);
		case 'count':
			return formatNumber(metric.value);
	}
}

/** Sparkline colour follows whether the trend is going the way you want. */
export function trendTone(metric: FleetMetric): Tone {
	if (metric.deltaRatio === 0) return 'pending';
	return metric.deltaRatio > 0 === metric.higherIsBetter ? 'ok' : 'warn';
}
