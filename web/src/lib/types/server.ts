/** Health of a managed node, as reported by its agent. */
export type ServerStatus = 'online' | 'degraded' | 'offline' | 'provisioning' | 'unknown';

export interface ResourceUsage {
	usedBytes: number;
	totalBytes: number;
}

export interface ManagedServer {
	id: string;
	name: string;
	hostname: string;
	ipAddress: string;
	region: string;
	operatingSystem: string;
	agentVersion: string;
	status: ServerStatus;
	/** 0–1 ratio, not a percentage. */
	cpuUsage: number;
	memory: ResourceUsage;
	disk: ResourceUsage;
	uptimeMs: number;
	lastSeenAt: string;
	services: string[];
}

export function usageRatio({ usedBytes, totalBytes }: ResourceUsage): number {
	if (totalBytes <= 0) return 0;
	return usedBytes / totalBytes;
}
