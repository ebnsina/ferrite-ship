import type { Tone } from '$components/ui/tone';
import type { ActivityStatus } from '$types/activity';
import type { StepOutcome } from '$types/job';
import type { ServerStatus } from '$types/server';
import type { LucideProps } from '@lucide/svelte';
import CircleAlert from '@lucide/svelte/icons/circle-alert';
import CircleCheck from '@lucide/svelte/icons/circle-check';
import CircleDashed from '@lucide/svelte/icons/circle-dashed';
import CircleX from '@lucide/svelte/icons/circle-x';
import Hourglass from '@lucide/svelte/icons/hourglass';
import LoaderCircle from '@lucide/svelte/icons/loader-circle';
import Minus from '@lucide/svelte/icons/minus';
import PlugZap from '@lucide/svelte/icons/plug-zap';
import type { Component } from 'svelte';

export interface StatusPresentation {
	/** Plain language — what a person would say, not what a log would print. */
	label: string;
	tone: Tone;
	icon: Component<LucideProps>;
	/** Whether the icon should spin — only for genuinely in-flight states. */
	animated?: boolean;
}

export const SERVER_STATUS: Record<ServerStatus, StatusPresentation> = {
	online: { label: 'Running fine', tone: 'ok', icon: CircleCheck },
	degraded: { label: 'Working hard', tone: 'warn', icon: CircleAlert },
	offline: { label: 'Not responding', tone: 'error', icon: PlugZap },
	provisioning: { label: 'Setting up', tone: 'info', icon: LoaderCircle, animated: true },
	unknown: { label: 'Not sure yet', tone: 'pending', icon: CircleDashed }
};

export const ACTIVITY_STATUS: Record<ActivityStatus, StatusPresentation> = {
	succeeded: { label: 'Done', tone: 'ok', icon: CircleCheck },
	failed: { label: 'Did not work', tone: 'error', icon: CircleX },
	running: { label: 'In progress', tone: 'info', icon: LoaderCircle, animated: true },
	queued: { label: 'Waiting', tone: 'pending', icon: Hourglass }
};

/** How each step outcome is shown in a job's timeline. */
export const STEP_OUTCOME: Record<StepOutcome, StatusPresentation> = {
	changed: { label: 'Changed', tone: 'ok', icon: CircleCheck },
	unchanged: { label: 'Already fine', tone: 'pending', icon: Minus },
	skipped: { label: 'Not needed', tone: 'info', icon: CircleDashed },
	failed: { label: 'Failed', tone: 'error', icon: CircleX }
};
