import { env } from '$config/env';
import { apiRequest } from '$lib/api/client';

export interface FileEntry {
	name: string;
	path: string;
	isDir: boolean;
	size: number;
	mode: string;
	modifiedAt: string;
	isSymlink: boolean;
}

export interface DirectoryListing {
	path: string;
	parent: string;
	entries: FileEntry[];
}

export interface FileContent {
	path: string;
	text: string;
	size: number;
	mode: string;
}

function base(serverId: string): string {
	return `/v1/servers/${encodeURIComponent(serverId)}/files`;
}

export const filesClient = {
	list: (serverId: string, path: string, signal?: AbortSignal) =>
		apiRequest<DirectoryListing>(`${base(serverId)}?path=${encodeURIComponent(path)}`, { signal }),

	read: (serverId: string, path: string, signal?: AbortSignal) =>
		apiRequest<FileContent>(`${base(serverId)}/content?path=${encodeURIComponent(path)}`, {
			signal
		}),

	write: (serverId: string, path: string, text: string, signal?: AbortSignal) =>
		apiRequest<void>(`${base(serverId)}/content`, {
			method: 'PUT',
			body: { path, text },
			signal
		}),

	remove: (serverId: string, path: string, signal?: AbortSignal) =>
		apiRequest<void>(`${base(serverId)}?path=${encodeURIComponent(path)}`, {
			method: 'DELETE',
			signal
		}),

	/**
	 * Downloads go through the browser rather than fetch, so the file streams
	 * straight to disk instead of being buffered in memory first.
	 */
	downloadUrl: (serverId: string, path: string) =>
		`${env.apiBaseUrl}${base(serverId)}/download?path=${encodeURIComponent(path)}`
};
