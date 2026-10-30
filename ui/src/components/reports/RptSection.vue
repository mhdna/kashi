<template>
  <section class="rpt-section" :class="{ 'rpt-section--break': breakBefore }">
    <h2 v-if="title" class="rpt-h2">{{ title }}<RptHide :part="part" /></h2>
    <p v-if="note" class="rpt-note">{{ note }}</p>
    <slot />
  </section>
</template>

<script setup>
  import RptHide from './RptHide.vue'

  defineProps({
    // The part key(s) this section draws, for its Hide button.
    part: { type: [String, Array], default: '' },
    title: { type: String, default: '' },
    note: { type: String, default: '' },
    // Start this section on a new printed page.
    breakBefore: { type: Boolean, default: false },
  })
</script>
