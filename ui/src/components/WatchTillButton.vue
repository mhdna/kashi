<template>
  <span v-if="visible">
    <v-btn
      v-if="icon"
      aria-label="Watch till"
      data-test="watch-register"
      density="comfortable"
      icon="mdi-eye"
      size="small"
      :to="{ path: `/live/${registerId}`, query: { name } }"
      variant="text"
      @click.stop
    />
    <v-btn
      v-else
      class="ms-2"
      data-test="watch-register"
      prepend-icon="mdi-eye"
      size="small"
      text="Watch"
      :to="{ path: `/live/${registerId}`, query: { name } }"
      variant="outlined"
    />
    <span
      v-if="!icon && watchers.length > 0"
      class="text-caption text-medium-emphasis ms-2"
      data-test="watchers"
    >
      Watched by {{ watchers.join(', ') }}
    </span>
  </span>
</template>

<script setup lang="ts">
  import { computed, onMounted } from 'vue'
  import { useLiveView } from '@/composables/useLiveView'
  import { useAuthStore } from '@/stores/auth'

  // The "Watch" button of a till that is online, shown only while Settings has
  // live view on and the role can use it (System).
  const props = defineProps({
    registerId: { type: Number, required: true },
    name: { type: String, default: '' },
    online: { type: Boolean, default: false },
    // Just the eye, for tight lists such as the connected branches popup.
    icon: { type: Boolean, default: false },
  })

  const auth = useAuthStore()
  const { enabled, load, listen, watchersOf } = useLiveView()
  onMounted(() => {
    listen()
    load().catch(() => {})
  })

  const visible = computed(() => enabled.value && props.online && auth.canViewPage('system', 'settings'))
  const watchers = computed(() => watchersOf(props.registerId))
</script>
