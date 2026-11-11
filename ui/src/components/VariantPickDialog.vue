<template>
  <v-card data-test="variant-pick-dialog">
    <v-card-title class="d-flex align-center ga-2">
      <v-icon icon="mdi-palette" size="20" />
      <span class="text-h6">Colour/size on scan</span>
      <v-spacer />
      <v-btn icon="mdi-close" size="small" variant="text" @click="$emit('close')" />
    </v-card-title>
    <v-divider />
    <v-card-text class="py-4">
      <p class="text-body-2 text-medium-emphasis mb-4">
        For items whose barcodes aren't printed per colour/size: make the till ask which
        colour/size when one of them is scanned. Paste product codes or barcodes. The branch
        setting (Branch settings, Scanning) is the default for every item.
      </p>
      <v-textarea
        v-model="text"
        auto-grow
        data-test="variant-pick-entries"
        density="compact"
        hide-details="auto"
        label="Product codes or barcodes"
        rows="4"
        variant="outlined"
      />
      <v-radio-group v-model="mode" class="mt-3" density="compact" hide-details>
        <v-radio
          v-for="m in VARIANT_PICK_MODES"
          :key="m.value"
          :label="m.title"
          :value="m.value"
        />
      </v-radio-group>
      <v-alert
        v-if="unmatched.length > 0"
        class="mt-3"
        data-test="variant-pick-unmatched"
        density="compact"
        type="warning"
        variant="tonal"
      >
        Not found: {{ unmatched.join(', ') }}
      </v-alert>
    </v-card-text>
    <v-divider />
    <v-card-actions>
      <v-spacer />
      <v-btn text="Cancel" variant="text" @click="$emit('close')" />
      <v-btn
        color="primary"
        data-test="variant-pick-save"
        :disabled="entries.length === 0"
        :loading="saving"
        text="Save"
        variant="flat"
        @click="save"
      />
    </v-card-actions>
  </v-card>
</template>

<script setup>
  import { computed, ref } from 'vue'
  import { authFetch } from '@/composables/useApi'
  import { useNotify } from '@/composables/useNotify'
  import { API_BASE } from '@/config'
  import { parseEntries, savedMessage, VARIANT_PICK_MODES } from '@/utils/variantPick'

  const emit = defineEmits(['close', 'saved'])
  const notify = useNotify()

  const text = ref('')
  const mode = ref('always')
  const saving = ref(false)
  const unmatched = ref([])
  const entries = computed(() => parseEntries(text.value))

  async function save () {
    saving.value = true
    try {
      const res = await authFetch(`${API_BASE}/variant_pick`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ entries: entries.value, mode: mode.value }),
      })
      const data = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(data.error || `Request failed with status ${res.status}`)
      unmatched.value = data.unmatched ?? []
      notify.success(savedMessage(mode.value, data.set ?? 0))
      emit('saved', data)
      if (unmatched.value.length === 0) emit('close')
    } catch (error) {
      notify.error(error.message)
    } finally {
      saving.value = false
    }
  }
</script>
