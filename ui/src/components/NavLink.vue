<!--
  One page in the navigation drawer, with what it reports beside it: a status
  dot (stock health, commands, conflicts, WhatsApp), the Alerts counts, or the
  Security page's open alerts.
-->
<template>
  <v-list-item
    :data-test="`nav-${item.to}`"
    :prepend-icon="item.icon"
    :title="item.title"
    :to="item.to"
    :value="item.to"
  >
    <template v-if="status" #append>
      <span
        :aria-label="`${item.title} status is ${status}`"
        class="nav-status d-flex align-center ga-2"
        role="img"
      >
        <span class="nav-status-dot" :class="`nav-status-dot--${status}`" />
      </span>
    </template>
    <template v-else-if="item.append === 'alert-count' && !alertCounts.loaded" #append>
      <span aria-label="Alerts are loading" class="nav-status d-flex align-center ga-2" role="img">
        <span class="nav-status-dot nav-status-dot--loading" />
      </span>
    </template>
    <template v-else-if="item.append === 'alert-count' && (alertCounts.out > 0 || alertCounts.low > 0)" #append>
      <span class="d-flex align-center ga-1">
        <v-badge v-if="alertCounts.out > 0" color="error" :content="alertCounts.out" inline />
        <v-badge v-if="alertCounts.low > 0" color="warning" :content="alertCounts.low" inline />
      </span>
    </template>
    <template v-else-if="item.to === '/security' && securityCount > 0" #append>
      <v-badge color="error" :content="securityCount" inline />
    </template>
  </v-list-item>
</template>

<script setup lang="ts">
  defineProps({
    item: { type: Object, required: true },
    status: { type: String, default: null },
    alertCounts: { type: Object, default: () => ({ loaded: true, out: 0, low: 0 }) },
    securityCount: { type: Number, default: 0 },
  })
</script>

<style scoped>
.nav-status-dot {
  width: 18px;
  height: 18px;
  display: block;
  border-radius: 50%;
  background-color: rgb(var(--v-theme-success));
}
.nav-status-dot--healthy {
  background-color: rgb(var(--v-theme-success));
}
.nav-status-dot--warning {
  background-color: rgb(var(--v-theme-warning));
}
.nav-status-dot--critical {
  background-color: rgb(var(--v-theme-error));
}
/* Placeholder while the first status fetch is in flight. */
.nav-status-dot--loading {
  background-color: rgba(var(--v-theme-on-surface), 0.12);
  animation: nav-status-pulse 1.2s ease-in-out infinite;
}
@keyframes nav-status-pulse {
  50% { opacity: 0.4; }
}
.nav-status {
  min-width: 24px;
  justify-content: center;
  margin-inline-end: 8px;
}
</style>
