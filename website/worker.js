/**
 * Cloudflare Worker for VitePress (SSG).
 * Serves hashed assets as-is; maps clean paths like /guide/getting-started
 * to getting-started.html (or .../index.html) so refresh does not 404.
 */
export default {
  async fetch(request, env) {
    const url = new URL(request.url)
    let pathname = url.pathname

    if (pathname !== '/' && pathname.endsWith('/')) {
      pathname = pathname.slice(0, -1)
    }

    const tryFetch = async (path) => {
      const assetUrl = new URL(path, url.origin)
      assetUrl.search = url.search
      const res = await env.ASSETS.fetch(
        new Request(assetUrl.toString(), {
          method: request.method,
          headers: request.headers,
        }),
      )
      return res.status === 200 ? res : null
    }

    if (pathname === '/' || pathname === '/index.html') {
      const home = await tryFetch('/index.html')
      if (home) return home
    }

    const hasFileExtension = /\.\w+$/.test(pathname)
    if (hasFileExtension) {
      const asset = await tryFetch(pathname)
      if (asset) return asset
    } else {
      const candidates = [
        pathname,
        `${pathname}.html`,
        `${pathname}/index.html`,
      ]
      for (const path of candidates) {
        const hit = await tryFetch(path)
        if (hit) return hit
      }
    }

    const notFound = await tryFetch('/404.html')
    if (notFound) {
      return new Response(notFound.body, {
        status: 404,
        headers: notFound.headers,
      })
    }

    return new Response('Not Found', { status: 404 })
  },
}
