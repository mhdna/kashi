<!--
  The panel for one part of the receipt: the whole part at once (capitals,
  size, weight, alignment, its lines and box, its place), then each of its
  lines. Words are typed on the receipt itself (ReceiptLivePaper). Nothing
  picked in a toggle is the look's own; Default puts a line or the part back.
-->
<template>
  <div class="rpp" data-no-discard-guard>
    <template v-if="info">
      <div class="d-flex align-center ga-1 mb-3">
        <div class="text-subtitle-1 font-weight-medium flex-grow-1" data-test="panel-title">{{ info.label }}</div>
        <template v-if="movable">
          <v-btn
            aria-label="Move up"
            data-test="move-up"
            density="comfortable"
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
            density="comfortable"
            :disabled="!canMove(1)"
            icon="mdi-arrow-down"
            size="small"
            title="Move down (Alt+Down)"
            variant="text"
            @click="update(moveSection(design, catalog, info.section, 1))"
          />
        </template>
        <v-btn
          v-if="changed"
          data-test="reset-part"
          density="comfortable"
          prepend-icon="mdi-restore"
          size="small"
          text="Default"
          variant="text"
          @click="update(resetGroup(design, catalog, group))"
        />
      </div>

      <v-switch
        v-if="info.section"
        class="mb-2"
        color="primary"
        data-test="part-printed"
        density="compact"
        hide-details
        inset
        label="Print this part"
        :model-value="!isSectionHidden(design, info.section)"
        @update:model-value="update(toggleSection(design, info.section))"
      />

      <v-select
        v-if="group === 'meta'"
        class="mb-4"
        data-test="date-format"
        density="compact"
        hide-details
        item-title="label"
        item-value="key"
        :items="catalog.date_formats"
        label="Date format"
        :model-value="design.date_format || ''"
        variant="outlined"
        @update:model-value="v => update({ ...design, date_format: v || '' })"
      />

      <section class="rpp-block" data-test="whole-part">
        <div class="rpp-heading">Whole part</div>
        <div class="rpp-controls">
          <v-select
            class="rpp-case"
            data-test="group-case"
            density="compact"
            hide-details
            :items="caseOptions"
            label="Capitals"
            :model-value="groupSettings.case || ''"
            variant="outlined"
            @update:model-value="v => update(setGroup(design, group, { case: v }))"
          />
          <SizeStepper
            data-key="group"
            :max="catalog.size_max"
            :min="catalog.size_min"
            :value="groupSettings.size ?? 0"
            @step="n => update(groupSizeStep(design, catalog, group, n))"
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
            <v-btn size="small" value="bold"><strong>Bold</strong></v-btn>
            <v-btn size="small" value="regular">Regular</v-btn>
          </v-btn-toggle>
          <v-btn-toggle
            color="primary"
            data-test="group-align"
            density="compact"
            divided
            :model-value="groupSettings.align || undefined"
            variant="outlined"
            @update:model-value="v => update(setGroup(design, group, { align: v || '' }))"
          >
            <v-btn v-for="a in alignOptions" :key="a.value" size="small" :value="a.value">{{ a.title }}</v-btn>
          </v-btn-toggle>
        </div>
      </section>

      <section class="rpp-block" data-test="part-lines">
        <div class="rpp-heading">Lines and box</div>
        <div class="rpp-decor-row">
          <span class="rpp-decor-label">Line above</span>
          <v-btn-toggle
            color="primary"
            data-test="decor-above"
            density="compact"
            divided
            :model-value="decor.above || undefined"
            variant="outlined"
            @update:model-value="v => update(setDecor(design, group, { above: v || '' }))"
          >
            <v-btn v-for="o in lineStyleOptions" :key="o.value" size="small" :value="o.value">{{ o.title }}</v-btn>
          </v-btn-toggle>
        </div>
        <div class="rpp-decor-row">
          <span class="rpp-decor-label">Line under</span>
          <v-btn-toggle
            color="primary"
            data-test="decor-below"
            density="compact"
            divided
            :model-value="decor.below || undefined"
            variant="outlined"
            @update:model-value="v => update(setDecor(design, group, { below: v || '' }))"
          >
            <v-btn size="small" value="none">None</v-btn>
            <v-btn v-for="o in lineStyleOptions" :key="o.value" size="small" :value="o.value">{{ o.title }}</v-btn>
          </v-btn-toggle>
        </div>
        <div class="rpp-decor-row">
          <span class="rpp-decor-label">Box</span>
          <v-btn-toggle
            color="primary"
            data-test="decor-box"
            density="compact"
            divided
            :model-value="decor.box || undefined"
            variant="outlined"
            @update:model-value="v => update(setDecor(design, group, { box: v || '' }))"
          >
            <v-btn v-for="o in lineStyleOptions" :key="o.value" size="small" :value="o.value">{{ o.title }}</v-btn>
          </v-btn-toggle>
          <v-checkbox-btn
            v-if="decor.box"
            data-test="decor-rounded"
            density="compact"
            label="Round corners"
            :model-value="Boolean(decor.rounded)"
            @update:model-value="v => update(setDecor(design, group, { rounded: v }))"
          />
        </div>
        <div v-if="decor.above || (decor.below && decor.below !== 'none') || decor.box" class="rpp-decor-row">
          <SizeStepper
            data-key="thickness"
            label="Thickness"
            :max="4"
            :min="1"
            :signed="false"
            :value="decor.thickness || 2"
            @step="n => update(setDecor(design, group, { thickness: Math.min(4, Math.max(1, (decor.thickness || 2) + n)) }))"
          />
        </div>
        <div class="text-caption text-medium-emphasis">Line under replaces the look's own divider; None takes it away.</div>
      </section>

      <div
        v-for="e in lines"
        :key="e.key"
        class="rpp-line"
        :class="{ 'rpp-line--off': isSectionHidden(design, info.section) }"
        :data-test="`line-${e.key}`"
      >
        <div class="d-flex align-center ga-2 mb-1">
          <div class="text-body-2 font-weight-medium flex-grow-1">{{ e.label }}</div>
          <v-btn
            v-if="isElementChanged(design, e.key)"
            :data-test="`default-${e.key}`"
            density="comfortable"
            prepend-icon="mdi-restore"
            size="x-small"
            text="Default"
            variant="text"
            @click="update(resetElement(design, e.key))"
          />
          <v-switch
            v-if="can(catalog, e, 'hide')"
            color="primary"
            :data-test="`printed-${e.key}`"
            density="compact"
            hide-details
            inset
            :model-value="!element(e.key).hide"
            @update:model-value="v => set(e.key, { hide: !v })"
          />
        </div>
        <div v-if="can(catalog, e, 'text')" class="text-caption text-medium-emphasis mb-1" :data-test="`words-${e.key}`">
          {{ element(e.key).text ? `"${element(e.key).text}"` : 'Click its words on the receipt to change them.' }}
          <template v-if="hint(e)"> {{ hint(e) }}</template>
        </div>
        <div class="rpp-controls">
          <v-select
            v-if="can(catalog, e, 'case')"
            class="rpp-case"
            :data-test="`case-${e.key}`"
            density="compact"
            hide-details
            :items="caseOptions"
            label="Capitals"
            :model-value="element(e.key).case || ''"
            variant="outlined"
            @update:model-value="v => set(e.key, { case: v })"
          />
          <SizeStepper
            v-if="can(catalog, e, 'size')"
            :data-key="e.key"
            :max="catalog.size_max"
            :min="catalog.size_min"
            :value="element(e.key).size ?? 0"
            @step="n => update(sizeStep(design, catalog, e.key, n))"
          />
          <v-btn-toggle
            v-if="can(catalog, e, 'bold')"
            color="primary"
            :data-test="`bold-${e.key}`"
            density="compact"
            divided
            :model-value="boldState(design, e.key)"
            variant="outlined"
            @update:model-value="v => update(setBold(design, e.key, v))"
          >
            <v-btn size="small" value="bold"><strong>Bold</strong></v-btn>
            <v-btn size="small" value="regular">Regular</v-btn>
          </v-btn-toggle>
          <v-btn-toggle
            v-if="can(catalog, e, 'align')"
            color="primary"
            :data-test="`align-${e.key}`"
            density="compact"
            divided
            :model-value="element(e.key).align || undefined"
            variant="outlined"
            @update:model-value="v => set(e.key, { align: v || '' })"
          >
            <v-btn v-for="a in alignOptions" :key="a.value" size="small" :value="a.value">{{ a.title }}</v-btn>
          </v-btn-toggle>
        </div>
      </div>
    </template>
    <div v-else class="text-body-2 text-medium-emphasis" data-test="panel-empty">
      Click a part of the receipt to change it, or its words to type over them.
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import SizeStepper from '@/components/Receipt/SizeStepper.vue'
  import {
    alignOptions, boldState, can, caseOptions, elementsOf, footerOrder, groupBoldState, groupInfo, groupSizeStep,
    isElementChanged, isGroupChanged, isSectionHidden, lineStyleOptions, moveSection, resetElement, resetGroup,
    setBold, setDecor, setElement, setGroup, setGroupBold, sizeStep, toggleSection,
  } from '@/utils/receiptDesign'

  const props = defineProps({
    catalog: { type: Object, required: true },
    design: { type: Object, required: true },
    group: { type: String, default: '' },
  })
  const emit = defineEmits(['update:design'])

  const info = computed(() => groupInfo(props.catalog, props.group))
  const lines = computed(() => elementsOf(props.catalog, props.group))
  const changed = computed(() => isGroupChanged(props.design, props.catalog, props.group))
  const groupSettings = computed(() => props.design.groups?.[props.group] ?? {})
  const decor = computed(() => props.design.decor?.[props.group] ?? {})
  const movable = computed(() => Boolean(info.value?.section) && (props.catalog.footer_order ?? []).includes(info.value.section))

  function canMove (step: number) {
    const order = footerOrder(props.design, props.catalog)
    const i = order.indexOf(info.value?.section)
    return i !== -1 && i + step >= 0 && i + step < order.length
  }

  function element (key: string) {
    return props.design.elements[key] ?? {}
  }
  function update (design: Record<string, any>) {
    emit('update:design', design)
  }
  function set (key: string, patch: Record<string, any>) {
    update(setElement(props.design, key, patch))
  }
  function hint (e: { default?: string }) {
    if ((e.default ?? '').includes('{percent}')) {return '{percent} prints as the reward.'}
    if ((e.default ?? '').includes('{currency}')) {return '{currency} prints as the currency.'}
    return ''
  }
</script>

<style scoped>
.rpp-block {
  padding: 10px 0 12px;
  border-top: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}
.rpp-heading {
  margin-bottom: 8px;
  font-size: 0.8rem;
  font-weight: 600;
  color: rgba(var(--v-theme-on-surface), 0.6);
}
.rpp-line {
  padding: 12px 0;
  border-top: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}
.rpp-line--off {
  opacity: 0.5;
}
.rpp-controls {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.rpp-case {
  flex: 0 1 170px;
}
.rpp-decor-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.rpp-decor-label {
  min-width: 84px;
  font-size: 0.85rem;
}
</style>
