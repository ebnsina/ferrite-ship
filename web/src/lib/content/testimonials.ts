export interface Testimonial {
	id: string;
	quote: string;
	name: string;
	role: string;
	/** Initials shown in the avatar circle. */
	initials: string;
}

/**
 * PLACEHOLDER TESTIMONIALS.
 *
 * These are written to size and style the section — they are NOT real quotes
 * from real people. Replace every entry with genuine, attributed feedback
 * before this page is published, or delete the section. Shipping invented
 * endorsements is both misleading and, in most jurisdictions, unlawful.
 */
export const testimonials: Testimonial[] = [
	{
		id: 't1',
		quote:
			'I used to set aside a whole evening for a new server. Now it is done before I have finished making tea.',
		name: 'Placeholder Name',
		role: 'Indie developer',
		initials: 'PN'
	},
	{
		id: 't2',
		quote:
			'We look after servers for eleven clients. Being able to see all of them on one screen changed how our week runs.',
		name: 'Placeholder Name',
		role: 'Agency owner',
		initials: 'PN'
	},
	{
		id: 't3',
		quote:
			'The streaming support is the part nobody else does. Everything else we tried stopped at ordinary websites.',
		name: 'Placeholder Name',
		role: 'Video platform founder',
		initials: 'PN'
	}
];
