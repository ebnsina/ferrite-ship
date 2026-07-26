import { AppError } from '$lib/errors';
import type { ActivityEntry } from '$types/activity';
import type { FleetMetric } from '$types/metric';
import type { ManagedServer } from '$types/server';
import { mockActivity, mockMetrics, mockServers } from './fixtures';
import type { DashboardRepository } from './repository';

const LATENCY_MS = 320;

/** Resolves after a beat so loading states are exercised, and respects aborts. */
function delayed<T>(value: T, signal?: AbortSignal): Promise<T> {
	return new Promise<T>((resolve, reject) => {
		if (signal?.aborted) {
			reject(new AppError({ code: 'timeout', message: 'Request cancelled.' }));
			return;
		}

		const timer = setTimeout(() => {
			signal?.removeEventListener('abort', onAbort);
			resolve(value);
		}, LATENCY_MS);

		function onAbort() {
			clearTimeout(timer);
			reject(new AppError({ code: 'timeout', message: 'Request cancelled.' }));
		}

		signal?.addEventListener('abort', onAbort, { once: true });
	});
}

export const mockRepository: DashboardRepository = {
	listServers: (signal) => delayed<ManagedServer[]>(mockServers, signal),
	listActivity: (signal) => delayed<ActivityEntry[]>(mockActivity, signal),
	listMetrics: (signal) => delayed<FleetMetric[]>(mockMetrics, signal)
};
