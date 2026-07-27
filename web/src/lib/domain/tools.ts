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
	Astroid
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
 * How a tool's ports are described in one line.
 *
 * Whether something is reachable from the internet is the single most
 * important thing to say about a newly installed database, so it is said
 * plainly rather than left for someone to work out from a port number.
 */
export function reachability(tool: Tool): { label: string; detail: string } {
	const isPublic = tool.ports.some((port) => port.public);

	return isPublic
		? {
				label: 'Open to the internet',
				detail: 'Anyone who knows the address can reach it, which is the point of a media server.'
			}
		: {
				label: 'Private to this server',
				detail: 'Nothing outside the server can reach it. Use the tunnel command to connect.'
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
