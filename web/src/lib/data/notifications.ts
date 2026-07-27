import { apiRequest } from '$lib/api/client';

/**
 * What this account asked to be told, and whether this installation can tell
 * it.
 *
 * `canSend` is separate from the preferences on purpose: without a mail server
 * the settings still save, and a page that showed switches without saying so
 * would leave somebody believing alerts were on.
 */
export interface NotificationSettings {
	canSend: boolean;
	email: string;
	onBackupFailed: boolean;
	onServerDown: boolean;
	onDiskLow: boolean;
	/** How full a disk has to be before it is worth saying so. */
	diskPercent: number;
	updatedAt?: string;
}

export type NotificationInput = Omit<NotificationSettings, 'canSend' | 'updatedAt'>;

export type AlertKind = 'backup-failed' | 'server-down' | 'disk-low';

/** A condition that is true right now. Cleared ones are not returned. */
export interface Alert {
	id: string;
	serverId: string;
	kind: AlertKind;
	/** What within the server it concerns — a tool id — or empty. */
	subject: string;
	detail: string;
	openedAt: string;
	clearedAt: string | null;
}

export const notificationsClient = {
	get: (signal?: AbortSignal) => apiRequest<NotificationSettings>('/v1/notifications', { signal }),

	save: (input: NotificationInput, signal?: AbortSignal) =>
		apiRequest<NotificationSettings>('/v1/notifications', {
			method: 'PUT',
			body: input,
			signal
		}),

	/** Proves the whole path. The reply names the address it reached. */
	test: (signal?: AbortSignal) =>
		apiRequest<{ sentTo: string }>('/v1/notifications/test', { method: 'POST', signal }),

	alerts: (signal?: AbortSignal) => apiRequest<Alert[]>('/v1/alerts', { signal })
};
