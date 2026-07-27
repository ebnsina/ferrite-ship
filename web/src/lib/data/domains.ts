import { apiRequest } from '$lib/api/client';
import type { ManagedServer } from '$types/server';

/**
 * What is sent when someone points a domain at a server.
 *
 * Both fields move together, and both empty means "route nothing here" — the
 * state every server is in until somebody says otherwise.
 */
export interface DomainInput {
	domain: string;
	email: string;
}

export const domainsClient = {
	/** Returns the whole server, because the domain is part of how it reads. */
	save: (serverId: string, input: DomainInput, signal?: AbortSignal) =>
		apiRequest<ManagedServer>(`/v1/servers/${encodeURIComponent(serverId)}/domain`, {
			method: 'PUT',
			body: input,
			signal
		})
};
