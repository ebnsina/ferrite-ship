/**
 * The product's entire status vocabulary. Servers, jobs, backups and checks all
 * resolve to one of these five tones so a colour never means two things.
 * Colour is always paired with an icon or text — never used alone.
 */
export type Tone = 'ok' | 'warn' | 'error' | 'info' | 'pending';

interface ToneClasses {
	/** Solid fill, for dots and bar fills. */
	fill: string;
	text: string;
	/** Tinted background for badges and callouts. */
	soft: string;
}

export const TONE_CLASSES: Record<Tone, ToneClasses> = {
	ok: { fill: 'bg-ok', text: 'text-ok', soft: 'bg-ok-soft text-ok' },
	warn: { fill: 'bg-warn', text: 'text-warn', soft: 'bg-warn-soft text-warn' },
	error: { fill: 'bg-error', text: 'text-error', soft: 'bg-error-soft text-error' },
	info: { fill: 'bg-info', text: 'text-info', soft: 'bg-info-soft text-info' },
	pending: {
		fill: 'bg-pending',
		text: 'text-content-muted',
		soft: 'bg-pending-soft text-content-muted'
	}
};

/** Thresholds shared by every resource meter so "red" means the same everywhere. */
export function toneForUsage(ratio: number): Tone {
	if (ratio >= 0.9) return 'error';
	if (ratio >= 0.75) return 'warn';
	return 'ok';
}
