<template>
  <button
    v-if="hide && keys.length > 0"
    class="rpt-screen-only rpt-hide-btn rpt-hide-btn--part"
    type="button"
    @click="hide(keys)"
  >
    {{ t('Hide') }}
  </button>
</template>

<script setup>
  import { computed, inject } from 'vue'
  import { t } from './reportI18n'

  // "Hide" beside a heading on the sheet: unticks its part(s) on the Reports
  // page (provided as reportHideParts). On screen only; the print copy
  // drops it.
  const props = defineProps({ part: { type: [String, Array], default: '' } })
  const hide = inject('reportHideParts', null)
  const keys = computed(() => [props.part].flat().filter(Boolean))
</script>
