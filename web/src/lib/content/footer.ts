export interface FooterLink {
	label: string;
	href: string;
}

export interface FooterColumn {
	heading: string;
	links: FooterLink[];
}

export const footerColumns: FooterColumn[] = [
	{
		heading: 'Product',
		links: [
			{ label: 'Features', href: '#features' },
			{ label: 'How it works', href: '#how-it-works' },
			{ label: 'Tools', href: '#tools' },
			{ label: 'Security', href: '#security' }
		]
	},
	{
		heading: 'Who it is for',
		links: [
			{ label: 'Side projects', href: '#use-cases' },
			{ label: 'Agencies', href: '#use-cases' },
			{ label: 'Video and streaming', href: '#use-cases' },
			{ label: 'Data and reporting', href: '#use-cases' }
		]
	},
	{
		heading: 'Resources',
		links: [
			{ label: 'Getting started', href: '#how-it-works' },
			{ label: 'Questions', href: '#faq' },
			{ label: 'Dashboard', href: '/dashboard' },
			{ label: 'Help', href: '/dashboard/help' }
		]
	}
];
