import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { SERVER_DOWN_MESSAGE } from '@/composables/useApi'
import { API_BASE } from '@/config'

// Turn a login response status into a message the user can act on. The
// backend leaks raw Go errors, so nothing here relies on its body text.
function loginMessage (status, bodyError) {
  switch (status) {
    case 400: {
      return 'Enter a valid username and password.'
    }
    case 401:
    case 403: {
      return 'Incorrect username or password.'
    }
    case 423: {
      return 'This account is locked. Try again later or contact an administrator.'
    }
    case 429: {
      return bodyError || 'Too many attempts. Please wait a while and try again.'
    }
    default: {
      return 'Something went wrong on our end. Please try again in a moment.'
    }
  }
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const refreshToken = ref(localStorage.getItem('refreshToken') || '')
  const user = ref(JSON.parse(localStorage.getItem('user') || 'null'))

  const isAuthenticated = computed(() => !!token.value)

  async function login (name, password) {
    let res
    try {
      res = await fetch(`${API_BASE}/users/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, password }),
      })
    } catch {
      // fetch only rejects on network-level failures (server unreachable,
      // DNS, offline) -- never on HTTP error status. Same message authFetch
      // uses so the "server is down" state stays consistent.
      throw new Error(SERVER_DOWN_MESSAGE)
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
    localStorage.removeItem('token')
    localStorage.removeItem('refreshToken')
    localStorage.removeItem('user')
  }

  return { token, refreshToken, user, isAuthenticated, login, renew, logout }
})
