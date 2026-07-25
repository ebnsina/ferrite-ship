import { cachedFormatter, type LocaleOption } from './intl-cache';

/**
 * Durations via Intl.NumberFormat unit styles joined by Intl.ListFormat.
 * Intl.DurationFormat would be terser but is not yet available everywhere we
 * need to run, and this composition is fully localised either way.
 */

const SEGMENTS = [
	{ unit: 'day', ms: 86_400_000 },
	{ unit: 'hour', ms: 3_600_000 },
	{ unit: 'minute', ms: 60_000 },
	{ unit: 'second', ms: 1000 }
] as const;

type DurationStyle = 'narrow' | 'short' | 'long';

function unitPart(
	value: number,
	unit: Intl.NumberFormatOptions['unit'],
	unitDisplay: DurationStyle,
	locale: LocaleOption
): string {
	const options: Intl.NumberFormatOptions = {
		style: 'unit',
		unit,
		unitDisplay,
		maximumFractionDigits: 0
	};
	return cachedFormatter(
		'duration-part',
		locale,
		options,
		() => new Intl.NumberFormat(locale, options)
	).format(value);
}

/**
 * 5_405_000 → "1 hr, 30 min" (maxParts 2). Used for job runtimes and uptime.
 */
export function formatDuration(
	milliseconds: number,
	{ maxParts = 2, style = 'short' }: { maxParts?: number; style?: DurationStyle } = {},
	locale?: LocaleOption
): string {
	if (!Number.isFinite(milliseconds) || milliseconds < 0) {
		throw new TypeError(`formatDuration expected a non-negative number, received ${milliseconds}`);
	}

	let remaining = Math.round(milliseconds);
	const parts: string[] = [];

	for (const segment of SEGMENTS) {
		if (parts.length >= maxParts) break;

		const amount = Math.floor(remaining / segment.ms);
		remaining -= amount * segment.ms;

		// Skip leading zero segments, but keep interior ones (1 hr, 0 min).
		if (amount === 0 && parts.length === 0) continue;
		parts.push(unitPart(amount, segment.unit, style, locale));
	}

	if (parts.length === 0) {
		return unitPart(0, 'second', style, locale);
	}

	const listOptions: Intl.ListFormatOptions = { style: 'narrow', type: 'unit' };
	return cachedFormatter(
		'duration-list',
		locale,
		listOptions,
		() => new Intl.ListFormat(locale, listOptions)
	).format(parts);
}
