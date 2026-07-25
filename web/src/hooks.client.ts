import { toAppError } from '$lib/errors';
import type { HandleClientError } from '@sveltejs/kit';

/**
 * Last line of defence: anything thrown during navigation or rendering that no
 * component caught lands here. It must never throw, and must never put an
 * internal message in front of a user — the original is logged, the returned
 * shape is what +error.svelte renders.
 */
export const handleClientError: HandleClientError = ({ error, status, message }) => {
	const appError = toAppError(error);

	console.error('[ferrite-ship] unhandled client error', {
		status,
		message,
		code: appError.code,
		cause: appError.cause ?? error
	});

	return {
		message: appError.message,
		code: appError.code
	};
};

export { handleClientError as handleError };
