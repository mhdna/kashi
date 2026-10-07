import router from '@/router'
import { useAuthStore } from '@/stores/auth'

// Thrown by authFetch when the server can't be reached at all (powered off,
// wrong host). ErrorBoundary matches on these exact messages (via
// isConnectionError) to swap a blank screen for a friendly state, so keep
// them in sync.
export const SERVER_DOWN_MESSAGE = 'Server is down, we\'ll be back soon.'
// Same, but the device itself has no network -- telling the user the server
// is down would send them chasing the wrong problem.
export const NO_INTERNET_MESSAGE = 'No internet connection. Check your network and try again.'

// The message for a fetch() that rejected outright (no HTTP response).
export function connectionErrorMessage () {
  return typeof navigator !== 'undefined' && navigator.onLine === false
    ? NO_INTERNET_MESSAGE
    : SERVER_DOWN_MESSAGE
}

export function isConnectionError (error) {
  return error?.message === SERVER_DOWN_MESSAGE || error?.message === NO_INTERNET_MESSAGE
}

let refreshPromise = null

function renewOnce (authStore) {
  if (!refreshPromise) {
    refreshPromise = authStore.renew().finally(() => {
      refreshPromise = null
    })
  }
  return refreshPromise
}

export async function authFetch (url, options = {}) {
  const authStore = useAuthStore()

  const request = token => {
    const headers = { ...options.headers }
    if (token) {
      headers.Authorization = `Bearer ${token}`
    }
    return fetch(url, { ...options, headers })
  }

  let res
  try {
    res = await request(authStore.token)
  } catch {
    // fetch() rejects (rather than resolving with a bad status) when the
    // server can't be reached at all -- e.g. it's powered off. Surface a
    // clear, distinct message instead of the browser's raw "Failed to fetch".
    throw new Error(connectionErrorMessage())
  }

  if (res.status === 401 && authStore.refreshToken) {
    try {
      const newToken = await renewOnce(authStore)
      res = await request(newToken)
    } catch {
      // refresh failed, fall through to the logout/redirect below
    }
  }

  if (res.status === 401) {
    authStore.logout()
    if (router.currentRoute.value.path !== '/login') {
      router.push('/login')
    }
  }

  return res
}
