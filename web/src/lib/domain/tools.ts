import type { StatusPresentation } from '$lib/domain/status';
import type { Tool, ToolStatus } from '$lib/data/tools';
import type { LucideProps } from '@lucide/svelte';
import Astroid from '@lucide/svelte/icons/astroid';
import ChartColumn from '@lucide/svelte/icons/chart-column';
import ChartLine from '@lucide/svelte/icons/chart-line';
import CircleCheck from '@lucide/svelte/icons/circle-check';
import CircleDashed from '@lucide/svelte/icons/circle-dashed';
import CircleX from '@lucide/svelte/icons/circle-x';
import Database from '@lucide/svelte/icons/database';
import HardDrive from '@lucide/svelte/icons/hard-drive';
import Layers from '@lucide/svelte/icons/layers';
import KeyRound from '@lucide/svelte/icons/key-round';
import LoaderCircle from '@lucide/svelte/icons/loader-circle';
import Package from '@lucide/svelte/icons/package';
import Radio from '@lucide/svelte/icons/radio';
import Route from '@lucide/svelte/icons/route';
import Search from '@lucide/svelte/icons/search';
import Video from '@lucide/svelte/icons/video';
import Zap from '@lucide/svelte/icons/zap';
import type { Component } from 'svelte';

/**
 * The API names an icon; this resolves it to a component.
 *
 * A lookup rather than a dynamic import because icons are imported one by one
 * to keep them out of the bundle wholesale, and a name built at runtime cannot
 * be tree-shaken. An unknown name falls back rather than rendering nothing.
 */
const ICONS: Record<string, Component<LucideProps>> = {
	Database,
	Zap,
	ChartColumn,
	ChartLine,
	Video,
	Route,
	Search,
	Radio,
	Astroid,
	Layers,
	HardDrive,
	KeyRound
};

export function toolIcon(name: string): Component<LucideProps> {
	return ICONS[name] ?? Package;
}

/**
 * How each installation state is shown. Empty status means the tool has never
 * been installed here, which is not a state worth a badge — the card offers to
 * install it instead.
 */
export const TOOL_STATUS: Record<Exclude<ToolStatus, ''>, StatusPresentation> = {
	ready: { label: 'Running', tone: 'ok', icon: CircleCheck },
	installing: { label: 'Setting up', tone: 'info', icon: LoaderCircle, animated: true },
	removing: { label: 'Removing', tone: 'info', icon: LoaderCircle, animated: true },
	failed: { label: 'Did not finish', tone: 'error', icon: CircleX }
};

export function toolStatus(status: ToolStatus): StatusPresentation | null {
	if (!status) return null;
	return TOOL_STATUS[status] ?? { label: 'Not sure', tone: 'pending', icon: CircleDashed };
}

/** True while a job is working on this tool, so the card's buttons wait. */
export function isBusy(tool: Tool): boolean {
	return tool.status === 'installing' || tool.status === 'removing';
}

/** Tools grouped by category, in the order the API returned them. */
export function byCategory(tools: Tool[]): { category: string; tools: Tool[] }[] {
	const groups: { category: string; tools: Tool[] }[] = [];

	for (const tool of tools) {
		const existing = groups.find((group) => group.category === tool.category);
		if (existing) existing.tools.push(tool);
		else groups.push({ category: tool.category, tools: [tool] });
	}

	return groups;
}

/**
 * How a tool's reach is described in one line.
 *
 * Whether something can be reached from the internet is the single most
 * important thing to say about a newly installed database, so it is said
 * plainly rather than left for someone to work out from a port number.
 *
 * Ports alone are no longer enough to answer it. A routed tool deliberately
 * has no public port — Traefik reaches it over the shared network — so
 * reading the ports would call Grafana private while it is published under a
 * domain with a certificate. Keycloak is the worst case: it cannot be
 * installed without a domain, so it is never private, and saying so would be
 * wrong every single time.
 *
 * `fromOutside` is returned rather than recomputed by each caller, because
 * both of them were deciding the padlock icon from the ports themselves — so
 * a routed tool got a padlock beside a label saying it had a web address.
 *
 * `domain` is the server's, and is absent on the catalogue page, which asks
 * "what can I have?" before any server is chosen. The honest answer there is
 * what it depends on, not a guess.
 */
export function reachability(
	tool: Tool,
	domain?: string
): { label: string; detail: string; fromOutside: boolean } {
	if (tool.ports.some((port) => port.public)) {
		return {
			label: 'Open to the internet',
			detail: 'Anyone who knows the address can reach it. That is deliberate: it cannot do its job otherwise.',
			fromOutside: true
		};
	}

	if (tool.web && domain) {
		return {
			label: 'Has its own web address',
			detail: `Reachable at ${tool.id}.${domain} over a secure connection, so anyone can open the sign-in page. Keep the password to yourself.`,
			fromOutside: true
		};
	}

	if (tool.web) {
		return {
			label: 'Private to this server',
			detail:
				'Nothing outside the server can reach it. Give the server a domain and it gets ' +
				'a web address of its own; until then, use the tunnel command to connect.',
			fromOutside: false
		};
	}

	return {
		label: 'Private to this server',
		detail: 'Nothing outside the server can reach it. Use the tunnel command to connect.',
		fromOutside: false
	};
}

/**
 * How a deployed application's state is shown.
 *
 * Separate from SERVER_STATUS: "running" means something different about a
 * container than it does about a machine, and sharing one vocabulary would
 * make the two look interchangeable when they are not.
 */
const UNKNOWN_APP: StatusPresentation = {
	label: 'Never deployed',
	tone: 'pending',
	icon: CircleDashed
};

export const APP_STATUS: Record<string, StatusPresentation> = {
	running: { label: 'Running', tone: 'ok', icon: CircleCheck },
	deploying: { label: 'Deploying', tone: 'info', icon: LoaderCircle, animated: true },
	failed: { label: 'Did not deploy', tone: 'error', icon: CircleX },
	new: UNKNOWN_APP
};

export function appStatus(status: string): StatusPresentation {
	return APP_STATUS[status] ?? UNKNOWN_APP;
}
