import type { AlertKind } from '$lib/data/notifications';

/**
 * How a problem is described on screen.
 *
 * The email wording lives on the server, in `internal/notify`, because that is
 * where it is composed. This is the short form — one line and somewhere to go —
 * and it is kept here rather than inside the component so the two can be read
 * side by side and kept saying the same thing.
 */
export const ALERT_COPY: Record<
	AlertKind,
	{
		title: (server: string, subject: string) => string;
		action: string;
		href: (serverId: string, subject: string) => string;
	}
> = {
	'backup-failed': {
		title: (server, subject) => `The scheduled backup of ${subject} on ${server} did not finish`,
		action: 'See backups',
		href: (serverId, subject) => `/dashboard/servers/${serverId}/tools/${subject}`
	},
	'server-down': {
		title: (server) => `${server} is not responding`,
		action: 'See the server',
		href: (serverId) => `/dashboard/servers/${serverId}`
	},
	'disk-low': {
		title: (server) => `${server} is running out of disk`,
		action: 'Free up space',
		href: (serverId) => `/dashboard/servers/${serverId}/storage`
	}
};
