function hostnameMatchesAllowedDomain(hostname, allowedDomain) {
  const normalizedHostname = String(hostname || '').trim().toLowerCase()
  const normalizedAllowedDomain = String(allowedDomain || '')
    .trim()
    .toLowerCase()

  if (!normalizedHostname || !normalizedAllowedDomain) {
    return false
  }

  return (
    normalizedHostname === normalizedAllowedDomain ||
    normalizedHostname.endsWith(`.${normalizedAllowedDomain}`)
  )
}

function getAllowedHosts(value) {
  return String(value || '')
    .split(',')
    .map((host) => host.trim().toLowerCase())
    .filter(Boolean)
}

export default {
  async fetch(request, env) {
    const requestUrl = new URL(request.url)

    if (request.method === 'GET' && requestUrl.pathname === '/health') {
      return Response.json({
        ok: true,
        service: 'api-worker-proxy',
      })
    }

    if (request.method !== 'POST') {
      return new Response('Method Not Allowed', { status: 405 })
    }

    let payload
    try {
      payload = await request.json()
    } catch {
      return new Response('Invalid JSON', { status: 400 })
    }

    const { url, key, method = 'GET', headers = {}, body } = payload || {}

    if (!key || key !== env.WORKER_ACCESS_KEY) {
      return new Response('Unauthorized', { status: 401 })
    }

    if (!url || typeof url !== 'string') {
      return new Response('Missing url', { status: 400 })
    }

    let target
    try {
      target = new URL(url)
    } catch {
      return new Response('Invalid url', { status: 400 })
    }

    if (!['http:', 'https:'].includes(target.protocol)) {
      return new Response('Unsupported protocol', { status: 400 })
    }

    const allowHttp = env.ALLOW_HTTP === 'true'
    if (target.protocol === 'http:' && !allowHttp) {
      return new Response('HTTP not allowed', { status: 403 })
    }

    const allowedHosts = getAllowedHosts(env.ALLOWED_HOSTS)
    if (
      allowedHosts.length > 0 &&
      !allowedHosts.some((allowedHost) =>
        hostnameMatchesAllowedDomain(target.hostname, allowedHost)
      )
    ) {
      return new Response('Host not allowed', { status: 403 })
    }

    const upperMethod = String(method || 'GET').toUpperCase()
    const outgoingHeaders = new Headers(headers || {})
    const init = {
      method: upperMethod,
      headers: outgoingHeaders,
      redirect: 'follow',
    }

    if (upperMethod !== 'GET' && upperMethod !== 'HEAD' && body !== undefined) {
      if (typeof body === 'string') {
        init.body = body
      } else {
        if (!outgoingHeaders.has('content-type')) {
          outgoingHeaders.set('content-type', 'application/json')
        }
        init.body = JSON.stringify(body)
      }
    }

    let upstream
    try {
      upstream = await fetch(target.toString(), init)
    } catch (error) {
      return new Response(
        `Upstream fetch failed: ${error?.message || 'error'}`,
        { status: 502 }
      )
    }

    const responseHeaders = new Headers(upstream.headers)
    responseHeaders.delete('content-encoding')
    responseHeaders.delete('content-length')

    return new Response(upstream.body, {
      status: upstream.status,
      statusText: upstream.statusText,
      headers: responseHeaders,
    })
  },
}
