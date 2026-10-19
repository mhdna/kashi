<!--
  Picks the look of a branch's printed invoice: a theme and one of its
  layouts, with a preview of each. The catalog and the preview images are
  generated from the till's own renderer (`make receipt-previews` in
  kashi-pos), so what is shown here is what the till prints; kashi cannot run
  that renderer itself. The previews use sample items under the Classic theme's
  default fonts; a till's own store name, fonts and paper width are shown by
  the preview on the till.
-->
<template>
  <div class="rlp">
    <div class="rlp-choices">
      <div class="rlp-label">Theme</div>
      <v-list class="rlp-themes" density="compact" nav>
        <v-list-item
          v-for="t in themes"
          :key="t.id"
          :active="t.id === currentTheme"
          :disabled="disabled"
          rounded="lg"
          :subtitle="t.best_for"
          :title="t.name"
          @click="pickTheme(t)"
        >
          <template v-if="t.id === savedTheme" #append>
            <v-chip color="success" size="x-small" variant="tonal">In use</v-chip>
          </template>
        </v-list-item>
      </v-list>

      <div class="rlp-label mt-4">Layout</div>
      <v-btn-toggle
        class="rlp-layouts"
        color="primary"
        density="comfortable"
        :disabled="disabled"
        divided
        mandatory
        :model-value="currentLayout"
        variant="outlined"
        @update:model-value="v => emit('update:layout', v)"
      >
        <v-btn v-for="l in themeInfo.layouts" :key="l.id" :value="l.id">{{ l.id }}</v-btn>
      </v-btn-toggle>
      <div class="mt-2">
        <div class="text-body-2 font-weight-medium">{{ layoutInfo.name }}</div>
        <div class="text-body-2 text-medium-emphasis">{{ layoutInfo.description }}</div>
      </div>
      <div class="text-caption text-medium-emphasis mt-4">{{ themeInfo.description }}</div>
    </div>

    <div class="rlp-preview">
      <img
        :alt="`${themeInfo.name} receipt, layout ${layoutInfo.id}`"
        :src="`/receipt-previews/${layoutInfo.preview}`"
      >
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import catalog from '@/data/receiptThemes.json'

  type Layout = { id: string, name: string, description: string, preview: string }
  type Theme = { id: string, name: string, description: string, best_for: string, layouts: Layout[] }

  const themes: Theme[] = catalog.themes

  const props = withDefaults(defineProps<{
    theme?: string
    layout?: string
    // The look the branch has now, to mark in the list while browsing.
    savedTheme?: string
    disabled?: boolean
  }>(), { theme: '', layout: '', savedTheme: '', disabled: false })
  const emit = defineEmits<{
    'update:theme': [id: string]
    'update:layout': [id: string]
  }>()

  // An id the catalog does not know (a theme added on a newer till, or none
  // set yet) shows as the default, which is also what the till falls back to.
  const themeInfo = computed<Theme>(() =>
    themes.find(t => t.id === props.theme)
    ?? themes.find(t => t.id === catalog.default_theme)
    ?? themes[0]!)
  const currentTheme = computed(() => themeInfo.value.id)
  const layoutInfo = computed<Layout>(() =>
    themeInfo.value.layouts.find(l => l.id === props.layout) ?? themeInfo.value.layouts[0]!)
  const currentLayout = computed(() => layoutInfo.value.id)

  // Switching theme keeps the same layout letter when the new theme has it,
  // so browsing with A selected stays on A.
  function pickTheme (t: Theme) {
    emit('update:theme', t.id)
    if (!t.layouts.some(l => l.id === currentLayout.value)) {
      emit('update:layout', t.layouts[0]!.id)
    }
  }
</script>

<style scoped>
.rlp {
  display: grid;
  grid-template-columns: 250px minmax(0, 1fr);
  gap: 20px;
  align-items: start;
}
.rlp-label {
  font-size: 0.75rem;
  color: rgba(var(--v-theme-on-surface), 0.6);
  margin-bottom: 4px;
}
.rlp-preview {
  display: flex;
  justify-content: center;
  padding: 12px;
  border-radius: 8px;
  background: rgba(var(--v-theme-on-surface), 0.06);
}
/* The receipt is white paper in either app theme. */
.rlp-preview img {
  width: min(100%, 340px);
  background: #fff;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.25);
}
@media (max-width: 760px) {
  .rlp {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
