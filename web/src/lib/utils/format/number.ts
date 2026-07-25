import { cachedFormatter, type LocaleOption } from './intl-cache';

export function formatNumber(
	value: number,
	options: Intl.NumberFormatOptions = {},
	locale?: LocaleOption
): string {
	return cachedFormatter(
		'number',
		locale,
		options,
		() => new Intl.NumberFormat(locale, options)
	).format(value);
}

/** 12_400 → "12.4K". For counts in tight spaces such as stat tiles. */
export function formatCompactNumber(value: number, locale?: LocaleOption): string {
	return formatNumber(value, { notation: 'compact', maximumFractionDigits: 1 }, locale);
}

/** Takes a ratio, not a percentage: 0.734 → "73%". */
export function formatPercent(
	ratio: number,
	{ fractionDigits = 0 }: { fractionDigits?: number } = {},
	locale?: LocaleOption
): string {
	return formatNumber(
		ratio,
		{
			style: 'percent',
			minimumFractionDigits: fractionDigits,
			maximumFractionDigits: fractionDigits
		},
		locale
	);
}
