import { apiRequest } from '$lib/api/client';

/** One place the app is installed — a person's account or an organisation. */
export interface GitHubInstallation {
	id: number;
	account: string;
	/** "all" or "selected", so a missing repository has an explanation. */
	selection: string;
	connectedAt: string;
}

/**
 * Whether GitHub can be used here, and whether it has been.
 *
 * `configured` is separate from having installations on purpose: "this
 * installation cannot talk to GitHub" and "you have not connected it yet" need
 * different sentences, and one flag would produce a button leading nowhere.
 */
export interface GitHubStatus {
	configured: boolean;
	installations: GitHubInstallation[];
}

export const githubClient = {
	status: (signal?: AbortSignal) => apiRequest<GitHubStatus>('/v1/github/status', { signal }),

	/** Returns where to send the browser to choose repositories. */
	connect: (signal?: AbortSignal) =>
		apiRequest<{ url: string }>('/v1/github/connect', { method: 'POST', signal }),

	/** Forgets it here. The app stays installed on GitHub until removed there. */
	disconnect: (id: number, signal?: AbortSignal) =>
		apiRequest<void>(`/v1/github/installations/${id}`, { method: 'DELETE', signal })
};
