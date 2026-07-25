import { cachedFormatter, type LocaleOption } from './intl-cache';

function toDate(value: Date | string | number): Date {
	const date = value instanceof Date ? value : new Date(value);
	if (Number.isNaN(date.getTime())) {
		throw new TypeError(`Expected a valid date, received ${String(value)}`);
	}
	return date;
}

function dateTimeFormatter(
	locale: LocaleOption,
	options: Intl.DateTimeFormatOptions
): Intl.DateTimeFormat {
	return cachedFormatter(
		'datetime',
		locale,
		options,
		() => new Intl.DateTimeFormat(locale, options)
	);
}

export function formatDate(value: Date | string | number, locale?: LocaleOption): string {
	return dateTimeFormatter(locale, { dateStyle: 'medium' }).format(toDate(value));
}

export function formatDateTime(value: Date | string | number, locale?: LocaleOption): string {
	return dateTimeFormatter(locale, { dateStyle: 'medium', timeStyle: 'short' }).format(
		toDate(value)
	);
}

/** Precise wall-clock time for log lines and job timelines. */
export function formatTimestamp(value: Date | string | number, locale?: LocaleOption): string {
	return dateTimeFormatter(locale, {
		hour: '2-digit',
		minute: '2-digit',
		second: '2-digit',
		hour12: false
	}).format(toDate(value));
}

const DIVISIONS: readonly { limit: number; unit: Intl.RelativeTimeFormatUnit }[] = [
	{ limit: 60, unit: 'second' },
	{ limit: 60, unit: 'minute' },
	{ limit: 24, unit: 'hour' },
	{ limit: 7, unit: 'day' },
	{ limit: 4.34524, unit: 'week' },
	{ limit: 12, unit: 'month' },
	{ limit: Number.POSITIVE_INFINITY, unit: 'year' }
];

/** "3 minutes ago", "in 2 days". Picks the largest sensible unit. */
export function formatRelativeTime(
	value: Date | string | number,
	{ now = Date.now() }: { now?: number } = {},
	locale?: LocaleOption
): string {
	const options: Intl.RelativeTimeFormatOptions = { numeric: 'auto', style: 'long' };
	const formatter = cachedFormatter(
		'relativetime',
		locale,
		options,
		() => new Intl.RelativeTimeFormat(locale, options)
	);

	let delta = (toDate(value).getTime() - now) / 1000;

	for (const { limit, unit } of DIVISIONS) {
		if (Math.abs(delta) < limit) {
			return formatter.format(Math.round(delta), unit);
		}
		delta /= limit;
	}

	// DIVISIONS ends at Infinity, so this is unreachable in practice.
	return formatter.format(Math.round(delta), 'year');
}
