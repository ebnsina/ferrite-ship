import { apiRequest } from '$lib/api/client';

/** Where an installation is in its life. Empty means never installed here. */
export type ToolStatus = '' | 'installing' | 'ready' | 'failed' | 'removing';

export interface ToolPort {
	number: number;
	protocol: string;
	/** Plain language: "Database connections", not "pgwire". */
	purpose: string;
	/** Whether the firewall is opened for it. */
	public: boolean;
}

export interface ToolAccess {
	scheme: string;
	username: string;
	database: string;
	port: number;
}

export interface Tool {
	id: string;
	name: string;
	summary: string;
	category: string;
	/** A lucide icon name, resolved by $lib/domain/tool-icons. */
	icon: string;
	image: string;
	version: string;
	ports: ToolPort[];
	access?: ToolAccess;
	/** What removing this leaves behind, in the words shown to the person. */
	dataNote: string;
	/** Whether removing it can leave data behind, and so whether deleting that
	 *  data is worth offering. */
	keepsData: boolean;
	status: ToolStatus;
	installedAt: string;
	lastJobId?: string;
}

export interface ToolConnection {
	toolId: string;
	name: string;
	url: string;
	host: string;
	port: number;
	username: string;
	password: string;
	database?: string;
	public: boolean;
	/** The ssh command that makes a private tool reachable from a laptop. */
	tunnel?: string;
}

interface StartedJob {
	id: string;
}

function base(serverId: string): string {
	return `/v1/servers/${encodeURIComponent(serverId)}/tools`;
}

export const toolsClient = {
	/** Everything installable, with no server in mind. */
	catalog: (signal?: AbortSignal) => apiRequest<Tool[]>('/v1/catalog', { signal }),

	list: (serverId: string, signal?: AbortSignal) => apiRequest<Tool[]>(base(serverId), { signal }),

	install: (serverId: string, toolId: string, signal?: AbortSignal) =>
		apiRequest<StartedJob>(base(serverId), { method: 'POST', body: { toolId }, signal }),

	/**
	 * purge additionally deletes the stored data and cannot be undone, so it is
	 * always passed explicitly rather than defaulted.
	 */
	remove: (serverId: string, toolId: string, purge: boolean, signal?: AbortSignal) =>
		apiRequest<StartedJob>(
			`${base(serverId)}/${encodeURIComponent(toolId)}?purge=${purge ? 'true' : 'false'}`,
			{ method: 'DELETE', signal }
		),

	connection: (serverId: string, toolId: string, signal?: AbortSignal) =>
		apiRequest<ToolConnection>(
			`${base(serverId)}/${encodeURIComponent(toolId)}/connection`,
			{ signal }
		)
};
