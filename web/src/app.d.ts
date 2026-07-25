/// <reference types="@sveltejs/kit" />

declare global {
	namespace App {
		/**
		 * Shape of `page.error`. Extends SvelteKit's default `{ message }` so
		 * error pages can render an actionable code and a support-quotable id.
		 */
		interface Error {
			message: string;
			code?: string;
			requestId?: string;
		}
	}
}

export {};
