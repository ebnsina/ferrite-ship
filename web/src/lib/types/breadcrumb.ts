export interface Crumb {
	label: string;
	/** Omit on the final crumb — the page you are already on is not a link. */
	href?: string;
}
