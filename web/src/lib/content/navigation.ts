import type { LucideProps } from '@lucide/svelte';
import Blocks from '@lucide/svelte/icons/blocks';
import History from '@lucide/svelte/icons/history';
import LayoutDashboard from '@lucide/svelte/icons/layout-dashboard';
import LifeBuoy from '@lucide/svelte/icons/life-buoy';
import Server from '@lucide/svelte/icons/server';
import Settings from '@lucide/svelte/icons/settings';
import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
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
			// Above History on purpose: this answers "is anything wrong?", which
			// is a different question from "what has run?" and the more urgent
			// of the two. History is where you go once you know.
			{ label: 'Needs attention', href: '/dashboard/problems', icon: TriangleAlert },
			{ label: 'History', href: '/dashboard/activity', icon: History },
			{ label: 'Settings', href: '/dashboard/settings', icon: Settings },
			{ label: 'Help', href: '/dashboard/help', icon: LifeBuoy }
		]
	}
];
