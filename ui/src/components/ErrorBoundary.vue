<template>
  <ServerDownState v-if="serverDown" :offline="!online" />
  <div v-else-if="!ready" class="eb-boot" />
  <slot v-else />
</template>

<script setup lang="ts">
  import { onErrorCaptured, ref, watch } from 'vue'
  import { useRoute } from 'vue-router'
  import ServerDownState from '@/components/ServerDownState.vue'
  import { authFetch, isConnectionError } from '@/composables/useApi'
  import { useNetworkStatus } from '@/composables/useNetworkStatus'
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

  onErrorCaptured(error => {
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
