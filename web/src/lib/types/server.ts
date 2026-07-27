/** Health of a managed node, as reported by its agent. */
export type ServerStatus = 'online' | 'degraded' | 'offline' | 'provisioning' | 'unknown';

export interface ResourceUsage {
	usedBytes: number;
	totalBytes: number;
}

/** How the control plane reaches a server. */
export type ServerConnectionKind = 'demo' | 'ssh';

export interface ManagedServer {
	id: string;
	name: string;
	connectionKind: ServerConnectionKind;
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
	/** When the baseline last succeeded here. Empty means it never has. */
	setUpAt: string;
	services: string[];
	/**
	 * The domain whose wildcard record points at this server, without a scheme
	 * — `example.com`, so a tool is reached at `grafana.example.com`. Empty
	 * means nothing is routed here and everything is reached over a tunnel.
	 */
	domain: string;
	/** Where Let's Encrypt writes about the certificates it issues. */
	acmeEmail: string;
}

export function usageRatio({ usedBytes, totalBytes }: ResourceUsage): number {
	if (totalBytes <= 0) return 0;
	return usedBytes / totalBytes;
}
