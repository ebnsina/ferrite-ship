import type { ActivityEntry } from '$types/activity';
import type { ManagedServer } from '$types/server';

/**
 * The dashboard talks to this interface only. Swapping fixtures for the real
 * control plane is a one-line change in `index.ts` — no component touches fetch.
 */
export interface DashboardRepository {
	listServers(signal?: AbortSignal): Promise<ManagedServer[]>;
	listActivity(signal?: AbortSignal): Promise<ActivityEntry[]>;
}
