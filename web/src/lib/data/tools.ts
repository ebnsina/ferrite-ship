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

/** A ready-made query offered as a starting point. */
export interface Preset {
	label: string;
	description: string;
	query: string;
}

/** A query the owner kept for later. */
export interface SavedQuery {
	id: string;
	toolId: string;
	name: string;
	query: string;
	savedAt: string;
}

export interface Tool {
	id: string;
	name: string;
	summary: string;
	category: string;
	/** A lucide icon name, resolved by $lib/domain/tools. */
	icon: string;
	/** The tool's own brand colour, as #rrggbb. */
	accent: string;
	/** Whether this tool can be queried from the dashboard. */
	hasConsole: boolean;
	/** What you type into the console, e.g. "SQL". */
	consoleLanguage?: string;
	consolePlaceholder?: string;
	/** Ready-made queries to start from. */
	consolePresets?: Preset[];
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
	/**
	 * Opened in a browser rather than pasted into a client. The url then has no
	 * password in it, and the sign-in details stand on their own.
	 */
	web: boolean;
	/**
	 * The browser interface of a tool that also speaks to clients on another
	 * port — RabbitMQ's management page, MinIO's file browser. Absent for
	 * everything else, including tools that are nothing but a web interface,
	 * where `url` is already that address.
	 */
	webUrl?: string;
}

interface StartedJob {
	id: string;
}

/** One query's answer. Rows are strings: the shapes differ per database and
 *  the console displays them rather than computing with them. */
export interface QueryResult {
	columns: string[];
	rows: string[][];
	/** True when the row cap cut the answer short. */
	truncated: boolean;
	/** True when the database rejected the query; message is what it said. */
	failed: boolean;
	message?: string;
	tookMs: number;
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

	/** Runs one query. Never cached and never logged — it is the owner's data. */
	query: (serverId: string, toolId: string, query: string, signal?: AbortSignal) =>
		apiRequest<QueryResult>(`${base(serverId)}/${encodeURIComponent(toolId)}/query`, {
			method: 'POST',
			body: { query },
			signal
		}),

	savedQueries: (serverId: string, toolId: string, signal?: AbortSignal) =>
		apiRequest<SavedQuery[]>(`${base(serverId)}/${encodeURIComponent(toolId)}/queries`, { signal }),

	saveQuery: (serverId: string, toolId: string, name: string, query: string, signal?: AbortSignal) =>
		apiRequest<SavedQuery>(`${base(serverId)}/${encodeURIComponent(toolId)}/queries`, {
			method: 'POST',
			body: { name, query },
			signal
		}),

	deleteSavedQuery: (serverId: string, toolId: string, queryId: string, signal?: AbortSignal) =>
		apiRequest<void>(
			`${base(serverId)}/${encodeURIComponent(toolId)}/queries/${encodeURIComponent(queryId)}`,
			{ method: 'DELETE', signal }
		),

	connection: (serverId: string, toolId: string, signal?: AbortSignal) =>
		apiRequest<ToolConnection>(
			`${base(serverId)}/${encodeURIComponent(toolId)}/connection`,
			{ signal }
		)
};
