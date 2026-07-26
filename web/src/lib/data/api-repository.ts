import { apiRequest } from '$lib/api/client';
import type { ActivityEntry } from '$types/activity';
import type { FleetMetric } from '$types/metric';
import type { ManagedServer } from '$types/server';
import type { DashboardRepository } from './repository';

/** Talks to the Go control plane. Errors surface as AppError from apiRequest. */
export const apiRepository: DashboardRepository = {
	listServers: (signal) => apiRequest<ManagedServer[]>('/v1/servers', { signal }),
	listActivity: (signal) => apiRequest<ActivityEntry[]>('/v1/activity', { signal }),
	listMetrics: (signal) => apiRequest<FleetMetric[]>('/v1/metrics', { signal })
};
