import type { LucideProps } from '@lucide/svelte';
import BellRing from '@lucide/svelte/icons/bell-ring';
import Blocks from '@lucide/svelte/icons/blocks';
import Globe from '@lucide/svelte/icons/globe';
import MonitorPlay from '@lucide/svelte/icons/monitor-play';
import ShieldCheck from '@lucide/svelte/icons/shield-check';
import Zap from '@lucide/svelte/icons/zap';
import type { Component } from 'svelte';

export interface Feature {
	id: string;
	title: string;
	description: string;
	icon: Component<LucideProps>;
}

export const features: Feature[] = [
	{
		id: 'setup',
		title: 'Set up in one step',
		description:
			'Copy one line, paste it on your new server, and you are done. No config files, no keys to look after, about a minute start to finish.',
		icon: Zap
	},
	{
		id: 'safe',
		title: 'Made safe automatically',
		description:
			'We close the doors nothing needs, block repeat login attempts, and keep security updates arriving — without you learning a single command.',
		icon: ShieldCheck
	},
	{
		id: 'control',
		title: 'Your whole server, one window',
		description:
			'Browse and edit files, start and stop things, see what is using space, open a real terminal. Everything you would normally do by logging in.',
		icon: MonitorPlay
	},
	{
		id: 'tools',
		title: 'Add tools with one click',
		description:
			'Databases, caches, file storage, video streaming. Pick one and it starts running, with passwords made for you and backups already booked in.',
		icon: Blocks
	},
	{
		id: 'web',
		title: 'Websites and video, both',
		description:
			'Your sites get a secure web address on their own. Live video and streaming work here too, which most tools like this cannot handle.',
		icon: Globe
	},
	{
		id: 'alerts',
		title: 'Know before your users do',
		description:
			'Watch what your server is doing as it happens, and get a message the moment something starts going wrong.',
		icon: BellRing
	}
];
