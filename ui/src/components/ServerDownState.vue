<template>
  <div class="server-down" :class="{ 'server-down--overlay': overlay }">
    <section class="server-down__content text-center">
      <v-progress-circular v-if="updating && !offline" class="server-down__icon" indeterminate size="28" width="3" />
      <v-icon v-else class="server-down__icon" :icon="icon" size="40" />

      <h1 class="text-h6 font-weight-medium mt-5">{{ title }}</h1>
      <p class="text-body-2 text-medium-emphasis mt-1 mb-0">{{ text }}</p>

      <v-btn
        v-if="!overlay && !updating"
        class="mt-6"
        :loading="retrying"
        prepend-icon="mdi-refresh"
        variant="tonal"
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

  // A deploy is over in seconds; past a minute, say so rather than keep
  // promising "a few seconds".
  const slow = ref(false)
  const slowTimer = setTimeout(() => {
    slow.value = true
  }, 60_000)
  onUnmounted(() => clearTimeout(slowTimer))

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
    if (slow.value) return 'Taking longer than usual. Still checking…'
    return props.updating ? 'Back in a few seconds.' : props.message
  })

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
  background: rgb(var(--v-theme-background));
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
  max-width: 340px;
  width: 100%;
}

.server-down__icon {
  color: rgba(var(--v-theme-on-surface), 0.4);
}
</style>
