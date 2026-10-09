<template>
  <div class="year-stepper">
    <v-btn
      aria-label="Previous year"
      color="primary"
      icon="mdi-chevron-left"
      size="small"
      variant="tonal"
      @click="year--"
    />
    <span>{{ year }}</span>
    <v-btn
      aria-label="Next year"
      color="primary"
      :disabled="year >= currentYear"
      icon="mdi-chevron-right"
      size="small"
      variant="tonal"
      @click="year++"
    />
  </div>
</template>

<script setup lang="ts">
  // ‹ 2026 › -- steps a year back or forward, never past this year.
  const year = defineModel<number>({ required: true })
  const currentYear = new Date().getFullYear()
</script>

<style scoped>
.year-stepper {
  display: inline-flex;
  align-items: center;
  gap: 2px;
}

.year-stepper span {
  min-width: 46px;
  text-align: center;
  font-size: 0.95rem;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}
</style>
