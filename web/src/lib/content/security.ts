import type { LucideProps } from '@lucide/svelte';
import DoorClosed from '@lucide/svelte/icons/door-closed';
import Eye from '@lucide/svelte/icons/eye';
import KeyRound from '@lucide/svelte/icons/key-round';
import Trash2 from '@lucide/svelte/icons/trash-2';
import type { Component } from 'svelte';

export interface SecurityPoint {
	id: string;
	title: string;
	description: string;
	icon: Component<LucideProps>;
}

/** Every claim here is a property of the agent design, not a marketing promise. */
export const securityPoints: SecurityPoint[] = [
	{
		id: 'keys',
		title: 'We never keep your password',
		description:
			'Your server checks in with us using its own key, made on the machine itself. There is no master password on our side to lose.',
		icon: KeyRound
	},
	{
		id: 'ports',
		title: 'Nothing new is left open',
		description:
			'Your server reaches out to us, never the other way around. We do not open a single extra door to the internet.',
		icon: DoorClosed
	},
	{
		id: 'audit',
		title: 'Every action is written down',
		description:
			'Who did what, on which server, and when. You can read the whole record, and export it whenever you like.',
		icon: Eye
	},
	{
		id: 'exit',
		title: 'Leave whenever you want',
		description:
			'One command removes us completely and leaves your server exactly as it was. Nothing is held hostage.',
		icon: Trash2
	}
];
