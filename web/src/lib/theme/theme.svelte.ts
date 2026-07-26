import { browser } from '$app/environment';

export type Theme = 'light' | 'dark';

function readStored(key: string, fallback: Theme): Theme {
	if (!browser) return fallback;
	try {
		const stored = localStorage.getItem(key);
		return stored === 'light' || stored === 'dark' ? stored : fallback;
	} catch {
		// Storage blocked (private mode, blocked cookies) — use the default.
		return fallback;
	}
}

/**
 * A themeable surface. Two exist: the document (marketing, dark by default)
 * and the dashboard subtree (light by default, applied via ThemeScope).
 *
 * Keeping them separate is deliberate — the marketing site and the operations
 * console are read in different contexts and do not have to agree.
 */
class ThemeStore {
	readonly #key: string;
	readonly #appliesToDocument: boolean;

	value = $state<Theme>('dark');

	constructor(key: string, fallback: Theme, appliesToDocument = false) {
		this.#key = key;
		this.#appliesToDocument = appliesToDocument;
		this.value = readStored(key, fallback);
	}

	toggle(): void {
		this.set(this.value === 'dark' ? 'light' : 'dark');
	}

	set(theme: Theme): void {
		this.value = theme;
		if (!browser) return;

		if (this.#appliesToDocument) {
			document.documentElement.dataset.theme = theme;
		}

		try {
			localStorage.setItem(this.#key, theme);
		} catch {
			// The choice simply will not persist across reloads.
		}
	}
}

/** Drives <html data-theme>. Must match the key read by the script in app.html. */
export const siteTheme = new ThemeStore('ferrite-theme', 'dark', true);

/** Drives the dashboard's ThemeScope only. */
export const dashboardTheme = new ThemeStore('ferrite-dashboard-theme', 'light');
