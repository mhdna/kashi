import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { authFetch, connectionErrorMessage, SERVER_UPDATING_MESSAGE } from '@/composables/useApi'
import { isGatewayStatus } from '@/composables/useServerStatus'
import { API_BASE } from '@/config'

// Turn a login response status into a message the user can act on. The
// backend leaks raw Go errors, so nothing here relies on its body text.
function loginMessage (status, bodyError) {
  switch (status) {
    case 400: {
      return 'Enter a valid username and password.'
    }
    // 404 is an unknown username; same message so it doesn't reveal which
    // usernames exist.
    case 401:
    case 403:
    case 404: {
      return 'Incorrect username or password.'
    }
    case 423: {
      return 'This account is locked. Try again later or contact an administrator.'
    }
    case 429: {
      return bodyError || 'Too many attempts. Please wait a while and try again.'
    }
    default: {
      // Only a 5xx is actually the server's fault; anything else unexpected
      // gets a neutral message. The status is shown so a report is traceable.
      if (isGatewayStatus(status)) {
        return SERVER_UPDATING_MESSAGE
      }
      if (status >= 500) {
        return `The server couldn't sign you in (error ${status}). Try again; if it keeps happening, tell an administrator.`
      }
      return `Couldn't sign you in (error ${status}). Check your username and password and try again.`
    }
  }
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const refreshToken = ref(localStorage.getItem('refreshToken') || '')
  const user = ref(JSON.parse(localStorage.getItem('user') || 'null'))
  // What the user's role allows:
  //   { role_name, is_admin, sections: { key: { view, edit } }, pages: { key: { view, edit } } }
  // A section's level is the highest of its pages; the API enforces sections
  // and the UI additionally hides pages (api/access.go).
  const access = ref(JSON.parse(localStorage.getItem('access') || 'null'))

  // Tabs share the sign-in through localStorage but each keeps its own copy
  // in memory, and its own admin socket -- which is what puts a user in the
  // app bar's online list. So when another tab logs out, or signs in as
  // someone else, this tab follows by reloading into that state rather than
  // staying signed in (and online) as the old user. A token refresh keeps
  // the same user, so it's left alone.
  window.addEventListener('storage', event => {
    if (event.key !== 'user' && event.key !== 'token' && event.key !== null) {
      return
    }
    const stored = JSON.parse(localStorage.getItem('user') || 'null')
    const signedIn = Boolean(localStorage.getItem('token'))
    if ((stored?.id ?? null) !== (user.value?.id ?? null) || signedIn !== Boolean(token.value)) {
      window.location.reload()
    }
  })

  const isAuthenticated = computed(() => !!token.value)
  const isAdmin = computed(() => !!access.value?.is_admin)

  function setAccess (value) {
    access.value = value
    localStorage.setItem('access', JSON.stringify(value))
    applyTextSelection()
  }

  // Edit implies view (the server normalises it, this just mirrors that).
  function canView (section) {
    return isAdmin.value || !!(access.value?.sections?.[section]?.view || access.value?.sections?.[section]?.edit)
  }

  function canEdit (section) {
    return isAdmin.value || !!access.value?.sections?.[section]?.edit
  }

  // Per page. A cached access from before per-page roles has no `pages`, so
  // fall back to the section until refreshAccess replaces it.
  function canViewPage (section, page) {
    if (!access.value?.pages) {return canView(section)}
    return isAdmin.value || !!access.value.pages[page]?.view
  }

  function canEditPage (section, page) {
    if (!access.value?.pages) {return canEdit(section)}
    return isAdmin.value || !!access.value.pages[page]?.edit
  }

  // A route's `meta` (router/index.js): no section means open to everyone.
  function canViewRoute ({ section, page } = {}) {
    if (!section) {return true}
    return page ? canViewPage(section, page) : canView(section)
  }

  function canEditRoute ({ section, page } = {}) {
    if (!section) {return true}
    return page ? canEditPage(section, page) : canEdit(section)
  }

  // Text selection is off by default (see global.scss) and turned on per role
  // in Settings > Roles ("Text Selection"), like any other section -- it just
  // doesn't gate an API route.
  function applyTextSelection () {
    document.documentElement.classList.toggle('text-selection-enabled', canView('text_selection'))
  }
  applyTextSelection()

  // Re-read the role from the server so a change made in Settings > Roles shows
  // up on the next page load without logging out. Failures keep the cached
  // value: the API rejects anything the role no longer allows regardless.
  async function refreshAccess () {
    try {
      const res = await authFetch(`${API_BASE}/users/me`)
      if (!res.ok) {return}
      const data = await res.json()
      user.value = data.user
      localStorage.setItem('user', JSON.stringify(data.user))
      setAccess(data.access)
    } catch {
      // keep cached access
    }
  }

  async function login (name, password) {
    let res
    try {
      res = await fetch(`${API_BASE}/users/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username: name, password }),
      })
    } catch {
      // fetch only rejects on network-level failures (server unreachable,
      // DNS, offline) -- never on HTTP error status. Same message authFetch
      // uses so the "server is down" / "no internet" states stay consistent.
      throw new Error(connectionErrorMessage())
    }

    if (!res.ok) {
      // The API returns raw Go error strings in `body.error` (bcrypt
      // mismatches, DB dial failures, ...), so map by status instead of
      // trusting the body. Only 429's body is safe enough to pass through
      // since it may carry a specific lockout duration.
      const body = await res.json().catch(() => ({}))
      const err = new Error(loginMessage(res.status, body.error))
      err.status = res.status
      throw err
    }

    const data = await res.json()
    token.value = data.access_token
    refreshToken.value = data.refresh_token
    user.value = data.user
    localStorage.setItem('token', token.value)
    localStorage.setItem('refreshToken', refreshToken.value)
    localStorage.setItem('user', JSON.stringify(user.value))
    setAccess(data.access)
    return true
  }

  async function renew () {
    if (!refreshToken.value) {
      throw new Error('No refresh token')
    }

    const res = await fetch(`${API_BASE}/tokens/renew_access`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken.value }),
    })

    if (!res.ok) {
      throw new Error('Session expired')
    }

    const data = await res.json()
    token.value = data.access_token
    localStorage.setItem('token', token.value)
    return token.value
  }

  function logout () {
    token.value = ''
    refreshToken.value = ''
    user.value = null
    access.value = null
    localStorage.removeItem('access')
    localStorage.removeItem('token')
    localStorage.removeItem('refreshToken')
    localStorage.removeItem('user')
    applyTextSelection()
  }

  return { token, refreshToken, user, access, isAuthenticated, isAdmin, canView, canEdit, canViewPage, canEditPage, canViewRoute, canEditRoute, refreshAccess, login, renew, logout }
})
