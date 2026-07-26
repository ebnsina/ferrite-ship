import type { ManagedServer, ServerStatus } from '$types/server';
import { SERVER_STATUS, type StatusPresentation } from './status';

export interface ServerGroup {
	status: ServerStatus;
	presentation: StatusPresentation;
	servers: ManagedServer[];
}

/** Column order on the board: healthy first, problems last but never hidden. */
const GROUP_ORDER: readonly ServerStatus[] = [
	'online',
	'degraded',
	'provisioning',
	'offline',
	'unknown'
];

/**
 * Buckets servers into status columns. Empty statuses are dropped so the board
 * never shows a column that says nothing.
 */
export function groupServersByStatus(servers: readonly ManagedServer[]): ServerGroup[] {
	return GROUP_ORDER.map((status) => ({
		status,
		presentation: SERVER_STATUS[status],
		servers: servers.filter((server) => server.status === status)
	})).filter((group) => group.servers.length > 0);
}
