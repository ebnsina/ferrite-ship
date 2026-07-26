/** How a metric's raw value should be rendered. */
export type MetricFormat = 'count' | 'percent' | 'bytes';

export interface FleetMetric {
	id: string;
	label: string;
	value: number;
	format: MetricFormat;
	/** Change versus the previous period, as a ratio: 0.15 is +15%. */
	deltaRatio: number;
	/** Whether a rise is good news — decides whether green means up or down. */
	higherIsBetter: boolean;
	/** Recent readings, oldest first. Drives the sparkline. */
	series: number[];
}
