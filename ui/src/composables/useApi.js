import { isGatewayStatus, markServerUpdating } from '@/composables/useServerStatus'
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
// The proxy answered but the API behind it didn't (502/503/504) -- in
// practice a deploy restarting it. See useServerStatus.
export const SERVER_UPDATING_MESSAGE = 'The server is being updated, it\'ll be back in a moment.'

// The message for a fetch() that rejected outright (no HTTP response).
export function connectionErrorMessage () {
  return typeof navigator !== 'undefined' && navigator.onLine === false
    ? NO_INTERNET_MESSAGE
    : SERVER_DOWN_MESSAGE
}

// The API's 403 for a request the user's role doesn't allow, worded by
// permissionMiddleware in api/access.go ("your role (Viewer) can't view
// Sales") -- keep the two in sync. The UI already hides what a role can't
// use; whatever still trips this (a shared widget loading data from another
// section, say) is dropped silently rather than shown to the user.
const PERMISSION_DENIED = /your role \(.*\) can't (view|change) /

export function isPermissionDenied (message) {
  return PERMISSION_DENIED.test(String(message ?? ''))
}

export function isConnectionError (error) {
  return error?.message === SERVER_DOWN_MESSAGE
    || error?.message === NO_INTERNET_MESSAGE
    || error?.message === SERVER_UPDATING_MESSAGE
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

  if (isGatewayStatus(res.status)) {
    markServerUpdating()
    throw new Error(SERVER_UPDATING_MESSAGE)
  }

  if (res.status === 401 && authStore.refreshToken) {
    try {
      const newToken = await renewOnce(authStore)
      res = await request(newToken)
    } catch (err) {
      // A transient refresh failure (server unreachable mid-deploy-restart,
      // or a gateway status while it's coming back up) says nothing about
      // whether the session itself is still good -- see the matching
      // comment in stores/auth.js's renew(). Surface it as a retryable
      // error instead of silently falling through to a logout the user
      // did nothing to deserve; markServerUpdating already has its own
      // recovery/retry flow (useServerStatus.js) for this case.
      if (err?.transientRefreshFailure) {
        markServerUpdating()
        throw err
      }
      // refresh failed for a real reason (bad/expired refresh token), fall
      // through to the logout/redirect below
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

// authUpload is authFetch for a file upload that reports progress: fetch()
// can't tell how much of a request body has been sent, XMLHttpRequest can.
// onProgress gets 0..1 as the body goes out. Token handling matches
// authFetch (one renew-and-retry on a 401); the result is a Response.
export async function authUpload (url, { method = 'POST', body, onProgress } = {}) {
  const authStore = useAuthStore()

  const send = token => new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open(method, url)
    if (token) {xhr.setRequestHeader('Authorization', `Bearer ${token}`)}
    xhr.upload.addEventListener('progress', e => {
      if (e.lengthComputable) {onProgress?.(e.loaded / e.total)}
    })
    xhr.addEventListener('load', () => {
      resolve(new Response(xhr.responseText, { status: xhr.status }))
    })
    xhr.addEventListener('error', () => reject(new Error(connectionErrorMessage())))
    xhr.send(body)
  })

  let res = await send(authStore.token)
  if (isGatewayStatus(res.status)) {
    markServerUpdating()
    throw new Error(SERVER_UPDATING_MESSAGE)
  }
  if (res.status === 401 && authStore.refreshToken) {
    try {
      onProgress?.(0)
      res = await send(await renewOnce(authStore))
    } catch {
      // refresh failed, fall through to the logout below
    }
  }
  if (res.status === 401) {
    authStore.logout()
    if (router.currentRoute.value.path !== '/login') {router.push('/login')}
  }
  return res
}
