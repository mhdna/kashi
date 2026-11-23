import { ref } from 'vue'
import { API_BASE } from '@/config'
import { useAdminSocket } from './useAdminSocket'
import { authFetch } from './useApi'

// The two switches for the live view of tills (Settings), and which tills are
// watched right now. Module state, so the Settings page and the branches page
// share one copy and a change shows in both.

async function requestJSON (url, options) {
  const res = await authFetch(url, options)
  const body = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(body.error || `Request failed with status ${res.status}`)
  }
  return body
}

const enabled = ref(false)
const controlEnabled = ref(false)
const loaded = ref(false)
/** [{ register_id, branch_id, watchers: [names], virtual? }] kept current by the admin socket. */
const sessions = ref([])

let listening = false
let stopListening = null
function listen () {
  if (listening) {
    return
  }
  listening = true
  const { ensureConnected, onMessage } = useAdminSocket()
  ensureConnected()
  stopListening = onMessage(message => {
    if (message.type === 'live_view_sessions') {
      sessions.value = message.live_sessions ?? []
    }
  })
}

function apply (settings) {
  enabled.value = settings?.live_view_enabled === true
  controlEnabled.value = enabled.value && settings?.live_view_control_enabled === true
  loaded.value = true
}

export function useLiveView () {
  async function load (force = false) {
    if (loaded.value && !force) {
      return
    }
    apply((await requestJSON(`${API_BASE}/app_settings`)).settings)
  }

  /** Saves both switches; control can only be on while view is. */
  async function save ({ enabled: view, controlEnabled: control }) {
    const body = { enabled: view === true, control_enabled: view === true && control === true }
    const data = await requestJSON(`${API_BASE}/app_settings/live_view`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
    apply(data.settings)
  }

  /** Who watches a register right now (names), for the till list. */
  function watchersOf (registerId) {
    return sessions.value.find(s => s.register_id === registerId && !s.virtual)?.watchers ?? []
  }

  /** Who uses a register's separate screen right now (a name), or ''. */
  function separateUserOf (registerId) {
    return sessions.value.find(s => s.register_id === registerId && s.virtual)?.watchers?.[0] ?? ''
  }

  return { enabled, controlEnabled, loaded, sessions, load, save, watchersOf, separateUserOf, listen }
}

/** Test helper: forget everything. */
export function resetLiveViewState () {
  enabled.value = false
  controlEnabled.value = false
  loaded.value = false
  sessions.value = []
  stopListening?.()
  stopListening = null
  listening = false
}
