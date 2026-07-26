/**
 * One error type for everything the UI renders. Anything catchable is
 * normalised into this before it reaches a component, so error rendering never
 * has to guess at an unknown shape.
 */

/**
 * The shared vocabulary.
 *
 * `internal/apierr` on the Go side is the source of truth; these codes mirror
 * it. The last four can only ever be raised here, because they describe a
 * request that never got an answer — the server cannot describe those.
 */
export type ErrorCode =
	// Answered by the API
	| 'invalid'
	| 'unauthorized'
	| 'forbidden'
	| 'not_found'
	| 'conflict'
	| 'too_large'
	| 'unsupported'
	| 'upstream'
	| 'internal'
	// Raised in the browser
	| 'network'
	| 'timeout'
	| 'parse'
	| 'config';

export interface AppErrorOptions {
	code: ErrorCode;
	status?: number;
	/**
	 * What happened, in words safe to show. For anything the API answered this
	 * is the server's own wording — the browser does not paraphrase it, because
	 * then the same sentence would live in two repositories.
	 */
	message: string;
	/** What to do next, if anything. */
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

/** Codes where trying the same thing again could plausibly work. */
const RETRYABLE_CODES = new Set<ErrorCode>(['network', 'timeout', 'upstream', 'internal']);

export function isAppError(value: unknown): value is AppError {
	return value instanceof AppError;
}
