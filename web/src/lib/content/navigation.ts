import type { LucideProps } from '@lucide/svelte';
import Blocks from '@lucide/svelte/icons/blocks';
import CircleQuestionMark from '@lucide/svelte/icons/circle-question-mark';
import History from '@lucide/svelte/icons/history';
import LayoutDashboard from '@lucide/svelte/icons/layout-dashboard';
import LifeBuoy from '@lucide/svelte/icons/life-buoy';
import Server from '@lucide/svelte/icons/server';
import Settings from '@lucide/svelte/icons/settings';
import Sparkles from '@lucide/svelte/icons/sparkles';
import type { Component } from 'svelte';

export interface NavItem {
	label: string;
	href: string;
	icon: Component<LucideProps>;
}

export interface NavGroup {
	label: string;
	items: NavItem[];
}

export const dashboardNavGroups: NavGroup[] = [
	{
		label: 'Main menu',
		items: [
			{ label: 'Overview', href: '/dashboard', icon: LayoutDashboard },
			{ label: 'Servers', href: '/dashboard/servers', icon: Server },
			{ label: 'Tools', href: '/dashboard/services', icon: Blocks }
		]
	},
	{
		label: 'Workspace',
		items: [
			{ label: 'History', href: '/dashboard/activity', icon: History },
			{ label: 'Settings', href: '/dashboard/settings', icon: Settings },
			{ label: 'Help', href: '/dashboard/help', icon: LifeBuoy }
		]
	}
];

export const marketingNav: NavItem[] = [
	{ label: 'What you get', href: '#features', icon: Sparkles },
	{ label: 'Tools', href: '#services', icon: Blocks },
	{ label: 'How it works', href: '#how-it-works', icon: CircleQuestionMark }
];
