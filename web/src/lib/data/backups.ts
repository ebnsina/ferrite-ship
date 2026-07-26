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
