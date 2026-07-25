import { toAppError, type AppError } from '$lib/errors';

export interface Resource<T> {
	readonly data: T | null;
	readonly error: AppError | null;
	readonly loading: boolean;
	reload(): void;
}

/**
 * Wraps an async read in loading / error / data state, so no view has to
 * hand-roll it and no failure can be silently swallowed. In-flight requests are
 * aborted when superseded or when the owning component is destroyed.
 *
 * Must be called during component initialisation (it registers an $effect).
 */
export function createResource<T>(load: (signal: AbortSignal) => Promise<T>): Resource<T> {
	let data = $state<T | null>(null);
	let error = $state<AppError | null>(null);
	let loading = $state(true);

	let controller: AbortController | null = null;

	function run(): void {
		controller?.abort();
		const current = new AbortController();
		controller = current;

		loading = true;
		error = null;

		load(current.signal)
			.then((result) => {
				if (current.signal.aborted) return;
				data = result;
				loading = false;
			})
			.catch((cause: unknown) => {
				if (current.signal.aborted) return;
				error = toAppError(cause);
				loading = false;
			});
	}

	run();

	$effect(() => () => controller?.abort());

	return {
		get data() {
			return data;
		},
		get error() {
			return error;
		},
		get loading() {
			return loading;
		},
		reload: run
	};
}
