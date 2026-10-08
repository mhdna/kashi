import { ref } from 'vue'
import { WS_BASE } from '@/config'
import { useAuthStore } from '@/stores/auth'

// Module-scope (not inside useAdminSocket) so every component that calls
// this composable shares one connection instead of each opening its own --
// SyncCard and the branch-activity toast queue both need the same stream of
// admin pushes, and there's no reason for a browser tab to hold two sockets
// to the same endpoint.
const status = ref('connecting')
const listeners = new Set()
let ws = null
let reconnectTimer = null

// Reconnect backoff: 2s, 4s, 8s ... capped at a minute, back to 2s once a
// connection opens. A fixed 5s retry hammered the server for as long as it
// was down (or the token was stale) from every open tab.
const RETRY_MIN_MS = 2000
const RETRY_MAX_MS = 60_000
let retryDelay = RETRY_MIN_MS

function handleMessage (event) {
  let message
  try {
    message = JSON.parse(event.data)
  } catch {
    return // Ignore malformed pushes -- consumers have their own poll fallback.
  }
  for (const listener of listeners) {
    listener(message)
  }
}

function scheduleReconnect () {
  clearTimeout(reconnectTimer)
  // Logged out (or the token was cleared after a 401): nothing to connect
  // as. The next ensureConnected() after logging in starts it again.
  if (!useAuthStore().token) {
    return
  }
  reconnectTimer = setTimeout(connect, retryDelay)
  retryDelay = Math.min(retryDelay * 2, RETRY_MAX_MS)
}

function handleClose () {
  status.value = 'closed'
  scheduleReconnect()
}

// Back online after a network drop: retry now instead of waiting out the
// backoff.
if (typeof window !== 'undefined') {
  window.addEventListener('online', () => {
    if (status.value !== 'closed') {
      return
    }
    retryDelay = RETRY_MIN_MS
    clearTimeout(reconnectTimer)
    connect()
  })
}

function connect () {
  const authStore = useAuthStore()

  // Don't stomp a confirmed 'closed' back to 'connecting' on the reconnect
  // loop -- consumers show an outage state for 'closed' and we don't want it
  // flickering away on every retry. The initial connect still starts here
  // because status is 'connecting' at module load.
  if (status.value !== 'closed') {
    status.value = 'connecting'
  }
  ws = new WebSocket(`${WS_BASE}/admin/ws?token=${authStore.token}`)

  ws.addEventListener('open', () => {
    status.value = 'open'
    retryDelay = RETRY_MIN_MS
  })
  ws.addEventListener('message', handleMessage)
  ws.addEventListener('close', handleClose)
  ws.addEventListener('error', () => {
    ws.close()
  })
}

export function useAdminSocket () {
  function ensureConnected () {
    if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
      return
    }
    clearTimeout(reconnectTimer)
    retryDelay = RETRY_MIN_MS
    connect()
  }

  function onMessage (callback) {
    listeners.add(callback)
    return () => listeners.delete(callback)
  }

  return { status, ensureConnected, onMessage }
}
