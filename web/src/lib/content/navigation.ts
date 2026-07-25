import type { LucideProps } from '@lucide/svelte';
import Blocks from '@lucide/svelte/icons/blocks';
import CircleQuestionMark from '@lucide/svelte/icons/circle-question-mark';
import History from '@lucide/svelte/icons/history';
import LayoutDashboard from '@lucide/svelte/icons/layout-dashboard';
import Server from '@lucide/svelte/icons/server';
import Settings from '@lucide/svelte/icons/settings';
import Sparkles from '@lucide/svelte/icons/sparkles';
import type { Component } from 'svelte';

export interface NavItem {
	label: string;
	href: string;
	icon: Component<LucideProps>;
	/** Shown under the label in the sidebar, so nothing needs explaining twice. */
	hint?: string;
}

export const dashboardNav: NavItem[] = [
	{ label: 'Overview', href: '/dashboard', icon: LayoutDashboard, hint: 'How everything is doing' },
	{ label: 'Servers', href: '/dashboard/servers', icon: Server, hint: 'Your machines' },
	{ label: 'Tools', href: '/dashboard/services', icon: Blocks, hint: 'Databases and add-ons' },
	{ label: 'History', href: '/dashboard/activity', icon: History, hint: 'What has run lately' },
	{ label: 'Settings', href: '/dashboard/settings', icon: Settings, hint: 'Account and team' }
];

export const marketingNav: NavItem[] = [
	{ label: 'What you get', href: '#features', icon: Sparkles },
	{ label: 'Tools', href: '#services', icon: Blocks },
	{ label: 'How it works', href: '#how-it-works', icon: CircleQuestionMark }
];
