import type { ActivityEntry } from '$types/activity';
import type { ManagedServer } from '$types/server';

/**
 * Fixtures for `PUBLIC_DATA_SOURCE=mock`. Timestamps are relative to load so
 * the UI always shows plausible "x minutes ago" values.
 */

const minutesAgo = (minutes: number): string =>
	new Date(Date.now() - minutes * 60_000).toISOString();

const GB = 1000 ** 3;

export const mockServers: ManagedServer[] = [
	{
		id: 'srv_01',
		name: 'edge-fra-1',
		hostname: 'edge-fra-1.ferrite.internal',
		ipAddress: '198.51.100.24',
		region: 'Frankfurt',
		operatingSystem: 'Ubuntu 24.04 LTS',
		agentVersion: '0.4.2',
		status: 'online',
		cpuUsage: 0.23,
		memory: { usedBytes: 5.4 * GB, totalBytes: 16 * GB },
		disk: { usedBytes: 88 * GB, totalBytes: 320 * GB },
		uptimeMs: 41 * 86_400_000 + 6 * 3_600_000,
		lastSeenAt: minutesAgo(0.2),
		services: ['traefik', 'postgres', 'redis']
	},
	{
		id: 'srv_02',
		name: 'media-ams-1',
		hostname: 'media-ams-1.ferrite.internal',
		ipAddress: '198.51.100.71',
		region: 'Amsterdam',
		operatingSystem: 'Ubuntu 24.04 LTS',
		agentVersion: '0.4.2',
		status: 'degraded',
		cpuUsage: 0.87,
		memory: { usedBytes: 27.1 * GB, totalBytes: 32 * GB },
		disk: { usedBytes: 612 * GB, totalBytes: 640 * GB },
		uptimeMs: 12 * 86_400_000 + 3 * 3_600_000,
		lastSeenAt: minutesAgo(0.4),
		services: ['mediamtx', 'clickhouse']
	},
	{
		id: 'srv_03',
		name: 'db-fra-1',
		hostname: 'db-fra-1.ferrite.internal',
		ipAddress: '198.51.100.9',
		region: 'Frankfurt',
		operatingSystem: 'Ubuntu 22.04 LTS',
		agentVersion: '0.4.1',
		status: 'online',
		cpuUsage: 0.41,
		memory: { usedBytes: 19.8 * GB, totalBytes: 64 * GB },
		disk: { usedBytes: 240 * GB, totalBytes: 1024 * GB },
		uptimeMs: 96 * 86_400_000,
		lastSeenAt: minutesAgo(0.1),
		services: ['postgres', 'pgbouncer']
	},
	{
		id: 'srv_04',
		name: 'worker-sgp-1',
		hostname: 'worker-sgp-1.ferrite.internal',
		ipAddress: '203.0.113.44',
		region: 'Singapore',
		operatingSystem: 'Ubuntu 24.04 LTS',
		agentVersion: '0.4.2',
		status: 'offline',
		cpuUsage: 0,
		memory: { usedBytes: 0, totalBytes: 8 * GB },
		disk: { usedBytes: 31 * GB, totalBytes: 160 * GB },
		uptimeMs: 0,
		lastSeenAt: minutesAgo(37),
		services: ['nats']
	},
	{
		id: 'srv_05',
		name: 'build-fra-2',
		hostname: 'build-fra-2.ferrite.internal',
		ipAddress: '198.51.100.132',
		region: 'Frankfurt',
		operatingSystem: 'Ubuntu 24.04 LTS',
		agentVersion: '0.4.2',
		status: 'provisioning',
		cpuUsage: 0.12,
		memory: { usedBytes: 1.2 * GB, totalBytes: 16 * GB },
		disk: { usedBytes: 9 * GB, totalBytes: 320 * GB },
		uptimeMs: 4 * 60_000,
		lastSeenAt: minutesAgo(0.05),
		services: []
	}
];

export const mockActivity: ActivityEntry[] = [
	{
		id: 'job_9f2',
		title: 'Baseline hardening',
		serverName: 'build-fra-2',
		actor: 'ebnsina',
		status: 'running',
		startedAt: minutesAgo(4),
		durationMs: null
	},
	{
		id: 'job_9f1',
		title: 'Deploy mediamtx',
		serverName: 'media-ams-1',
		actor: 'ebnsina',
		status: 'succeeded',
		startedAt: minutesAgo(26),
		durationMs: 94_000
	},
	{
		id: 'job_9e8',
		title: 'Backup postgres',
		serverName: 'db-fra-1',
		actor: 'scheduler',
		status: 'succeeded',
		startedAt: minutesAgo(63),
		durationMs: 412_000
	},
	{
		id: 'job_9e4',
		title: 'Agent upgrade 0.4.1 → 0.4.2',
		serverName: 'worker-sgp-1',
		actor: 'scheduler',
		status: 'failed',
		startedAt: minutesAgo(38),
		durationMs: 21_000
	},
	{
		id: 'job_9e0',
		title: 'Rotate TLS certificates',
		serverName: 'edge-fra-1',
		actor: 'scheduler',
		status: 'succeeded',
		startedAt: minutesAgo(155),
		durationMs: 8_400
	}
];
