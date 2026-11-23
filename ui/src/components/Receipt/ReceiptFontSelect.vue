<!--
  A receipt font picker: every font the till prints, each name shown in its
  own letters. Clearing it is the look's own font.
-->
<template>
  <v-select
    class="rfont"
    clearable
    :data-test="test"
    density="compact"
    hide-details
    :items="items"
    :label="label"
    :model-value="modelValue || null"
    variant="outlined"
    @update:menu="open => open && loadAll()"
    @update:model-value="v => emit('update:modelValue', v || '')"
  >
    <template #selection="{ item }">
      <span :style="fontStyle(item.value)">{{ item.title }}</span>
    </template>
    <template #item="{ props: itemProps, item }">
      <v-list-item v-bind="itemProps" :subtitle="(item.raw ?? item).kind" :title="undefined">
        <span class="rfont-name" :style="fontStyle(item.value)">{{ item.title }}</span>
      </v-list-item>
    </template>
  </v-select>
</template>

<script setup lang="ts">
  import { computed, inject, watch } from 'vue'
  import { fontItems, fontStyle, loadReceiptFont } from '@/utils/receiptFonts'
  import { t } from '@/i18n'

  const props = withDefaults(defineProps<{
    catalog: Record<string, any>
    modelValue?: string
    label?: string
    test?: string
  }>(), { modelValue: '', label: () => t('Font'), test: 'font' })
  const emit = defineEmits<{ 'update:modelValue': [key: string] }>()

  // The page gives the way to fetch a font file (with the user's sign-in).
  const fetchFile = inject<(key: string, bold: boolean) => Promise<ArrayBuffer>>('receiptFontFile', async () => new ArrayBuffer(0))

  const items = computed(() => fontItems(props.catalog))
  watch(() => props.modelValue, key => key && loadReceiptFont(key, false, fetchFile), { immediate: true })
  function loadAll () {
    for (const f of items.value) {loadReceiptFont(f.value, false, fetchFile)}
  }
</script>

<style scoped>
.rfont {
  flex: 1 1 150px;
  min-width: 140px;
}
.rfont-name {
  font-size: 1.05rem;
}
</style>
