<template>
  <div class="server-down" :class="{ 'server-down--overlay': overlay }">
    <section class="server-down__content text-center">
      <v-icon class="server-down__icon" :icon="icon" size="72" />

      <h1 class="text-h4 font-weight-bold mt-6">{{ title }}</h1>
      <p class="text-body-1 text-medium-emphasis mt-3 mb-0">{{ text }}</p>
      <p v-if="!offline" class="d-flex align-center justify-center ga-2 text-body-2 text-medium-emphasis mt-4 mb-0">
        <v-progress-circular indeterminate size="14" width="2" />
        {{ slow ? 'Taking longer than usual. Still checking…' : 'This page will come back on its own.' }}
      </p>

      <v-btn
        v-if="!overlay"
        class="mt-8"
        color="primary"
        :loading="retrying"
        prepend-icon="mdi-refresh"
        rounded="pill"
        size="large"
        @click="retry"
      >
        Try again
      </v-btn>
    </section>
  </div>
</template>

<script setup lang="ts">
  import { computed, onUnmounted, ref } from 'vue'
  import { NO_INTERNET_MESSAGE, SERVER_DOWN_MESSAGE } from '@/composables/useApi'

  const props = defineProps({
    message: {
      type: String,
      default: SERVER_DOWN_MESSAGE,
    },
    // The device has no network at all, as opposed to the server being down.
    offline: {
      type: Boolean,
      default: false,
    },
    // Laid over a still-mounted page (ErrorBoundary) instead of replacing
    // it. No "Try again": it recovers on its own, and a reload would lose
    // whatever is on the page underneath.
    overlay: {
      type: Boolean,
      default: false,
    },
    // The API is being redeployed (see useServerStatus), not down.
    updating: {
      type: Boolean,
      default: false,
    },
  })

  const icon = computed(() => {
    if (props.offline) return 'mdi-wifi-off'
    return props.updating ? 'mdi-update' : 'mdi-server-network-off'
  })
  const title = computed(() => {
    if (props.offline) return 'No internet'
    return props.updating ? 'Updating Kashi Cloud' : 'Can\'t reach the server'
  })
  const text = computed(() => {
    if (props.offline) return NO_INTERNET_MESSAGE
    return props.updating ? 'A new version is being installed. This usually takes a few seconds.' : props.message
  })

  // A deploy is over in seconds; past a minute, say so rather than keep
  // promising "a few seconds".
  const slow = ref(false)
  const slowTimer = setTimeout(() => {
    slow.value = true
  }, 60_000)
  onUnmounted(() => clearTimeout(slowTimer))

  // A full reload is the simplest reliable recovery: it re-runs auth, re-mounts
  // every page and re-issues the requests that failed while the server was off.
  const retrying = ref(false)
  function retry () {
    retrying.value = true
    window.location.reload()
  }
</script>

<style scoped>
.server-down {
  align-items: center;
  background:
    radial-gradient(circle at top, rgba(var(--v-theme-primary), 0.14), transparent 45%),
    rgb(var(--v-theme-background));
  display: flex;
  justify-content: center;
  min-height: 100dvh;
  padding: 24px;
}

.server-down--overlay {
  position: fixed;
  inset: 0;
  /* Above the app bar, drawers and dialogs. */
  z-index: 3000;
}

.server-down__content {
  max-width: 380px;
  width: 100%;
}

.server-down__icon {
  color: rgba(var(--v-theme-on-surface), 0.35);
}
</style>
