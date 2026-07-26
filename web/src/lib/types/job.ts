export type JobStatus = 'queued' | 'running' | 'succeeded' | 'failed';

/** Matches store.Job on the Go side. */
export interface Job {
	id: string;
	serverId: string;
	kind: string;
	title: string;
	actor: string;
	status: JobStatus;
	startedAt: string;
	finishedAt: string | null;
	changed: number;
	unchanged: number;
	skipped: number;
	failed: number;
	error?: string;
}

export type JobEventType =
	| 'job-started'
	| 'step-started'
	| 'log'
	| 'step-ended'
	| 'job-ended';

export type LogLevel = 'command' | 'output' | 'info' | 'changed' | 'skipped' | 'error';

export type StepOutcome = 'changed' | 'unchanged' | 'skipped' | 'failed';

export interface JobEvent {
	id: number;
	jobId: string;
	seq: number;
	type: JobEventType;
	stepId?: string;
	stepTitle?: string;
	level?: LogLevel;
	message?: string;
	outcome?: StepOutcome;
	at: string;
}

/** A step and the log lines produced while it ran, assembled from the stream. */
export interface StepRun {
	id: string;
	title: string;
	outcome: StepOutcome | null;
	lines: { seq: number; level: LogLevel; message: string }[];
}
