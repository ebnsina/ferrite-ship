export interface MarketingNavItem {
	label: string;
	href: string;
}

/** Header navigation. Deliberately icon-free — marketing nav reads as words. */
export const marketingNav: MarketingNavItem[] = [
	{ label: 'Features', href: '#features' },
	{ label: 'How it works', href: '#how-it-works' },
	{ label: 'Tools', href: '#tools' },
	{ label: 'FAQ', href: '#faq' }
];
