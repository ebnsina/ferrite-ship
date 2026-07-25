import { env } from '$config/env';
import { AppError, codeForStatus, describe, toAppError } from '$lib/errors';

const DEFAULT_TIMEOUT_MS = 15_000;

/** Error envelope the Go control plane returns for every non-2xx response. */
interface ApiErrorEnvelope {
	code?: string;
	message?: string;
	request_id?: string;
}

export interface ApiRequestOptions {
	method?: 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE';
	body?: unknown;
	signal?: AbortSignal;
	timeoutMs?: number;
}

function buildUrl(path: string): string {
	if (!path.startsWith('/')) {
		throw new TypeError(`API path must start with "/", received "${path}"`);
	}
	return `${env.apiBaseUrl}${path}`;
}

async function readEnvelope(response: Response): Promise<ApiErrorEnvelope> {
	try {
		const parsed: unknown = await response.json();
		return parsed && typeof parsed === 'object' ? (parsed as ApiErrorEnvelope) : {};
	} catch {
		// A proxy or gateway returned HTML instead of our JSON envelope.
		return {};
	}
}

async function toResponseError(response: Response): Promise<AppError> {
	const code = codeForStatus(response.status);
	const copy = describe(code);
	const envelope = await readEnvelope(response);

	return new AppError({
		code,
		status: response.status,
		// Prefer the server's message only when it is meant for humans.
		message: envelope.message?.trim() || copy.message,
		action: copy.action,
		requestId: envelope.request_id
	});
}

/**
 * Single entry point for every control-plane call. Guarantees the caller sees
 * either parsed data or an AppError — never a raw fetch rejection, an unparsed
 * body, or a silently-ignored non-2xx response.
 */
export async function apiRequest<T>(path: string, options: ApiRequestOptions = {}): Promise<T> {
	const { method = 'GET', body, signal, timeoutMs = DEFAULT_TIMEOUT_MS } = options;

	const timeout = AbortSignal.timeout(timeoutMs);
	const combined = signal ? AbortSignal.any([signal, timeout]) : timeout;

	let response: Response;
	try {
		response = await fetch(buildUrl(path), {
			method,
			signal: combined,
			credentials: 'include',
			headers: body ? { 'content-type': 'application/json' } : undefined,
			body: body === undefined ? undefined : JSON.stringify(body)
		});
	} catch (cause) {
		throw toAppError(cause, 'network');
	}

	if (!response.ok) {
		throw await toResponseError(response);
	}

	if (response.status === 204) {
		return undefined as T;
	}

	try {
		return (await response.json()) as T;
	} catch (cause) {
		throw toAppError(cause, 'parse');
	}
}
