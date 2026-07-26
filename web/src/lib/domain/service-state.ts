import type { Tone } from '$components/ui/tone';
import type { ServiceUnit } from '$lib/data/services';

export interface StatePresentation {
	/** Plain language, not systemd's vocabulary. */
	label: string;
	tone: Tone;
}

/**
 * systemd distinguishes active/inactive/failed and running/dead/exited. Most
 * of that detail is noise to someone who just wants to know whether the thing
 * is working, so it collapses to three answers.
 */
export function describeState(unit: ServiceUnit): StatePresentation {
	if (unit.active === 'failed' || unit.sub === 'failed') {
		return { label: 'Not working', tone: 'error' };
	}
	if (unit.active === 'active') {
		// "exited" means a one-shot task that ran and finished, which is normal
		// and healthy — not the same as a daemon that died.
		return unit.sub === 'exited'
			? { label: 'Ran and finished', tone: 'pending' }
			: { label: 'Running', tone: 'ok' };
	}
	if (unit.active === 'activating' || unit.active === 'deactivating') {
		return { label: 'Changing', tone: 'info' };
	}
	return { label: 'Stopped', tone: 'pending' };
}

/** Whether the unit comes back on its own after a reboot. */
export function describeBoot(unit: ServiceUnit): string {
	switch (unit.enabled) {
		case 'enabled':
		case 'enabled-runtime':
			return 'Starts at boot';
		case 'disabled':
			return 'Does not start at boot';
		case 'static':
			return 'Started by something else';
		case 'masked':
			return 'Blocked';
		case '':
			return '—';
		default:
			return unit.enabled;
	}
}
