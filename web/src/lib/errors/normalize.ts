import { AppError, isAppError, type ErrorCode } from './app-error';

/**
 * Wording for the four failures the API cannot describe, because in each case
 * it never answered. Everything else arrives with the server's own message and
 * action, and is shown as sent — the catalogue in `internal/apierr` is the
 * single source of truth for that copy.
 */
const LOCAL_COPY: Record<'network' | 'timeout' | 'parse' | 'config', { message: string; action: string }> =
	{
		network: {
			message: 'We could not reach the control plane.',
			action: 'Check your connection, and that the server is running.'
		},
		timeout: {
			message: 'That took too long to come back.',
			action: 'Try again in a moment.'
		},
		parse: {
			message: 'The control plane sent something we could not read.',
			action: 'Try again. If it keeps happening it is a bug worth reporting.'
		},
		config: {
			message: 'This app is not set up correctly.',
			action: 'Check the environment variables it was built with.'
		}
	};

/**
 * Convert anything catchable into an AppError. Never throws, and never puts an
 * internal message in front of a person — the original is kept as `cause` for
 * the console.
 */
export function toAppError(value: unknown, fallback: ErrorCode = 'internal'): AppError {
	if (isAppError(value)) return value;

	// fetch() rejects with a TypeError when the request never reached the
	// server, and with an AbortError when it ran out of time.
	const code: ErrorCode =
		value instanceof DOMException && value.name === 'AbortError'
			? 'timeout'
			: value instanceof TypeError
				? 'network'
				: fallback;

	if (code === 'network' || code === 'timeout' || code === 'parse' || code === 'config') {
		const copy = LOCAL_COPY[code];
		return new AppError({ code, message: copy.message, action: copy.action, cause: value });
	}

	// Something threw that is neither an AppError nor a recognisable transport
	// failure. Say so plainly rather than surfacing its internals.
	return new AppError({
		code: 'internal',
		message: 'Something unexpected happened.',
		action: 'Try again. If it keeps happening it is a bug worth reporting.',
		cause: value
	});
}

/** Copy for a transport failure, used where no error object exists yet. */
export function localError(code: keyof typeof LOCAL_COPY, cause?: unknown): AppError {
	const copy = LOCAL_COPY[code];
	return new AppError({ code, message: copy.message, action: copy.action, cause });
}
