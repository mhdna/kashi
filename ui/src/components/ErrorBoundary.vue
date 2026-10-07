<template>
  <ServerDownState v-if="serverDown" :offline="!online" :updating="updating" />
  <div v-else-if="!ready" class="eb-boot" />
  <template v-else>
    <slot />
    <!-- Laid over the page rather than replacing it, so the page (and any
      half-filled form) is still there, untouched, once the API is back. -->
    <ServerDownState v-if="updating" overlay updating />
  </template>
</template>

<script setup lang="ts">
  import { onErrorCaptured, ref, watch } from 'vue'
  import { useRoute } from 'vue-router'
  import ServerDownState from '@/components/ServerDownState.vue'
  import { authFetch, isConnectionError, SERVER_UPDATING_MESSAGE } from '@/composables/useApi'
  import { useNetworkStatus } from '@/composables/useNetworkStatus'
  import { useServerStatus, waitForServer } from '@/composables/useServerStatus'
  import { API_BASE } from '@/config'
  import { useAuthStore } from '@/stores/auth'

  // A page's async setup that awaits authFetch throws all the way up here when
  // the server is unreachable. Without a boundary Vue tears the subtree down and
  // the user just sees a blank screen with console noise. Catch that one error
  // and render an explicit state instead; let everything else bubble to
  // app.config.errorHandler untouched.
  const serverDown = ref(false)

  // Before the first page (and its layout chrome) paints, probe the API once so
  // an outage lands straight on ServerDownState instead of flashing the app
  // shell while a page's setup fetch slowly rejects. Any HTTP response -- even
  // 401 -- means the server is up; only a network-level failure counts as down.
  const ready = ref(false)
  const authStore = useAuthStore()

  async function probe () {
    if (!authStore.isAuthenticated) {
      ready.value = true
      return
    }
    try {
      await authFetch(`${API_BASE}/app_settings`, { method: 'GET' })
    } catch (error) {
      if (isConnectionError(error)) {
        serverDown.value = true
      }
    } finally {
      ready.value = true
    }
  }
  probe()

  // Vue's error source: the text in dev, a #runtime-N docs link in prod
  // (0 = setup function).
  const isSetupError = (info: string) => info === 'setup function' || info.endsWith('#runtime-0')

  onErrorCaptured((error, _instance, info) => {
    // A 502 mid-deploy from a page that already rendered (a click handler, a
    // refresh timer): the updating overlay is up already, and replacing the
    // page would throw away what's on it. Only a page whose setup failed --
    // it has nothing on screen to keep -- is swapped out.
    if (error?.message === SERVER_UPDATING_MESSAGE && !isSetupError(info)) {
      return false
    }
    if (isConnectionError(error)) {
      serverDown.value = true
      return false
    }
    return undefined
  })

  // If the user navigates (e.g. via the still-mounted nav drawer) after a
  // transient outage, drop the error state and let the target page try again.
  const route = useRoute()
  watch(() => route.fullPath, () => {
    serverDown.value = false
  })

  // While the API is being redeployed (a 502 from the proxy, see
  // useServerStatus) the updating overlay covers the app. Once the API
  // answers again the overlay goes; a page that had failed to load is
  // re-mounted so it re-issues its requests.
  const { updating } = useServerStatus()
  watch(updating, isUpdating => {
    if (!isUpdating) serverDown.value = false
  })

  // A plain connection failure recovers the same way: keep checking in the
  // background and bring the page back once the server answers.
  watch(serverDown, isDown => {
    if (isDown) {
      waitForServer().then(() => {
        serverDown.value = false
      })
    }
  })

  // Coming back online is the cue to try again on its own: dropping the
  // error state re-mounts the page, which re-issues its requests.
  const { online } = useNetworkStatus()
  watch(online, isOnline => {
    if (isOnline) serverDown.value = false
  })
</script>

<style scoped>
.eb-boot {
  min-height: 100dvh;
  background: rgb(var(--v-theme-background));
}
</style>
