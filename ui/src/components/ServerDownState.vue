<template>
  <div class="server-down" :class="{ 'server-down--overlay': overlay }">
    <section class="server-down__card text-center">
      <PreparingState
        v-if="updating && !offline"
        kind="update"
        min-height="0"
        :subtitle="text"
        :title="title"
      />
      <template v-else>
        <div class="server-down__badge">
          <v-icon :icon="icon" size="34" />
        </div>
        <h1 class="text-h6 font-weight-medium mt-4">{{ title }}</h1>
        <p class="text-body-2 text-medium-emphasis mt-1 mb-0">{{ text }}</p>
      </template>

      <!-- Something is happening: a thin bar sweeps under the text. -->
      <div v-if="updating && !offline" class="server-down__sweep" />

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

/* Over a still-mounted page: the page shows through, dimmed and blurred,
   so it's clear nothing on it was lost. */
.server-down--overlay {
  position: fixed;
  inset: 0;
  /* Above the app bar, drawers and dialogs. */
  z-index: 3000;
  background: rgba(var(--v-theme-background), 0.7);
  backdrop-filter: blur(6px);
  animation: server-down-fade 0.2s ease-out;
}

.server-down__card {
  position: relative;
  overflow: hidden;
  max-width: 360px;
  width: 100%;
  padding: 28px 28px 32px;
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  border-radius: 16px;
  background: rgb(var(--v-theme-surface));
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.18);
  animation: server-down-rise 0.25s ease-out;
}

.server-down__card :deep(.preparing) {
  padding: 0;
}

.server-down__badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: rgba(var(--v-theme-on-surface), 0.06);
  color: rgba(var(--v-theme-on-surface), 0.55);
}

.server-down__sweep {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 3px;
  overflow: hidden;
  background: rgba(var(--v-theme-primary), 0.12);
}
.server-down__sweep::after {
  content: '';
  position: absolute;
  inset: 0;
  width: 40%;
  background: rgb(var(--v-theme-primary));
  border-radius: 2px;
  animation: server-down-sweep 1.4s ease-in-out infinite;
}

@keyframes server-down-fade {
  from { opacity: 0; }
}
@keyframes server-down-rise {
  from { opacity: 0; transform: translateY(8px) scale(0.98); }
}
@keyframes server-down-sweep {
  from { transform: translateX(-100%); }
  to { transform: translateX(250%); }
}

@media (prefers-reduced-motion: reduce) {
  .server-down--overlay,
  .server-down__card,
  .server-down__sweep::after {
    animation: none;
  }
}
</style>
