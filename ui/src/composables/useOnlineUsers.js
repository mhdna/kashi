import { ref } from 'vue'
import { API_BASE } from '@/config'
import { useAdminSocket } from './useAdminSocket'
import { authFetch } from './useApi'

// Module-scope so the top bar and any other consumer share one list instead
// of each fetching and subscribing on its own.
const users = ref([])
// False until the first list arrives (fetch or push), so the top bar can
// hold the avatar's place with a skeleton instead of popping it in.
const loaded = ref(false)
let started = false

// Who is signed in right now: an initial fetch, then kept current by the
// "admin_presence_changed" push over the admin WebSocket -- same
// poll-then-push pattern as the connected-branches pill. The push carries
// the full list, so a dropped one is corrected by the next.
export function useOnlineUsers () {
  const { ensureConnected, onMessage } = useAdminSocket()

  async function start () {
    if (started) {return}
    started = true
    onMessage(message => {
      if (message.type === 'admin_presence_changed') {
        users.value = message.users ?? []
        loaded.value = true
      }
    })
    ensureConnected()
    try {
      const res = await authFetch(`${API_BASE}/users/online`)
      if (res.ok) {
        users.value = (await res.json()).users ?? []
      }
    } catch {
      // Informational only -- the push still fills the list in.
    } finally {
      // Even on failure: a skeleton that never resolves is worse than none.
      loaded.value = true
    }
  }

  return { users, loaded, start }
}
