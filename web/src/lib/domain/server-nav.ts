import type { LucideProps } from '@lucide/svelte';
import Boxes from '@lucide/svelte/icons/boxes';
import Cog from '@lucide/svelte/icons/cog';
import FolderOpen from '@lucide/svelte/icons/folder-open';
import Gauge from '@lucide/svelte/icons/gauge';
import Package from '@lucide/svelte/icons/package';
import SquareTerminal from '@lucide/svelte/icons/square-terminal';
import type { Component } from 'svelte';

/**
 * The sections of one server, defined once.
 *
 * Both the sidebar and the breadcrumb read this. They were going to drift
 * otherwise — a section added to one and forgotten in the other is how a page
 * ends up reachable from exactly one place.
 */
export interface ServerSection {
	label: string;
	/** Appended to /dashboard/servers/{id}; empty is the server itself. */
	path: string;
	icon: Component<LucideProps>;
	description: string;
}

export const serverSections: ServerSection[] = [
	{ label: 'Overview', path: '', icon: Gauge, description: 'How hard it is working' },
	{ label: 'Applications', path: '/apps', icon: Boxes, description: 'Your own code' },
	{ label: 'Tools', path: '/tools', icon: Package, description: 'Databases and more' },
	{ label: 'Services', path: '/services', icon: Cog, description: 'What is running' },
	{ label: 'Files', path: '/files', icon: FolderOpen, description: 'Browse and edit' },
	{ label: 'Terminal', path: '/terminal', icon: SquareTerminal, description: 'A real shell' }
];

export function serverSectionHref(serverId: string, section: ServerSection): string {
	return `/dashboard/servers/${serverId}${section.path}`;
}

/**
 * The server id in the current path, or null when we are not inside one.
 *
 * Read from the URL rather than passed down, so every page gets the contextual
 * navigation without each one having to remember to provide it.
 */
export function serverIdFromPath(pathname: string): string | null {
	const match = pathname.match(/^\/dashboard\/servers\/([^/]+)/);
	const id = match?.[1];
	if (!id) return null;

	// /servers/new is the connect form, not a server.
	return id === 'new' ? null : id;
}

/** Which section of a server the path is in, for highlighting. */
export function activeSection(pathname: string, serverId: string): ServerSection | null {
	const rest = pathname.slice(`/dashboard/servers/${serverId}`.length);

	// Longest path first, so /tools/postgres matches /tools rather than ''.
	const ordered = [...serverSections].sort((a, b) => b.path.length - a.path.length);
	return ordered.find((section) => rest === section.path || rest.startsWith(section.path + '/')) ?? null;
}
