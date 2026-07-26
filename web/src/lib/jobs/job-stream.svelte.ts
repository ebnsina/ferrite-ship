import { env } from '$config/env';
import type { JobEvent, JobEventType, LogLevel, StepOutcome, StepRun } from '$types/job';

export type StreamState = 'connecting' | 'streaming' | 'done' | 'error';

const EVENT_TYPES: JobEventType[] = [
	'job-started',
	'step-started',
	'log',
	'step-ended',
	'job-ended'
];

export interface JobStream {
	readonly steps: StepRun[];
	readonly state: StreamState;
	readonly outcome: string | null;
	readonly summary: string | null;
	close(): void;
}

/**
 * Follows a job's log over Server-Sent Events and assembles it into steps.
 *
 * EventSource reconnects on its own and replays Last-Event-ID, and the server
 * resumes from that sequence number — so a dropped connection costs nothing.
 * Sequence numbers are tracked here too, because a reconnect can redeliver
 * events this client already has.
 *
 * Must be called during component initialisation (it registers an $effect).
 */
export function createJobStream(jobId: string): JobStream {
	let steps = $state<StepRun[]>([]);
	let state = $state<StreamState>('connecting');
	let outcome = $state<string | null>(null);
	let summary = $state<string | null>(null);

	let highestSeq = 0;

	const url = `${env.apiBaseUrl}/v1/jobs/${encodeURIComponent(jobId)}/events`;
	const source = new EventSource(url, { withCredentials: true });

	function currentStep(): StepRun {
		const last = steps[steps.length - 1];
		if (last) return last;

		// Logs can arrive before any step — a connection failure, for instance.
		// Give them somewhere to live rather than dropping them.
		const preamble: StepRun = { id: '__preamble', title: 'Connecting', outcome: null, lines: [] };
		steps = [...steps, preamble];
		return preamble;
	}

	function apply(event: JobEvent) {
		switch (event.type) {
			case 'job-started':
				state = 'streaming';
				break;

			case 'step-started':
				steps = [
					...steps,
					{
						id: event.stepId ?? `step-${event.seq}`,
						title: event.stepTitle ?? 'Working',
						outcome: null,
						lines: []
					}
				];
				break;

			case 'log': {
				const step = currentStep();
				step.lines.push({
					seq: event.seq,
					level: (event.level ?? 'output') as LogLevel,
					message: event.message ?? ''
				});
				// Reassign so the derived UI updates; lines are mutated in place.
				steps = [...steps];
				break;
			}

			case 'step-ended': {
				const step = steps.find((candidate) => candidate.id === event.stepId);
				if (step) {
					step.outcome = (event.outcome ?? null) as StepOutcome | null;
					steps = [...steps];
				}
				break;
			}

			case 'job-ended':
				outcome = event.outcome ?? null;
				summary = event.message ?? null;
				state = 'done';
				source.close();
				break;
		}
	}

	function onEvent(raw: MessageEvent<string>) {
		let event: JobEvent;
		try {
			event = JSON.parse(raw.data) as JobEvent;
		} catch {
			return; // A malformed frame is not worth tearing the view down for.
		}

		if (event.seq <= highestSeq) return; // Redelivered after a reconnect.
		highestSeq = event.seq;

		apply(event);
	}

	for (const type of EVENT_TYPES) {
		source.addEventListener(type, onEvent as EventListener);
	}

	// The server sends this once a finished job has been fully replayed.
	source.addEventListener('done', () => {
		state = 'done';
		source.close();
	});

	source.addEventListener('error', () => {
		// EventSource also fires this while retrying; only a closed source is fatal.
		if (source.readyState === EventSource.CLOSED && state !== 'done') {
			state = 'error';
		}
	});

	$effect(() => () => source.close());

	return {
		get steps() {
			return steps;
		},
		get state() {
			return state;
		},
		get outcome() {
			return outcome;
		},
		get summary() {
			return summary;
		},
		close: () => source.close()
	};
}
