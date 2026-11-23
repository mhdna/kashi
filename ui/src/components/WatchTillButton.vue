<template>
  <span v-if="visible">
    <v-btn
      v-if="icon"
      :aria-label="t('Watch till')"
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
      :text="t('Watch')"
      :to="{ path: `/live/${registerId}`, query: { name } }"
      variant="outlined"
    />
    <v-btn
      v-if="!icon && separateAllowed"
      class="ms-2"
      data-test="separate-register"
      prepend-icon="mdi-monitor-multiple"
      size="small"
      text="Open separate screen"
      :to="{ path: `/live/${registerId}`, query: { name, mode: 'virtual' } }"
      variant="outlined"
    />
    <span
      v-if="!icon && separateUser"
      class="text-caption text-medium-emphasis ms-2"
      data-test="separate-user"
    >
      Separate screen used by {{ separateUser }}
    </span>
    <span
      v-if="!icon && watchers.length > 0"
      class="text-caption text-medium-emphasis ms-2"
      data-test="watchers"
    >
      {{ t('Watched by {names}', { names: watchers.join(', ') }) }}
    </span>
  </span>
</template>

<script setup lang="ts">
  import { computed, onMounted } from 'vue'
  import { t } from '@/i18n'
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
  const { enabled, controlEnabled, load, listen, watchersOf, separateUserOf } = useLiveView()
  onMounted(() => {
    listen()
    load().catch(() => {})
  })

  const visible = computed(() => enabled.value && props.online && auth.canViewPage('system', 'settings'))
  const watchers = computed(() => watchersOf(props.registerId))
  const separateUser = computed(() => separateUserOf(props.registerId))
  // The separate screen reads and writes: it needs the control switch and the right to edit System.
  const separateAllowed = computed(() => visible.value && controlEnabled.value && auth.canEditPage('system', 'settings'))
</script>
