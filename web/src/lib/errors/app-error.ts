/**
 * A single error type for everything the UI has to render. Anything thrown
 * inside the app is normalised into this before it reaches a component, so
 * error rendering never has to guess at an unknown shape.
 */

export type ErrorCode =
	| 'network'
	| 'timeout'
	| 'unauthorized'
	| 'forbidden'
	| 'not_found'
	| 'conflict'
	| 'rate_limited'
	| 'server'
	| 'parse'
	| 'config'
	| 'unknown';

export interface AppErrorOptions {
	code: ErrorCode;
	status?: number;
	/** Safe to show a user. Never contains internals or secrets. */
	message: string;
	/** What the user can actually do next, if anything. */
	action?: string;
	/** Correlation id from the API, quotable in a support request. */
	requestId?: string;
	cause?: unknown;
	/** Whether retrying the same operation could plausibly succeed. */
	retryable?: boolean;
}

export class AppError extends Error {
	override readonly name = 'AppError';
	readonly code: ErrorCode;
	readonly status: number | undefined;
	readonly action: string | undefined;
	readonly requestId: string | undefined;
	readonly retryable: boolean;

	constructor(options: AppErrorOptions) {
		super(options.message, { cause: options.cause });
		this.code = options.code;
		this.status = options.status;
		this.action = options.action;
		this.requestId = options.requestId;
		this.retryable = options.retryable ?? RETRYABLE_CODES.has(options.code);
	}
}

const RETRYABLE_CODES = new Set<ErrorCode>(['network', 'timeout', 'rate_limited', 'server']);

export function isAppError(value: unknown): value is AppError {
	return value instanceof AppError;
}
