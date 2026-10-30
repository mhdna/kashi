<template>
  <section class="rpt-section" :class="{ 'rpt-section--break': breakBefore }">
    <h2 v-if="title" class="rpt-h2">{{ title }}</h2>
    <p v-if="note" class="rpt-note">{{ note }}</p>
    <slot />
  </section>
</template>

<script setup>
  defineProps({
    title: { type: String, default: '' },
    note: { type: String, default: '' },
    // Start this section on a new printed page.
    breakBefore: { type: Boolean, default: false },
  })
</script>
