import { apiRequest } from '$lib/api/client';
import type { AlertKind } from '$lib/data/notifications';

/**
 * A condition that is true right now, and will clear itself when it stops
 * being true — a server not answering, a disk filling up.
 */
export interface ProblemAlert {
	id: string;
	serverId: string;
	serverName: string;
	kind: AlertKind;
	/** What within the server it concerns — a tool id — or empty. */
	subject: string;
	detail: string;
	openedAt: string;
}

/**
 * A run that did not finish. Unlike an alert this never clears itself: a
 * backup that failed last night stays failed, and the only thing that changes
 * it is running it again.
 */
export interface ProblemFailure {
	jobId: string;
	serverId: string;
	serverName: string;
	title: string;
	kind: string;
	/** Who asked for it, or "Scheduled" where nobody did. */
	actor: string;
	/** What went wrong, in the words the server used. */
	error: string;
	finishedAt: string;
}

export interface Problems {
	alerts: ProblemAlert[];
	failures: ProblemFailure[];
}

export const problemsClient = {
	list: (signal?: AbortSignal) => apiRequest<Problems>('/v1/problems', { signal })
};
