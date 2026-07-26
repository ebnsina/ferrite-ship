/**
 * Facts about how the product works — not adoption numbers. Claims like
 * "2,400 servers managed" would be invented until there is real data behind
 * them, and a landing page that opens with a fabricated number is worth less
 * than one that opens with a true one.
 */
export interface Stat {
	value: string;
	label: string;
}

export const stats: Stat[] = [
	{ value: '1', label: 'command to get started' },
	{ value: '2 min', label: 'from bare server to secured' },
	{ value: '100+', label: 'tools ready to install' },
	{ value: '0', label: 'new ports opened to the internet' }
];
