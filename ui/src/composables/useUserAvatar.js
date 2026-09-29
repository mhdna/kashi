import { computed, ref, watch } from 'vue'
import { useAuthStore } from '@/stores/auth'

// The top-bar identity badge: the signed-in user's initials on a colour.
// The colour defaults to a value derived from the name (so it is stable
// without any setup) and can be overridden per user from Settings. The
// override lives in localStorage -- it is a personal, per-device preference,
// not something the shared app_settings row should carry.
export const AVATAR_COLORS = [
  '#F44336', '#E91E63', '#9C27B0', '#673AB7',
  '#3F51B5', '#2196F3', '#00897B', '#43A047',
  '#FB8C00', '#FF5722', '#6D4C41', '#546E7A',
]

export function initialsOf (name) {
  const clean = String(name || '').trim()
  return clean ? clean.slice(0, 2).toUpperCase() : '?'
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

function storageKey (user) {
  return `avatarColor:${user?.id ?? user?.name ?? 'anon'}`
}

// Module-level so every caller (top bar + Settings) shares one reactive
// value and a change in one place shows up immediately in the other.
const override = ref(null)
let currentKey = ''

function loadOverride (user) {
  const key = storageKey(user)
  if (key === currentKey) {
    return
  }
  currentKey = key
  try {
    override.value = localStorage.getItem(key) || null
  } catch {
    override.value = null
  }
}

export function useUserAvatar () {
  const auth = useAuthStore()

  loadOverride(auth.user)
  watch(() => auth.user, u => loadOverride(u))

  const initials = computed(() => initialsOf(auth.user?.name))
  const defaultColor = computed(() => hashColor(auth.user?.name ?? auth.user?.id))
  const color = computed(() => override.value || defaultColor.value)
  const isCustom = computed(() => !!override.value)

  function setColor (value) {
    override.value = value
    try {
      localStorage.setItem(currentKey, value)
    } catch { /* private mode / storage disabled -- keep the in-memory value */ }
  }

  function resetColor () {
    override.value = null
    try {
      localStorage.removeItem(currentKey)
    } catch { /* nothing to clean up if storage is unavailable */ }
  }

  return { initials, color, defaultColor, isCustom, setColor, resetColor, colors: AVATAR_COLORS }
}
