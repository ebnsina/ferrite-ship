import { env } from '$config/env';

export const prerender = true;

/** Only public, indexable routes belong here. */
const routes = ['/'];

export function GET(): Response {
	const urls = routes
		.map(
			(route) =>
				`\t<url>\n\t\t<loc>${env.appUrl}${route === '/' ? '' : route}/</loc>\n` +
				`\t\t<changefreq>weekly</changefreq>\n\t\t<priority>1.0</priority>\n\t</url>`
		)
		.join('\n');

	const body =
		`<?xml version="1.0" encoding="UTF-8"?>\n` +
		`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n${urls}\n</urlset>\n`;

	return new Response(body, {
		headers: {
			'content-type': 'application/xml; charset=utf-8',
			'cache-control': 'public, max-age=3600'
		}
	});
}
