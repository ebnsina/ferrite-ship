import { apiRequest } from '$lib/api/client';

export interface AuthStatus {
	/** True until the first account exists. */
	needsSetup: boolean;
	authenticated: boolean;
	email?: string;
}

export const authClient = {
	status: (signal?: AbortSignal) => apiRequest<AuthStatus>('/v1/auth/status', { signal }),

	setup: (email: string, password: string) =>
		apiRequest<AuthStatus>('/v1/auth/setup', {
			method: 'POST',
			body: { email, password }
		}),

	login: (email: string, password: string) =>
		apiRequest<AuthStatus>('/v1/auth/login', {
			method: 'POST',
			body: { email, password }
		}),

	logout: () => apiRequest<void>('/v1/auth/logout', { method: 'POST' })
};
