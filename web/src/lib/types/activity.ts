export type ActivityStatus = 'succeeded' | 'failed' | 'running' | 'queued';

export interface ActivityEntry {
	id: string;
	/** Human label for the job kind, e.g. "Baseline hardening". */
	title: string;
	serverName: string;
	actor: string;
	status: ActivityStatus;
	startedAt: string;
	durationMs: number | null;
}
