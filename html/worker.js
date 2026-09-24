/**
 * Cloudflare Worker for SPA routing + same-origin WebSocket proxy.
 *
 * Vanity/custom domains cannot open cross-site WSS to the API host; the SPA uses
 * wss://<page-host>/ws/... . This Worker bridges /ws to API_ORIGIN.
 *
 * Set [vars] API_ORIGIN in wrangler.toml (e.g. https://api.example.com).
 */
export default {
  async fetch(request, env) {
    const url = new URL(request.url);
    const pathname = url.pathname;

    if (pathname === '/ws' || pathname.startsWith('/ws/')) {
      return proxyWs(request, env, url);
    }

    if (pathname === '/index.html' || pathname === '/') {
      const indexRequest = new Request(new URL('/index.html', url.origin).toString(), {
        method: request.method,
        headers: request.headers,
      });
      return env.ASSETS.fetch(indexRequest);
    }

    const hasFileExtension = /\.\w+$/.test(pathname);
    if (hasFileExtension) {
      const asset = await env.ASSETS.fetch(request);
      if (asset.status === 200) {
        return asset;
      }
    }

    const indexRequest = new Request(new URL('/index.html', url.origin).toString(), {
      method: request.method,
      headers: request.headers,
    });
    return env.ASSETS.fetch(indexRequest);
  }
};

async function proxyWs(request, env, url) {
  const apiOrigin = String(env.API_ORIGIN || '').replace(/\/+$/, '');
  if (!apiOrigin) {
    return new Response('API_ORIGIN is not configured', { status: 502 });
  }

  const target = new URL(url.pathname + url.search, apiOrigin);
  const upgrade = (request.headers.get('Upgrade') || '').toLowerCase();

  if (upgrade !== 'websocket') {
    const resp = await fetch(new Request(target.toString(), request));
    const headers = new Headers(resp.headers);
    headers.set('X-Ws-Proxy', 'http');
    return new Response(resp.body, { status: resp.status, statusText: resp.statusText, headers });
  }

  const backendHeaders = {
    Upgrade: 'websocket',
    Connection: 'Upgrade',
  };
  const origin = request.headers.get('Origin');
  if (origin) backendHeaders.Origin = origin;
  const proto = request.headers.get('Sec-WebSocket-Protocol');
  if (proto) backendHeaders['Sec-WebSocket-Protocol'] = proto;
  const ua = request.headers.get('User-Agent');
  if (ua) backendHeaders['User-Agent'] = ua;

  let backend;
  try {
    backend = await fetch(target.toString(), { headers: backendHeaders });
  } catch (err) {
    return new Response('API websocket fetch failed: ' + String(err), {
      status: 502,
      headers: { 'X-Ws-Proxy': 'fetch-error' },
    });
  }

  if (!backend.webSocket) {
    const text = await backend.text();
    return new Response(text || 'backend did not accept websocket', {
      status: backend.status >= 400 ? backend.status : 502,
      headers: {
        'Content-Type': backend.headers.get('Content-Type') || 'text/plain',
        'X-Ws-Proxy': 'no-websocket',
        'X-Ws-Backend-Status': String(backend.status),
      },
    });
  }

  const apiWs = backend.webSocket;
  const pair = new WebSocketPair();
  const client = pair[0];
  const server = pair[1];

  apiWs.accept({ allowHalfOpen: true });
  server.accept({ allowHalfOpen: true });
  pipeWs(server, apiWs);
  pipeWs(apiWs, server);

  return new Response(null, {
    status: 101,
    webSocket: client,
    headers: { 'X-Ws-Proxy': 'bridged' },
  });
}

function pipeWs(from, to) {
  from.addEventListener('message', (event) => {
    try {
      to.send(event.data);
    } catch (_) {
      /* closed */
    }
  });
  from.addEventListener('close', (event) => {
    try {
      to.close(event.code, event.reason);
    } catch (_) {
      /* closed */
    }
  });
  from.addEventListener('error', () => {
    try {
      to.close(1011, 'proxy error');
    } catch (_) {
      /* closed */
    }
  });
}
