import type { ManagedServer } from '$types/server';

export interface FleetSummary {
	total: number;
	online: number;
	needsAttention: number;
	/** Mean CPU ratio across reachable servers only. */
	averageCpu: number;
	memoryUsedBytes: number;
	memoryTotalBytes: number;
}

export function summarizeFleet(servers: readonly ManagedServer[]): FleetSummary {
	const reachable = servers.filter((server) => server.status !== 'offline');

	const cpuTotal = reachable.reduce((sum, server) => sum + server.cpuUsage, 0);

	return {
		total: servers.length,
		online: servers.filter((server) => server.status === 'online').length,
		needsAttention: servers.filter(
			(server) => server.status === 'degraded' || server.status === 'offline'
		).length,
		averageCpu: reachable.length === 0 ? 0 : cpuTotal / reachable.length,
		memoryUsedBytes: servers.reduce((sum, server) => sum + server.memory.usedBytes, 0),
		memoryTotalBytes: servers.reduce((sum, server) => sum + server.memory.totalBytes, 0)
	};
}
