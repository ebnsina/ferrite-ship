import { cn } from '$utils/cn';

export type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger';
export type ButtonSize = 'sm' | 'md' | 'lg';

const BASE =
	'inline-flex items-center justify-center gap-2 font-medium whitespace-nowrap ' +
	'transition-[background-color,border-color,color,opacity] duration-150 ease-snappy ' +
	'disabled:pointer-events-none disabled:opacity-45 aria-disabled:pointer-events-none aria-disabled:opacity-45';

const VARIANTS: Record<ButtonVariant, string> = {
	primary: 'bg-accent-solid text-accent-content hover:bg-accent-hover',
	secondary:
		'border border-border bg-surface-raised text-content hover:border-border-strong hover:bg-surface',
	ghost: 'text-content-muted hover:bg-surface-raised hover:text-content',
	danger: 'bg-error text-white hover:opacity-90'
};

/*
 * Roomier than the label strictly needs, and each size gets its own corner.
 *
 * The horizontal padding is well over half the height everywhere: a generous
 * radius eats into the corners, and without the extra room a short label like
 * "Copy" ends up sitting inside the curve rather than against a straight edge.
 *
 * The radius is not constant, because the same number does not read the same
 * way at two heights. 1rem on a 44px button is a soft rectangle; on a 36px one
 * it is most of the way to a pill and the shape stops looking deliberate. So
 * the small size steps down to keep the *impression* consistent, which is what
 * the eye is actually comparing.
 */
const SIZES: Record<ButtonSize, string> = {
	sm: 'h-9 gap-1.5 rounded-[0.75rem] px-4 text-[0.8125rem]',
	md: 'h-11 rounded-control px-6 text-sm',
	lg: 'h-13 rounded-control px-8 text-base'
};

export function buttonClasses(
	variant: ButtonVariant,
	size: ButtonSize,
	extra?: string | undefined
): string {
	return cn(BASE, VARIANTS[variant], SIZES[size], extra);
}
