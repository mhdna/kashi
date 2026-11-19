<!--
  The toolbar that opens right under a line of the receipt: its capitals,
  size, font, weight and alignment, whether it prints, and Default. Only what the
  till lets that line change shows. Nothing picked is the look's own.
-->
<template>
  <div class="rtools" data-no-discard-guard data-test="line-tools">
    <div class="rtools-head">
      <span class="rtools-title" data-test="line-tools-title">{{ info.label }}</span>
      <v-btn
        v-if="isElementChanged(design, info.key)"
        data-test="line-default"
        density="compact"
        prepend-icon="mdi-restore"
        size="small"
        text="Default"
        variant="text"
        @click="update(resetElement(design, info.key))"
      />
      <v-btn
        aria-label="Close"
        data-test="line-close"
        density="compact"
        icon="mdi-close"
        size="small"
        variant="text"
        @click="emit('close')"
      />
    </div>
    <div class="rtools-row">
      <v-btn-toggle
        v-if="can(catalog, info, 'case')"
        color="primary"
        data-test="line-case"
        density="compact"
        divided
        :model-value="el.case || undefined"
        variant="outlined"
        @update:model-value="v => set({ case: v || '' })"
      >
        <v-btn size="small" title="ALL CAPITALS" value="upper">ABC</v-btn>
        <v-btn size="small" title="First Letters" value="title">Abc</v-btn>
        <v-btn size="small" title="all small" value="lower">abc</v-btn>
      </v-btn-toggle>
      <SizeStepper
        v-if="can(catalog, info, 'size')"
        data-key="line"
        :max="catalog.size_max"
        :min="catalog.size_min"
        :value="el.size ?? 0"
        @step="n => update(sizeStep(design, catalog, info.key, n))"
      />
      <ReceiptFontSelect
        v-if="can(catalog, info, 'font')"
        :catalog="catalog"
        :model-value="el.font"
        test="line-font"
        @update:model-value="v => set({ font: v })"
      />
      <v-btn-toggle
        v-if="can(catalog, info, 'bold')"
        color="primary"
        data-test="line-bold"
        density="compact"
        divided
        :model-value="boldState(design, info.key)"
        variant="outlined"
        @update:model-value="v => update(setBold(design, info.key, v))"
      >
        <v-btn size="small" title="Bold" value="bold"><strong>B</strong></v-btn>
        <v-btn size="small" title="Regular" value="regular">Regular</v-btn>
      </v-btn-toggle>
      <v-btn-toggle
        v-if="can(catalog, info, 'align')"
        color="primary"
        data-test="line-align"
        density="compact"
        divided
        :model-value="el.align || undefined"
        variant="outlined"
        @update:model-value="v => set({ align: v || '' })"
      >
        <v-btn aria-label="Left" icon="mdi-format-align-left" size="small" value="left" />
        <v-btn aria-label="Centre" icon="mdi-format-align-center" size="small" value="center" />
        <v-btn aria-label="Right" icon="mdi-format-align-right" size="small" value="right" />
      </v-btn-toggle>
      <v-btn-toggle
        v-if="can(catalog, info, 'arabic')"
        color="primary"
        data-test="line-lang"
        density="compact"
        divided
        mandatory
        :model-value="el.lang || ''"
        variant="outlined"
        @update:model-value="v => set({ lang: v || '' })"
      >
        <v-btn v-for="o in langOptions" :key="o.value" size="small" :value="o.value">{{ o.title }}</v-btn>
      </v-btn-toggle>
      <v-btn
        v-if="can(catalog, info, 'hide')"
        :aria-label="el.hide ? 'Print it' : 'Leave it out'"
        data-test="line-hide"
        density="comfortable"
        :prepend-icon="el.hide ? 'mdi-eye' : 'mdi-eye-off'"
        size="small"
        :text="el.hide ? 'Print it' : 'Leave out'"
        variant="text"
        @click="set({ hide: !el.hide })"
      />
    </div>
    <div v-if="hint" class="rtools-hint">{{ hint }}</div>
    <div v-if="el.lang && !el.text_ar && !info.ar_default" class="rtools-hint" data-test="line-lang-hint">
      Click the Arabic line on the receipt to write its words; until then it prints in English.
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import ReceiptFontSelect from '@/components/Receipt/ReceiptFontSelect.vue'
  import SizeStepper from '@/components/Receipt/SizeStepper.vue'
  import { boldState, can, isElementChanged, langOptions, resetElement, setBold, setElement, sizeStep } from '@/utils/receiptDesign'

  const props = defineProps<{
    catalog: Record<string, any>
    design: Record<string, any>
    // The line's catalog entry.
    info: { key: string, label: string, default?: string, ar_default?: string, can: number }
  }>()
  const emit = defineEmits<{ 'update:design': [design: Record<string, any>], close: [] }>()

  const el = computed(() => props.design.elements?.[props.info.key] ?? {})
  const hint = computed(() => {
    if ((props.info.default ?? '').includes('{percent}')) {return '{percent} prints as the reward.'}
    if ((props.info.default ?? '').includes('{currency}')) {return '{currency} prints as the currency.'}
    return ''
  })

  function update (design: Record<string, any>) {
    emit('update:design', design)
  }
  function set (patch: Record<string, any>) {
    update(setElement(props.design, props.info.key, patch))
  }
</script>

<style scoped>
.rtools {
  padding: 8px 10px;
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  border-radius: 8px;
  background: rgb(var(--v-theme-surface));
  color: rgb(var(--v-theme-on-surface));
  box-shadow: 0 6px 18px rgba(0, 0, 0, 0.3);
  font-family: Roboto, sans-serif;
}
.rtools-head {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-bottom: 6px;
}
.rtools-title {
  flex: 1 1 auto;
  font-size: 0.85rem;
  font-weight: 600;
}
.rtools-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}
.rtools-hint {
  margin-top: 6px;
  font-size: 0.8rem;
  color: rgba(var(--v-theme-on-surface), 0.6);
}
</style>
