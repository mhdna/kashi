<!--
  The toolbar that opens right under a part of the receipt: whether it
  prints, its place (the parts under the totals move), every line of it at
  once (capitals, size, font, weight, alignment), its lines and box, the date
  format for the customer, salesperson and date part, and Default.
-->
<template>
  <div class="rtools" data-no-discard-guard data-test="part-tools">
    <div class="rtools-head">
      <span class="rtools-title" data-test="part-tools-title">{{ info.label }}</span>
      <template v-if="movable">
        <v-btn
          aria-label="Move up"
          data-test="move-up"
          density="compact"
          :disabled="!canMove(-1)"
          icon="mdi-arrow-up"
          size="small"
          title="Move up (Alt+Up)"
          variant="text"
          @click="update(moveSection(design, catalog, info.section, -1))"
        />
        <v-btn
          aria-label="Move down"
          data-test="move-down"
          density="compact"
          :disabled="!canMove(1)"
          icon="mdi-arrow-down"
          size="small"
          title="Move down (Alt+Down)"
          variant="text"
          @click="update(moveSection(design, catalog, info.section, 1))"
        />
      </template>
      <v-btn
        v-if="isGroupChanged(design, catalog, group)"
        data-test="reset-part"
        density="compact"
        prepend-icon="mdi-restore"
        size="small"
        text="Default"
        variant="text"
        @click="update(resetGroup(design, catalog, group))"
      />
      <v-btn
        aria-label="Close"
        data-test="part-close"
        density="compact"
        icon="mdi-close"
        size="small"
        variant="text"
        @click="emit('close')"
      />
    </div>

    <div class="rtools-row">
      <v-btn
        v-if="info.section"
        data-test="part-printed"
        density="comfortable"
        :prepend-icon="hidden ? 'mdi-eye' : 'mdi-eye-off'"
        size="small"
        :text="hidden ? 'Print this part' : 'Leave out'"
        variant="text"
        @click="update(toggleSection(design, info.section))"
      />
      <v-select
        v-if="group === 'meta'"
        class="rtools-select"
        data-test="date-format"
        density="compact"
        hide-details
        item-title="label"
        item-value="key"
        :items="catalog.date_formats"
        label="Date"
        :model-value="design.date_format || ''"
        variant="outlined"
        @update:model-value="v => update({ ...design, date_format: v || '' })"
      />
    </div>

    <div class="rtools-label">Every line</div>
    <div class="rtools-row" data-test="whole-part">
      <v-btn-toggle
        color="primary"
        data-test="group-case"
        density="compact"
        divided
        :model-value="g.case || undefined"
        variant="outlined"
        @update:model-value="v => update(setGroup(design, group, { case: v || '' }))"
      >
        <v-btn size="small" title="ALL CAPITALS" value="upper">ABC</v-btn>
        <v-btn size="small" title="First Letters" value="title">Abc</v-btn>
        <v-btn size="small" title="all small" value="lower">abc</v-btn>
      </v-btn-toggle>
      <SizeStepper
        data-key="group"
        :max="catalog.size_max"
        :min="catalog.size_min"
        :value="g.size ?? 0"
        @step="n => update(groupSizeStep(design, catalog, group, n))"
      />
      <ReceiptFontSelect
        :catalog="catalog"
        :model-value="g.font"
        test="group-font"
        @update:model-value="v => update(setGroup(design, group, { font: v || '' }))"
      />
      <v-btn-toggle
        color="primary"
        data-test="group-bold"
        density="compact"
        divided
        :model-value="groupBoldState(design, group)"
        variant="outlined"
        @update:model-value="v => update(setGroupBold(design, group, v))"
      >
        <v-btn size="small" title="Bold" value="bold"><strong>B</strong></v-btn>
        <v-btn size="small" title="Regular" value="regular">Regular</v-btn>
      </v-btn-toggle>
      <v-btn-toggle
        color="primary"
        data-test="group-align"
        density="compact"
        divided
        :model-value="g.align || undefined"
        variant="outlined"
        @update:model-value="v => update(setGroup(design, group, { align: v || '' }))"
      >
        <v-btn aria-label="Left" icon="mdi-format-align-left" size="small" value="left" />
        <v-btn aria-label="Centre" icon="mdi-format-align-center" size="small" value="center" />
        <v-btn aria-label="Right" icon="mdi-format-align-right" size="small" value="right" />
      </v-btn-toggle>
    </div>

    <div class="rtools-label">Lines and box</div>
    <div class="rtools-row" data-test="part-lines">
      <v-select
        class="rtools-select"
        clearable
        data-test="decor-above"
        density="compact"
        hide-details
        :items="lineStyleOptions"
        label="Line above"
        :model-value="d.above || null"
        variant="outlined"
        @update:model-value="v => update(setDecor(design, group, { above: v || '' }))"
      />
      <v-select
        class="rtools-select"
        clearable
        data-test="decor-below"
        density="compact"
        hide-details
        :items="belowOptions"
        label="Line under"
        :model-value="d.below || null"
        variant="outlined"
        @update:model-value="v => update(setDecor(design, group, { below: v || '' }))"
      />
      <v-select
        class="rtools-select"
        clearable
        data-test="decor-box"
        density="compact"
        hide-details
        :items="lineStyleOptions"
        label="Box"
        :model-value="d.box || null"
        variant="outlined"
        @update:model-value="v => update(setDecor(design, group, { box: v || '' }))"
      />
      <v-checkbox-btn
        v-if="d.box"
        data-test="decor-rounded"
        density="compact"
        label="Round"
        :model-value="Boolean(d.rounded)"
        @update:model-value="v => update(setDecor(design, group, { rounded: v }))"
      />
      <SizeStepper
        v-if="d.above || (d.below && d.below !== 'none') || d.box"
        data-key="thickness"
        label="Thick"
        :max="4"
        :min="1"
        :signed="false"
        :value="d.thickness || 2"
        @step="n => update(setDecor(design, group, { thickness: Math.min(4, Math.max(1, (d.thickness || 2) + n)) }))"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import ReceiptFontSelect from '@/components/Receipt/ReceiptFontSelect.vue'
  import SizeStepper from '@/components/Receipt/SizeStepper.vue'
  import {
    footerOrder, groupBoldState, groupInfo, groupSizeStep, isGroupChanged, isSectionHidden, lineStyleOptions,
    moveSection, resetGroup, setDecor, setGroup, setGroupBold, toggleSection,
  } from '@/utils/receiptDesign'

  const props = defineProps<{
    catalog: Record<string, any>
    design: Record<string, any>
    group: string
  }>()
  const emit = defineEmits<{ 'update:design': [design: Record<string, any>], close: [] }>()

  const info = computed(() => groupInfo(props.catalog, props.group) ?? { key: props.group, label: props.group })
  const g = computed(() => props.design.groups?.[props.group] ?? {})
  const d = computed(() => props.design.decor?.[props.group] ?? {})
  const hidden = computed(() => isSectionHidden(props.design, info.value.section))
  const movable = computed(() => Boolean(info.value.section) && (props.catalog.footer_order ?? []).includes(info.value.section))
  const belowOptions = [{ value: 'none', title: 'None' }, ...lineStyleOptions]

  function canMove (step: number) {
    const order = footerOrder(props.design, props.catalog)
    const i = order.indexOf(info.value.section)
    return i !== -1 && i + step >= 0 && i + step < order.length
  }
  function update (design: Record<string, any>) {
    emit('update:design', design)
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
  gap: 2px;
  margin-bottom: 4px;
}
.rtools-title {
  flex: 1 1 auto;
  font-size: 0.85rem;
  font-weight: 600;
}
.rtools-label {
  margin: 8px 0 4px;
  font-size: 0.75rem;
  font-weight: 600;
  color: rgba(var(--v-theme-on-surface), 0.6);
}
.rtools-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}
.rtools-select {
  flex: 1 1 120px;
  min-width: 110px;
}
</style>
