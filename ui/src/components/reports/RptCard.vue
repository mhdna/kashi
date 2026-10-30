<template>
  <div class="rpt-card">
    <h3 v-if="title" class="rpt-h3">{{ title }}</h3>
    <p v-if="note" class="rpt-note">{{ note }}</p>
    <slot />
  </div>
</template>

<script setup>
  defineProps({
    title: { type: String, default: '' },
    note: { type: String, default: '' },
  })
</script>
