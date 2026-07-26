import { apiRequest } from '$lib/api/client';

export type AppStatus = 'new' | 'deploying' | 'running' | 'failed';

/** One application deployed from a git repository. */
export interface App {
	id: string;
	serverId: string;
	name: string;
	repository: string;
	branch: string;
	/** Empty for an app that runs but is not published to a domain. */
	domain: string;
	port: number;
	status: AppStatus;
	lastJobId?: string;
	createdAt: string;
	deployedAt: string | null;
}

export interface AppInput {
	name: string;
	repository: string;
	branch: string;
	domain: string;
	port: number;
	/** Written to the container's environment. Sealed before it is stored. */
	env: Record<string, string>;
}

interface StartedJob {
	id: string;
}

export const appsClient = {
	list: (serverId: string, signal?: AbortSignal) =>
		apiRequest<App[]>(`/v1/servers/${encodeURIComponent(serverId)}/apps`, { signal }),

	create: (serverId: string, input: AppInput, signal?: AbortSignal) =>
		apiRequest<App>(`/v1/servers/${encodeURIComponent(serverId)}/apps`, {
			method: 'POST',
			body: input,
			signal
		}),

	update: (appId: string, input: AppInput, signal?: AbortSignal) =>
		apiRequest<App>(`/v1/apps/${encodeURIComponent(appId)}`, {
			method: 'PUT',
			body: input,
			signal
		}),

	deploy: (appId: string, signal?: AbortSignal) =>
		apiRequest<StartedJob>(`/v1/apps/${encodeURIComponent(appId)}/deploy`, {
			method: 'POST',
			signal
		}),

	remove: (appId: string, signal?: AbortSignal) =>
		apiRequest<StartedJob>(`/v1/apps/${encodeURIComponent(appId)}`, {
			method: 'DELETE',
			signal
		})
};
