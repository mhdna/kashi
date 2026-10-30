<template>
  <div class="rpt-card">
    <h3 v-if="title" class="rpt-h3">{{ title }}<RptHide :part="part" /></h3>
    <p v-if="note" class="rpt-note">{{ note }}</p>
    <slot />
  </div>
</template>

<script setup>
  import RptHide from './RptHide.vue'

  defineProps({
    // The part key(s) this card draws, for its Hide button.
    part: { type: [String, Array], default: '' },
    title: { type: String, default: '' },
    note: { type: String, default: '' },
  })
</script>
