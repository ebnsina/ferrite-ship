/**
 * The dashboard is an authenticated client of the Go control plane, so there is
 * nothing to prerender and no server to render on — it ships as an SPA served
 * from the adapter-static fallback.
 */
export const prerender = false;
export const ssr = false;
