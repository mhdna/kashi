import { readonly, ref } from 'vue'

// A frontend-only deploy (`make deploy-ui`) swaps the files on the server
// without the API ever going down, so nothing tells an open tab -- it keeps
// running the old build. Every build writes version.json with its own build
// id (vite.config.mjs); this checks it every minute and whenever the tab
// comes back into view, and flags a new version once the ids differ.
// ErrorBoundary then shows the "Updating" screen and reloads.

const RUNNING_BUILD = import.meta.env.VITE_APP_BUILD
const CHECK_MS = 60_000

const newVersion = ref(false)
let started = false

export async function checkForNewVersion () {
  if (newVersion.value || !started) {
    return
  }
  try {
    const res = await fetch(`${import.meta.env.BASE_URL}version.json?t=${Date.now()}`, { cache: 'no-store' })
    if (!res.ok) {
      return
    }
    const { build } = await res.json()
    if (build && build !== RUNNING_BUILD) {
      newVersion.value = true
    }
  } catch {
    // Offline or mid-deploy: try again on the next check.
  }
}

// Production builds only: the dev server has no version.json.
export function startVersionWatch () {
  if (started || !import.meta.env.PROD || !RUNNING_BUILD) {
    return
  }
  started = true
  setInterval(checkForNewVersion, CHECK_MS)
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'visible') {
      checkForNewVersion()
    }
  })
  window.addEventListener('focus', checkForNewVersion)
}

export function useAppVersion () {
  return { newVersion: readonly(newVersion) }
}
