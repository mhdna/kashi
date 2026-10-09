import { computed, watch } from 'vue'
import { API_BASE } from '@/config'
import { useAuthStore } from '@/stores/auth'
import { authFetch } from './useApi'

// The top-bar identity badge: the signed-in user's initials on a colour.
// The colour defaults to a value derived from the name (so it is stable
// without any setup) and can be picked per user from Settings.
export const AVATAR_COLORS = [
  '#F44336', '#E91E63', '#9C27B0', '#673AB7',
  '#3F51B5', '#2196F3', '#00897B', '#43A047',
  '#FB8C00', '#FF5722', '#6D4C41', '#546E7A',
]

// Two letters: first and last name's initials ("Lara Hawili" -> "LH"), so
// lists that show only the first name still tell two Laras apart; a single
// name gives its first two letters ("Lama" -> "LA").
export function initialsOf (name) {
  const words = String(name || '').trim().split(/\s+/).filter(Boolean)
  if (words.length === 0) {return '?'}
  if (words.length === 1) {return words[0].slice(0, 2).toUpperCase()}
  return (words[0][0] + words.at(-1)[0]).toUpperCase()
}

// Small deterministic string hash -> palette index, so a given name always
// lands on the same colour.
export function hashColor (seed) {
  const s = String(seed || '')
  let h = 0
  for (let i = 0; i < s.length; i++) {
    h = (h * 31 + s.codePointAt(i)) >>> 0
  }
  return AVATAR_COLORS[h % AVATAR_COLORS.length]
}

// The colour picked in Settings is saved on the user in kashi
// (users.avatar_color), so every browser and every viewer sees the same one.
// It arrives with the signed-in user (login, /users/me), with presence, and
// with each IT Management row.
//
// Before that it lived in this browser's localStorage under this key; a
// colour still there and not yet on the server is uploaded once, so an old
// choice carries over.
function legacyKey (user) {
  return `avatarColor:${user?.id ?? user?.name ?? 'anon'}`
}

async function saveColor (auth, value) {
  const res = await authFetch(`${API_BASE}/users/me/avatar_color`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ color: value }),
  })
  if (!res.ok) {
    throw new Error('Could not save the avatar colour.')
  }
  if (auth.user) {
    auth.user = { ...auth.user, avatar_color: value }
    try {
      localStorage.setItem('user', JSON.stringify(auth.user))
    } catch { /* the in-memory user is updated either way */ }
  }
}

let migrated = ''
function migrateLegacy (auth) {
  const user = auth.user
  // A user cached from before avatar_color existed has no such field yet;
  // wait for the fresh one from /users/me rather than overwrite a colour
  // picked in another browser.
  if (!user?.id || !('avatar_color' in user) || migrated === String(user.id)) {
    return
  }
  migrated = String(user.id)
  let legacy = null
  try {
    legacy = localStorage.getItem(legacyKey(user))
  } catch { /* storage unavailable: nothing to carry over */ }
  if (!legacy) {
    return
  }
  const done = () => {
    try {
      localStorage.removeItem(legacyKey(user))
    } catch { /* nothing to clean */ }
  }
  if (user.avatar_color) {
    done()
    return
  }
  saveColor(auth, legacy).then(done).catch(() => { migrated = '' })
}

export function useUserAvatar () {
  const auth = useAuthStore()

  migrateLegacy(auth)
  watch(() => auth.user, () => migrateLegacy(auth))

  const initials = computed(() => initialsOf(auth.user?.name))
  const defaultColor = computed(() => hashColor(auth.user?.name ?? auth.user?.id))
  const color = computed(() => auth.user?.avatar_color || defaultColor.value)
  const isCustom = computed(() => !!auth.user?.avatar_color)

  function setColor (value) {
    return saveColor(auth, value)
  }

  function resetColor () {
    return saveColor(auth, '')
  }

  return { initials, color, defaultColor, isCustom, setColor, resetColor, colors: AVATAR_COLORS }
}

// The colour to show for any user row that carries avatar_color (presence,
// IT Management): theirs if they picked one, else the one from their name.
export function colorForUser (user) {
  return user?.avatar_color || hashColor(user?.name ?? user?.user_name)
}
