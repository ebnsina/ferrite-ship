import type { LucideProps } from '@lucide/svelte';
import Building2 from '@lucide/svelte/icons/building-2';
import ChartColumn from '@lucide/svelte/icons/chart-column';
import Rocket from '@lucide/svelte/icons/rocket';
import Video from '@lucide/svelte/icons/video';
import type { Component } from 'svelte';

export interface UseCase {
	id: string;
	title: string;
	description: string;
	icon: Component<LucideProps>;
	points: string[];
}

export const useCases: UseCase[] = [
	{
		id: 'builders',
		title: 'Side projects and small products',
		description:
			'You want your app online without spending a weekend reading server guides every time.',
		icon: Rocket,
		points: ['Set up once, reuse forever', 'Databases ready in a click', 'Backups from day one']
	},
	{
		id: 'agencies',
		title: 'Agencies and freelancers',
		description: 'You look after servers for several clients and need them all in one place.',
		icon: Building2,
		points: ['Keep clients separate', 'Invite your team', 'See every server at a glance']
	},
	{
		id: 'media',
		title: 'Video and live streaming',
		description:
			'Streaming needs more than ordinary web traffic, and most hosting panels cannot handle it.',
		icon: Video,
		points: ['Live video supported', 'Streaming tools included', 'Handles heavy traffic']
	},
	{
		id: 'data',
		title: 'Data and reporting',
		description: 'You run databases and dashboards, and you would rather not babysit them.',
		icon: ChartColumn,
		points: ['Databases made for you', 'Automatic backups', 'Alerts when space runs low']
	}
];
