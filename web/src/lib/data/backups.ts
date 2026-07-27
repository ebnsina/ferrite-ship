import { apiRequest } from '$lib/api/client';

/**
 * Where backups are sent.
 *
 * The keys are never returned — they are written once and sealed. So the
 * shape you read is not the shape you write, deliberately.
 */
export interface BackupDestination {
	configured: boolean;
	endpoint?: string;
	region?: string;
	bucket?: string;
	prefix?: string;
	updatedAt?: string;
}

export interface DestinationInput {
	endpoint: string;
	region: string;
	bucket: string;
	prefix: string;
	accessKey: string;
	secretKey: string;
}

export type BackupStatus = 'running' | 'ready' | 'failed';

export interface Backup {
	id: string;
	toolId: string;
	objectKey: string;
	sizeBytes: number;
	status: BackupStatus;
	jobId?: string;
	createdAt: string;
}

interface StartedJob {
	id: string;
}

export const backupsClient = {
	destination: (signal?: AbortSignal) =>
		apiRequest<BackupDestination>('/v1/backups/destination', { signal }),

	saveDestination: (input: DestinationInput, signal?: AbortSignal) =>
		apiRequest<BackupDestination>('/v1/backups/destination', {
			method: 'PUT',
			body: input,
			signal
		}),

	forgetDestination: (signal?: AbortSignal) =>
		apiRequest<void>('/v1/backups/destination', { method: 'DELETE', signal }),

	list: (serverId: string, toolId: string, signal?: AbortSignal) =>
		apiRequest<Backup[]>(
			`/v1/servers/${encodeURIComponent(serverId)}/tools/${encodeURIComponent(toolId)}/backups`,
			{ signal }
		),

	start: (serverId: string, toolId: string, signal?: AbortSignal) =>
		apiRequest<StartedJob>(
			`/v1/servers/${encodeURIComponent(serverId)}/tools/${encodeURIComponent(toolId)}/backups`,
			{ method: 'POST', signal }
		),

	/** Overwrites everything currently in the tool. Ask first. */
	restore: (backupId: string, signal?: AbortSignal) =>
		apiRequest<StartedJob>(`/v1/backups/${encodeURIComponent(backupId)}/restore`, {
			method: 'POST',
			signal
		})
};

export type Cadence = 'daily' | 'weekly';

/** When backups happen on their own. Null means they do not. */
export interface BackupSchedule {
	id: string;
	toolId: string;
	cadence: Cadence;
	/** UTC, 0–23. */
	hour: number;
	/** 0 (Sunday) to 6. Only meaningful for a weekly cadence. */
	weekday: number;
	/** How many to keep; older ones are deleted after each run. */
	keep: number;
	lastRunAt: string | null;
	nextRunAt: string;
}

export interface ScheduleInput {
	cadence: Cadence;
	hour: number;
	weekday: number;
	keep: number;
}

function scheduleUrl(serverId: string, toolId: string): string {
	return `/v1/servers/${encodeURIComponent(serverId)}/tools/${encodeURIComponent(toolId)}/schedule`;
}

export const schedulesClient = {
	get: (serverId: string, toolId: string, signal?: AbortSignal) =>
		apiRequest<BackupSchedule | null>(scheduleUrl(serverId, toolId), { signal }),

	save: (serverId: string, toolId: string, input: ScheduleInput, signal?: AbortSignal) =>
		apiRequest<BackupSchedule>(scheduleUrl(serverId, toolId), {
			method: 'PUT',
			body: input,
			signal
		}),

	turnOff: (serverId: string, toolId: string, signal?: AbortSignal) =>
		apiRequest<void>(scheduleUrl(serverId, toolId), { method: 'DELETE', signal })
};
