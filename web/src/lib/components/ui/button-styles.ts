import { cn } from '$utils/cn';

export type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger';
export type ButtonSize = 'sm' | 'md' | 'lg';

const BASE =
	'inline-flex items-center justify-center gap-2 rounded-control font-medium whitespace-nowrap ' +
	'transition-[background-color,border-color,color,opacity] duration-150 ease-snappy ' +
	'disabled:pointer-events-none disabled:opacity-45 aria-disabled:pointer-events-none aria-disabled:opacity-45';

const VARIANTS: Record<ButtonVariant, string> = {
	primary: 'bg-accent text-accent-content hover:bg-accent-hover',
	secondary:
		'border border-border bg-surface-raised text-content hover:border-border-strong hover:bg-surface',
	ghost: 'text-content-muted hover:bg-surface-raised hover:text-content',
	danger: 'bg-error text-white hover:opacity-90'
};

const SIZES: Record<ButtonSize, string> = {
	sm: 'h-8 gap-1.5 px-3.5 text-[0.8125rem]',
	md: 'h-10 px-5 text-sm',
	lg: 'h-12 px-7 text-base'
};

export function buttonClasses(
	variant: ButtonVariant,
	size: ButtonSize,
	extra?: string | undefined
): string {
	return cn(BASE, VARIANTS[variant], SIZES[size], extra);
}
