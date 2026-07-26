import { env } from '$config/env';
import { apiRequest } from '$lib/api/client';
import { AppError } from '$lib/errors';
import type { Job } from '$types/job';
import type { ManagedServer } from '$types/server';

/** What the user can ask the control plane to do, as opposed to read. */
export interface DashboardCommands {
	connectServer(input: ConnectServerInput, signal?: AbortSignal): Promise<ManagedServer>;
	startBaseline(serverId: string, options?: StartJobOptions, signal?: AbortSignal): Promise<Job>;
	getJob(jobId: string, signal?: AbortSignal): Promise<Job>;
	removeServer(serverId: string, signal?: AbortSignal): Promise<void>;
}

export type ConnectionKind = 'demo' | 'ssh';

export interface StartJobOptions {
	/** Run every check and report what would change, altering nothing. */
	dryRun?: boolean;
}

export interface ConnectServerInput {
	name: string;
	connectionKind: ConnectionKind;
	host?: string;
	port?: number;
	user?: string;
	region?: string;
	password?: string;
	privateKey?: string;
	/** Installed for the account the setup creates, so you can log in as it. */
	publicKey?: string;
}

const apiCommands: DashboardCommands = {
	connectServer: (input, signal) =>
		apiRequest<ManagedServer>('/v1/servers', { method: 'POST', body: input, signal }),

	startBaseline: (serverId, options, signal) =>
		apiRequest<Job>(`/v1/servers/${encodeURIComponent(serverId)}/jobs`, {
			method: 'POST',
			body: { kind: 'baseline', dryRun: options?.dryRun ?? false },
			signal
		}),

	getJob: (jobId, signal) =>
		apiRequest<Job>(`/v1/jobs/${encodeURIComponent(jobId)}`, { signal }),

	removeServer: (serverId, signal) =>
		apiRequest<void>(`/v1/servers/${encodeURIComponent(serverId)}`, {
			method: 'DELETE',
			signal
		})
};

/**
 * In mock mode there is no control plane to change anything on. Rather than
 * pretend a server was created, say plainly why it cannot be.
 */
function unavailable(): never {
	throw new AppError({
		code: 'config',
		message: 'This needs the control plane running.',
		action: 'Set PUBLIC_DATA_SOURCE=api and start the ferrite-ship server.',
		retryable: false
	});
}

const mockCommands: DashboardCommands = {
	connectServer: unavailable,
	startBaseline: unavailable,
	getJob: unavailable,
	removeServer: unavailable
};

export const dashboardCommands: DashboardCommands =
	env.dataSource === 'mock' ? mockCommands : apiCommands;
