/**
 * Intl formatter construction is expensive; the formatters themselves are
 * immutable and safe to share. Every formatter in this folder goes through here.
 */
const registry = new Map<string, unknown>();

export function cachedFormatter<T>(
	kind: string,
	locale: string | undefined,
	options: object,
	create: () => T
): T {
	const key = `${kind}|${locale ?? '*'}|${JSON.stringify(options)}`;
	const existing = registry.get(key);
	if (existing !== undefined) return existing as T;

	const created = create();
	registry.set(key, created);
	return created;
}

/**
 * `undefined` means "use the runtime's locale", which is what we want in the
 * browser. Callers may pass an explicit locale for tests or user preferences.
 */
export type LocaleOption = string | undefined;
