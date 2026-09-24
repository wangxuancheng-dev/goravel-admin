/**
 * Cloudflare Worker for SPA routing + same-origin WebSocket proxy.
 *
 * Vanity/custom domains (e.g. acme2.example.xyz) cannot open cross-site WSS to
 * api.example.top; the SPA uses wss://<page-host>/ws/... instead. This Worker
 * upgrades /ws to the API origin so the browser stays same-origin.
 *
 * Set [vars] API_ORIGIN in wrangler.toml (e.g. https://api.example.com).
 */
export default {
  async fetch(request, env) {
    const url = new URL(request.url);
    const pathname = url.pathname;

    // Proxy notification WebSocket (and any future /ws/*) to the API host.
    if (pathname === '/ws' || pathname.startsWith('/ws/')) {
      const apiOrigin = String(env.API_ORIGIN || '').replace(/\/+$/, '');
      if (!apiOrigin) {
        return new Response('API_ORIGIN is not configured', { status: 502 });
      }
      const target = new URL(pathname + url.search, apiOrigin);
      return fetch(target, request);
    }

    // Direct requests to index.html or root should be served as-is
    if (pathname === '/index.html' || pathname === '/') {
      const indexRequest = new Request(new URL('/index.html', url.origin).toString(), {
        method: request.method,
        headers: request.headers,
      });
      return env.ASSETS.fetch(indexRequest);
    }

    // Check if the request is for a static asset (has file extension)
    const hasFileExtension = /\.\w+$/.test(pathname);

    // If it's a static asset request, try to fetch it
    if (hasFileExtension) {
      const asset = await env.ASSETS.fetch(request);
      // If asset exists, return it; otherwise continue to serve index.html
      if (asset.status === 200) {
        return asset;
      }
    }

    // For non-file requests (SPA routes), serve index.html
    // This handles all SPA routes like /operation-logs, /login, /admins, etc.
    const indexRequest = new Request(new URL('/index.html', url.origin).toString(), {
      method: request.method,
      headers: request.headers,
    });

    return env.ASSETS.fetch(indexRequest);
  }
};
