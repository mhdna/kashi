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

function handleClose () {
  status.value = 'closed'
  clearTimeout(reconnectTimer)
  reconnectTimer = setTimeout(connect, 5000)
}

function connect () {
  const authStore = useAuthStore()

  // Don't stomp a confirmed 'closed' back to 'connecting' on the 5s reconnect
  // loop -- consumers show an outage state for 'closed' and we don't want it
  // flickering away on every retry. The initial connect still starts here
  // because status is 'connecting' at module load.
  if (status.value !== 'closed') {
    status.value = 'connecting'
  }
  ws = new WebSocket(`${WS_BASE}/admin/ws?token=${authStore.token}`)

  ws.addEventListener('open', () => {
    status.value = 'open'
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
    connect()
  }

  function onMessage (callback) {
    listeners.add(callback)
    return () => listeners.delete(callback)
  }

  return { status, ensureConnected, onMessage }
}
