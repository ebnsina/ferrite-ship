import { cachedFormatter, type LocaleOption } from './intl-cache';

/**
 * Decimal units (1000-based), because that is what Intl's `kilobyte`,
 * `megabyte` … units actually mean. Using them with 1024-based maths would
 * render a value that contradicts its own label.
 */
const STEP = 1000;

const UNITS = [
	'byte',
	'kilobyte',
	'megabyte',
	'gigabyte',
	'terabyte',
	'petabyte'
] as const satisfies readonly Intl.NumberFormatOptions['unit'][];

export function formatBytes(
	bytes: number,
	{ fractionDigits }: { fractionDigits?: number } = {},
	locale?: LocaleOption
): string {
	if (!Number.isFinite(bytes)) {
		throw new TypeError(`formatBytes expected a finite number, received ${bytes}`);
	}

	const magnitude = Math.abs(bytes);
	const exponent =
		magnitude < 1 ? 0 : Math.min(Math.floor(Math.log(magnitude) / Math.log(STEP)), UNITS.length - 1);

	const unit = UNITS[exponent] ?? 'byte';
	const scaled = bytes / STEP ** exponent;

	// Bytes are whole things; anything larger reads better with one decimal.
	const digits = fractionDigits ?? (exponent === 0 ? 0 : 1);

	const options: Intl.NumberFormatOptions = {
		style: 'unit',
		unit,
		unitDisplay: 'short',
		minimumFractionDigits: 0,
		maximumFractionDigits: digits
	};

	return cachedFormatter(
		'bytes',
		locale,
		options,
		() => new Intl.NumberFormat(locale, options)
	).format(scaled);
}
