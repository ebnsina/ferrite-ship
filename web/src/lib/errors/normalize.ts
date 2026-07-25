import { AppError, isAppError, type ErrorCode } from './app-error';

const STATUS_TO_CODE: ReadonlyMap<number, ErrorCode> = new Map([
	[400, 'parse'],
	[401, 'unauthorized'],
	[403, 'forbidden'],
	[404, 'not_found'],
	[408, 'timeout'],
	[409, 'conflict'],
	[429, 'rate_limited']
]);

export function codeForStatus(status: number): ErrorCode {
	const mapped = STATUS_TO_CODE.get(status);
	if (mapped) return mapped;
	if (status >= 500) return 'server';
	if (status >= 400) return 'parse';
	return 'unknown';
}

const COPY: Record<ErrorCode, { message: string; action: string }> = {
	network: {
		message: 'Could not reach the control plane.',
		action: 'Check your connection and try again.'
	},
	timeout: {
		message: 'The request took too long to complete.',
		action: 'Try again in a moment.'
	},
	unauthorized: {
		message: 'Your session has expired.',
		action: 'Sign in again to continue.'
	},
	forbidden: {
		message: 'You do not have access to this resource.',
		action: 'Ask an organisation owner to grant you access.'
	},
	not_found: {
		message: 'We could not find what you were looking for.',
		action: 'Check the address, or head back to the dashboard.'
	},
	conflict: {
		message: 'That change conflicts with the current state.',
		action: 'Reload to see the latest, then try again.'
	},
	rate_limited: {
		message: 'Too many requests.',
		action: 'Wait a moment before retrying.'
	},
	server: {
		message: 'Something broke on our side.',
		action: 'We have been notified. Try again shortly.'
	},
	parse: {
		message: 'The control plane returned an unexpected response.',
		action: 'Try again, or contact support if it persists.'
	},
	config: {
		message: 'The application is misconfigured.',
		action: 'Check the deployment environment variables.'
	},
	unknown: {
		message: 'Something unexpected happened.',
		action: 'Try again, or contact support if it persists.'
	}
};

export function describe(code: ErrorCode): { message: string; action: string } {
	return COPY[code];
}

/**
 * Convert anything catchable into an AppError. Never throws, never leaks an
 * internal message into user-facing copy — the original is kept as `cause`
 * for logging.
 */
export function toAppError(value: unknown, fallbackCode: ErrorCode = 'unknown'): AppError {
	if (isAppError(value)) return value;

	// fetch() rejects with a TypeError when the request never reached the server.
	const code: ErrorCode =
		value instanceof DOMException && value.name === 'AbortError'
			? 'timeout'
			: value instanceof TypeError
				? 'network'
				: fallbackCode;

	const copy = describe(code);
	return new AppError({ code, message: copy.message, action: copy.action, cause: value });
}
