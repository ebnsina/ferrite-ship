export interface CrumbLink {
	label: string;
	href: string;
}

export interface Crumb {
	label: string;
	/** Omit on the final crumb — the page you are already on is not a link. */
	href?: string;
	/**
	 * Places you can jump to from here.
	 *
	 * A breadcrumb normally only goes back up. These make it go sideways too:
	 * from a server's files straight to its terminal, or from one server to
	 * another, without walking back through the list first. The trail is where
	 * this belongs because it is already describing where you are.
	 */
	siblings?: CrumbLink[];
}
