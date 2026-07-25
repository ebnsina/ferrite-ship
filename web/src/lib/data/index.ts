import { env } from '$config/env';
import { apiRepository } from './api-repository';
import { mockRepository } from './mock-repository';
import type { DashboardRepository } from './repository';

export const dashboardRepository: DashboardRepository =
	env.dataSource === 'mock' ? mockRepository : apiRepository;

export type { DashboardRepository } from './repository';
