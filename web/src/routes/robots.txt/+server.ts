import { env } from '$config/env';

export const prerender = true;

/**
 * The dashboard is behind a login and rendered client-side; there is nothing
 * for a crawler there, and its URLs leak organisation structure.
 */
export function GET(): Response {
	const body = [
		'User-agent: *',
		'Allow: /',
		'Disallow: /dashboard',
		'',
		`Sitemap: ${env.appUrl}/sitemap.xml`,
		''
	].join('\n');

	return new Response(body, {
		headers: {
			'content-type': 'text/plain; charset=utf-8',
			'cache-control': 'public, max-age=3600'
		}
	});
}
