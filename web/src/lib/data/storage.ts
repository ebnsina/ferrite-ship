import { apiRequest } from '$lib/api/client';

/** One path and how much it holds. */
export interface StorageDirectory {
	path: string;
	bytes: number;
	/** How deep below the root, so the list can be indented. */
	depth: number;
}

/**
 * Space that can be recovered without losing anything you put there.
 *
 * `bytes` is an upper bound. Docker reports what each item would free on its
 * own, and image layers are shared, so the total is optimistic.
 */
export interface Reclaimable {
	id: string;
	label: string;
	detail: string;
	bytes: number;
}

export interface StorageReport {
	totalBytes: number;
	usedBytes: number;
	freeBytes: number;
	directories: StorageDirectory[];
	reclaimable: Reclaimable[];
}

interface StartedJob {
	id: string;
}

export const storageClient = {
	/**
	 * Inspects the server there and then.
	 *
	 * Given its own long timeout: this walks the whole disk, which takes tens
	 * of seconds on a full one. The default fifteen turned a slow answer into
	 * "we could not reach the control plane", which is both wrong and the
	 * least helpful thing it could have said.
	 */
	report: (serverId: string, signal?: AbortSignal) =>
		apiRequest<StorageReport>(`/v1/servers/${encodeURIComponent(serverId)}/storage`, {
			signal,
			timeoutMs: 120_000
		}),

	reclaim: (serverId: string, items: string[], signal?: AbortSignal) =>
		apiRequest<StartedJob>(`/v1/servers/${encodeURIComponent(serverId)}/storage/reclaim`, {
			method: 'POST',
			body: { items },
			signal
		})
};
