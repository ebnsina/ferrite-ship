import type { ActivityEntry } from '$types/activity';
import type { FleetMetric } from '$types/metric';
import type { ManagedServer } from '$types/server';

/**
 * Fixtures for `PUBLIC_DATA_SOURCE=mock`. Timestamps are relative to load so
 * the UI always shows plausible "x minutes ago" values.
 */

const minutesAgo = (minutes: number): string =>
	new Date(Date.now() - minutes * 60_000).toISOString();

/** Decimal, matching how hosts advertise capacity and how Intl labels units. */
const GB = 1000 ** 3;

export const mockServers: ManagedServer[] = [
	{
		id: 'srv_01',
		name: 'edge-fra-1',
		setUpAt: minutesAgo(2880),
		connectionKind: 'demo',
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
		services: ['traefik', 'postgres', 'redis'],
		domain: 'example.com',
		acmeEmail: 'ops@example.com'
	},
	{
		id: 'srv_02',
		name: 'media-ams-1',
		setUpAt: minutesAgo(1440),
		connectionKind: 'demo',
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
		services: ['mediamtx', 'clickhouse'],
		domain: 'media.example.net',
		acmeEmail: 'ops@example.com'
	},
	{
		id: 'srv_03',
		name: 'db-fra-1',
		setUpAt: minutesAgo(5760),
		connectionKind: 'demo',
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
		services: ['postgres', 'pgbouncer'],
		domain: '',
		acmeEmail: ''
	},
	{
		id: 'srv_04',
		name: 'worker-sgp-1',
		setUpAt: '',
		connectionKind: 'demo',
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
		services: ['nats'],
		domain: '',
		acmeEmail: ''
	},
	{
		id: 'srv_05',
		name: 'build-fra-2',
		setUpAt: '',
		connectionKind: 'demo',
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
		services: [],
		domain: '',
		acmeEmail: ''
	}
];

export const mockMetrics: FleetMetric[] = [
	{
		id: 'servers',
		label: 'Servers connected',
		value: 5,
		format: 'count',
		deltaRatio: 0.25,
		higherIsBetter: true,
		series: [3, 3, 4, 4, 4, 5, 5]
	},
	{
		id: 'uptime',
		label: 'Time up and running',
		value: 0.994,
		format: 'percent',
		deltaRatio: 0.008,
		higherIsBetter: true,
		series: [0.981, 0.986, 0.99, 0.984, 0.992, 0.993, 0.994]
	},
	{
		id: 'busy',
		label: 'How busy they are',
		value: 0.41,
		format: 'percent',
		deltaRatio: -0.06,
		higherIsBetter: false,
		series: [0.52, 0.49, 0.47, 0.51, 0.44, 0.43, 0.41]
	},
	{
		id: 'storage',
		label: 'Storage in use',
		value: 980 * GB,
		format: 'bytes',
		deltaRatio: 0.12,
		higherIsBetter: false,
		series: [720, 760, 800, 845, 890, 940, 980].map((value) => value * GB)
	}
];

export const mockActivity: ActivityEntry[] = [
	{
		id: 'job_9f2',
		title: 'Setting up a new server',
		serverName: 'build-fra-2',
		actor: 'ebnsina',
		status: 'running',
		startedAt: minutesAgo(4),
		durationMs: null
	},
	{
		id: 'job_9f1',
		title: 'Installed live video streaming',
		serverName: 'media-ams-1',
		actor: 'ebnsina',
		status: 'succeeded',
		startedAt: minutesAgo(26),
		durationMs: 94_000
	},
	{
		id: 'job_9e8',
		title: 'Backed up the database',
		serverName: 'db-fra-1',
		actor: 'Scheduled',
		status: 'succeeded',
		startedAt: minutesAgo(63),
		durationMs: 412_000
	},
	{
		id: 'job_9e4',
		title: 'Updated the helper program',
		serverName: 'worker-sgp-1',
		actor: 'Scheduled',
		status: 'failed',
		startedAt: minutesAgo(38),
		durationMs: 21_000
	},
	{
		id: 'job_9e0',
		title: 'Renewed security certificates',
		serverName: 'edge-fra-1',
		actor: 'Scheduled',
		status: 'succeeded',
		startedAt: minutesAgo(155),
		durationMs: 8_400
	}
];
