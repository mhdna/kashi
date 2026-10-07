<template>
  <!-- Fixed reel at top-right, styled to match kashi-pos snackbars. -->
  <div class="app-notif-reel">
    <TransitionGroup name="notif" tag="div" class="app-notif-list">
      <div
        v-for="item in items"
        :key="item.id"
        class="app-notif-card d-flex align-center ga-3"
      >
        <v-icon
          :color="iconColorFor(item.type)"
          :icon="iconFor(item.type)"
          size="18"
          class="flex-shrink-0"
        />
        <span class="app-notif-text text-body-2">{{ item.message }}</span>
        <v-btn
          class="flex-shrink-0"
          density="compact"
          icon="mdi-close"
          size="x-small"
          variant="text"
          @click="store.remove(item.id)"
        />
      </div>
    </TransitionGroup>
  </div>
</template>

<script setup>
  import { computed } from 'vue'
  import { useNotifyStore } from '@/stores/notify'

  const store = useNotifyStore()
  const items = computed(() => store.items)

  function iconFor (type) {
    switch (type) {
      case 'success': return 'mdi-check-circle'
      case 'error': return 'mdi-alert-circle'
      case 'warning': return 'mdi-alert'
      default: return 'mdi-information'
    }
  }

  function iconColorFor (type) {
    switch (type) {
      case 'success': return 'success'
      case 'error': return 'error'
      case 'warning': return 'warning'
      default: return 'info'
    }
  }
</script>

<style scoped>
.app-notif-reel {
  position: fixed;
  top: 64px; /* below the app bar */
  right: 16px;
  z-index: 3000;
  pointer-events: none;
}

.app-notif-list {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 8px;
}

.app-notif-card {
  width: 360px;
  max-width: calc(100vw - 32px);
  padding: 10px 10px 10px 14px;
  border-radius: 6px;
  pointer-events: auto;
  background: rgba(var(--v-theme-surface-variant), 0.72);
  color: rgb(var(--v-theme-on-surface-variant));
  border: 1px solid rgba(var(--v-theme-on-surface-variant), 0.1);
  box-shadow: 0 4px 18px rgba(0, 0, 0, 0.12);
  backdrop-filter: blur(8px);
}

.app-notif-text {
  flex: 1 1 auto;
  min-width: 0;
  line-height: 1.4;
  word-break: break-word;
}

.notif-enter-active,
.notif-leave-active {
  transition: transform 0.25s ease, opacity 0.25s ease;
}

.notif-enter-from,
.notif-leave-to {
  transform: translateX(14px);
  opacity: 0;
}

.notif-leave-active {
  position: absolute;
}

.notif-move {
  transition: transform 0.25s ease;
}
</style>
