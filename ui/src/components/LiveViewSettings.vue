<template>
  <div :key="version" data-test="live-view-settings">
    <h3 class="text-subtitle-1 mt-8 mb-1">Live view of tills</h3>
    <p class="text-body-2 text-medium-emphasis mb-2">
      Watch a connected till's screen live from the Branches page, without AnyDesk. Nothing is recorded or
      stored, and while it is off the tills and the server do no extra work at all. At most 3 tills can be
      watched at once. Passwords are never shown.
    </p>
    <v-switch
      color="primary"
      data-test="live-view-enabled"
      density="compact"
      :disabled="!canEdit || saving"
      hide-details
      inset
      label="Allow watching tills live"
      :model-value="enabled"
      @update:model-value="value => change({ enabled: value, controlEnabled: controlEnabled })"
    />
    <v-switch
      color="primary"
      data-test="live-view-control"
      density="compact"
      :disabled="!canEdit || saving || !enabled"
      hide-details
      inset
      label="Allow remote control (click and type on the till)"
      :model-value="controlEnabled"
      @update:model-value="value => change({ enabled, controlEnabled: value })"
    />
    <p v-if="error" class="text-body-2 text-error mt-2" data-test="live-view-error">{{ error }}</p>
  </div>
</template>

<script setup lang="ts">
  import { onMounted, ref } from 'vue'
  import { useLiveView } from '@/composables/useLiveView'

  defineProps({ canEdit: { type: Boolean, default: false } })

  const { enabled, controlEnabled, load, save } = useLiveView()
  const saving = ref(false)
  const error = ref('')
  // Bumped after a failed save so the switches are drawn again from the saved
  // state (a rejected click would otherwise leave the switch showing on).
  const version = ref(0)

  onMounted(async () => {
    try {
      await load(true)
    } catch (error_) {
      error.value = error_.message
    }
  })

  async function change (next) {
    saving.value = true
    error.value = ''
    try {
      await save(next)
    } catch (error_) {
      error.value = error_.message
      version.value++
    } finally {
      saving.value = false
    }
  }
</script>
