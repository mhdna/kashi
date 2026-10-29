import { readonly, ref } from 'vue'
import { API_BASE } from '@/config'

// The API is briefly unavailable while it's being redeployed: `make
// deploy-api` restarts kashi-api, and for that moment Caddy is still up but
// answers /api/* with a 502 (503/504 for the same thing behind other
// proxies). authFetch reports those here; the app then shows "updating" and
// polls until the API answers again, so the page comes back on its own.

const GATEWAY_STATUSES = new Set([502, 503, 504])
const POLL_MS = 2000

const updating = ref(false)
const since = ref(0)
let timer = null
const waiters = new Set()

export function isGatewayStatus (status) {
  return GATEWAY_STATUSES.has(status)
}

// A gateway status from the proxy itself, not the API: the API's own errors
// are JSON ({"error": ...}) and never mean it's down -- a WhatsApp bridge
// failure while linking a number used to come back as a 502 and pin the
// overlay up, since the QR poll kept re-raising it.
export function isProxyOutage (res) {
  return isGatewayStatus(res.status)
    && !String(res.headers?.get('Content-Type') ?? '').includes('application/json')
}

// Any real HTTP answer from the API -- even 401 -- means it's back. Plain
// fetch rather than authFetch: no token renewal or logout redirect while
// we're only checking whether the server is there.
async function apiIsUp () {
  try {
    const res = await fetch(`${API_BASE}/app_settings`, { cache: 'no-store' })
    return !isGatewayStatus(res.status)
  } catch {
    return false
  }
}

function startPolling () {
  if (timer) {
    return
  }
  timer = setTimeout(async function tick () {
    if (await apiIsUp()) {
      timer = null
      updating.value = false
      for (const resolve of waiters) {
        resolve()
      }
      waiters.clear()
      return
    }
    timer = setTimeout(tick, POLL_MS)
  }, POLL_MS)
}

export function markServerUpdating () {
  if (!updating.value) {
    updating.value = true
    since.value = Date.now()
  }
  startPolling()
}

// Resolves once the API answers again. Used to recover from a plain
// connection failure (no response at all) the same way as from a 502.
export function waitForServer () {
  return new Promise(resolve => {
    waiters.add(resolve)
    startPolling()
  })
}

export function useServerStatus () {
  return { updating: readonly(updating), since: readonly(since) }
}
