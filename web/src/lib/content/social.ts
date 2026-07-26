export type SocialKind = 'github' | 'x' | 'email';

export interface SocialLink {
	kind: SocialKind;
	label: string;
	href: string;
}

/**
 * PLACEHOLDER LINKS — only the GitHub URL points somewhere real. Fill in the
 * others (or delete the entry) before launch; anything left blank is simply
 * not rendered, so removing one is safe.
 */
const allSocialLinks: SocialLink[] = [
	{
		kind: 'github',
		label: 'Ferrite Ship on GitHub',
		href: 'https://github.com/ebnsina/ferrite-ship'
	},
	{ kind: 'x', label: 'Ferrite Ship on X', href: '' },
	{ kind: 'email', label: 'Email Ferrite Ship', href: '' }
];

export const socialLinks = allSocialLinks.filter((link) => link.href !== '');
