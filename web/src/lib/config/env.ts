import {
	PUBLIC_API_BASE_URL,
	PUBLIC_APP_ENV,
	PUBLIC_APP_URL,
	PUBLIC_DATA_SOURCE
} from '$env/static/public';

/**
 * Validated public configuration.
 *
 * Deliberately has no defaults and no fallbacks: a missing or malformed value
 * throws at module load, which fails the build (or the very first import in the
 * browser) instead of silently running against the wrong control plane.
 */

export type AppEnvironment = 'development' | 'staging' | 'production';
export type DataSource = 'api' | 'mock';

const APP_ENVIRONMENTS: readonly AppEnvironment[] = ['development', 'staging', 'production'];
const DATA_SOURCES: readonly DataSource[] = ['api', 'mock'];

export class EnvConfigError extends Error {
	override readonly name = 'EnvConfigError';

	constructor(variable: string, reason: string) {
		super(
			`Invalid environment configuration: ${variable} ${reason}. ` +
				`See .env.example for the required variables.`
		);
	}
}

function requireUrl(variable: string, value: string | undefined): string {
	if (!value || value.trim() === '') {
		throw new EnvConfigError(variable, 'is required but was empty');
	}

	let parsed: URL;
	try {
		parsed = new URL(value);
	} catch {
		throw new EnvConfigError(variable, `is not a valid absolute URL (received "${value}")`);
	}

	if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
		throw new EnvConfigError(variable, `must use http or https (received "${parsed.protocol}")`);
	}

	// Normalise so callers can always join with a leading-slash path.
	return value.replace(/\/+$/, '');
}

function requireOneOf<T extends string>(
	variable: string,
	value: string | undefined,
	allowed: readonly T[]
): T {
	if (!value || value.trim() === '') {
		throw new EnvConfigError(variable, 'is required but was empty');
	}
	if (!allowed.includes(value as T)) {
		throw new EnvConfigError(variable, `must be one of ${allowed.join(', ')} (received "${value}")`);
	}
	return value as T;
}

export const env = Object.freeze({
	apiBaseUrl: requireUrl('PUBLIC_API_BASE_URL', PUBLIC_API_BASE_URL),
	appUrl: requireUrl('PUBLIC_APP_URL', PUBLIC_APP_URL),
	appEnv: requireOneOf('PUBLIC_APP_ENV', PUBLIC_APP_ENV, APP_ENVIRONMENTS),
	/** Which repository implementation backs the dashboard. See $lib/data. */
	dataSource: requireOneOf('PUBLIC_DATA_SOURCE', PUBLIC_DATA_SOURCE, DATA_SOURCES)
});

export const isProduction = env.appEnv === 'production';
